package service

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// Codex 强类型反序列化 response.completed 时要求 input_tokens_details.cached_tokens
// 与 output_tokens_details.reasoning_tokens 存在，缺了会把整段流当作中断并重试。
func TestNormalizeResponsesUsageTokenDetails_FillsMissingFields(t *testing.T) {
	// 线上第三方聚合上游在图片请求上的真实形状：只有 image_tokens / text_tokens。
	event := []byte(`{"type":"response.completed","response":{"model":"gpt-image-2","usage":{"input_tokens":4380,"input_tokens_details":{"image_tokens":0,"text_tokens":4380},"output_tokens":2058,"output_tokens_details":{"image_tokens":2058,"text_tokens":0},"total_tokens":6438}}}`)

	out, changed := normalizeResponsesUsageTokenDetails(event)
	require.True(t, changed)
	require.Equal(t, int64(0), gjson.GetBytes(out, "response.usage.input_tokens_details.cached_tokens").Int())
	require.Equal(t, int64(0), gjson.GetBytes(out, "response.usage.output_tokens_details.reasoning_tokens").Int())
	// 原有数值不能被改动（计费依赖它们）
	require.Equal(t, int64(4380), gjson.GetBytes(out, "response.usage.input_tokens").Int())
	require.Equal(t, int64(4380), gjson.GetBytes(out, "response.usage.input_tokens_details.text_tokens").Int())
	require.Equal(t, int64(2058), gjson.GetBytes(out, "response.usage.output_tokens_details.image_tokens").Int())
	require.Equal(t, int64(6438), gjson.GetBytes(out, "response.usage.total_tokens").Int())
}

func TestNormalizeResponsesUsageTokenDetails_PreservesExistingValues(t *testing.T) {
	event := []byte(`{"usage":{"input_tokens":10,"input_tokens_details":{"cached_tokens":7},"output_tokens":3,"output_tokens_details":{"reasoning_tokens":2}}}`)

	out, changed := normalizeResponsesUsageTokenDetails(event)
	require.False(t, changed, "字段齐全时不应改写")
	require.Equal(t, int64(7), gjson.GetBytes(out, "usage.input_tokens_details.cached_tokens").Int())
	require.Equal(t, int64(2), gjson.GetBytes(out, "usage.output_tokens_details.reasoning_tokens").Int())
}

func TestNormalizeResponsesUsageTokenDetails_RootLevelUsage(t *testing.T) {
	body := []byte(`{"object":"response","usage":{"input_tokens":5,"input_tokens_details":{"text_tokens":5},"output_tokens":1}}`)

	out, changed := normalizeResponsesUsageTokenDetails(body)
	require.True(t, changed)
	require.Equal(t, int64(0), gjson.GetBytes(out, "usage.input_tokens_details.cached_tokens").Int())
	// output_tokens_details 本身不存在时不凭空造出来
	require.False(t, gjson.GetBytes(out, "usage.output_tokens_details").Exists())
}

func TestNormalizeResponsesUsageTokenDetails_NoUsageOrInvalid(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
	}{
		{"没有 usage", `{"type":"response.created","response":{"model":"gpt-image-2"}}`},
		{"usage 不是对象", `{"usage":"none"}`},
		{"非法 JSON", `not json`},
		{"空", ``},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, changed := normalizeResponsesUsageTokenDetails([]byte(tc.in))
			require.False(t, changed)
			require.Equal(t, tc.in, string(out))
		})
	}
}
