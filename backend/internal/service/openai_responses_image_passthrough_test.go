package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResponsesImageModelPassthroughEnabled(t *testing.T) {
	for _, tc := range []struct {
		name    string
		account *Account
		want    bool
	}{
		{
			name:    "nil account defaults to rewrite",
			account: nil,
			want:    false,
		},
		{
			name:    "missing flag defaults to rewrite",
			account: &Account{Platform: PlatformOpenAI, Extra: map[string]any{}},
			want:    false,
		},
		{
			name: "top-level flag enables passthrough",
			account: &Account{Platform: PlatformOpenAI, Extra: map[string]any{
				featureKeyResponsesImageModelPassthrough: true,
			}},
			want: true,
		},
		{
			name: "explicit false keeps rewrite",
			account: &Account{Platform: PlatformOpenAI, Extra: map[string]any{
				featureKeyResponsesImageModelPassthrough: false,
			}},
			want: false,
		},
		{
			name: "nested openai config enables passthrough",
			account: &Account{Platform: PlatformOpenAI, Extra: map[string]any{
				PlatformOpenAI: map[string]any{featureKeyResponsesImageModelPassthrough: true},
			}},
			want: true,
		},
		{
			name: "non-openai platform ignores the flag",
			account: &Account{Platform: PlatformAnthropic, Extra: map[string]any{
				featureKeyResponsesImageModelPassthrough: true,
			}},
			want: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.account.ResponsesImageModelPassthroughEnabled())
		})
	}
}

// 透传模式必须保留工具注入与图片参数迁移（上游认 tools[].image_generation.size），
// 只是不把 model 改写成主文本模型。
func TestNormalizeOpenAIResponsesImageOnlyModel_KeepsImageModelWhenPassthrough(t *testing.T) {
	reqBody := map[string]any{
		"model":  "gpt-image-2",
		"prompt": "draw a cat",
		"size":   "2048x2048",
	}

	modified := normalizeOpenAIResponsesImageOnlyModel(reqBody, true)
	require.True(t, modified)
	require.Equal(t, "gpt-image-2", reqBody["model"], "passthrough must not rewrite the model")
	require.Equal(t, "draw a cat", reqBody["input"])

	tools, ok := reqBody["tools"].([]any)
	require.True(t, ok)
	require.Len(t, tools, 1)
	tool, ok := tools[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "image_generation", tool["type"])
	require.Equal(t, "gpt-image-2", tool["model"])
	require.Equal(t, "2048x2048", tool["size"], "size must migrate into the tool")

	_, hasTopLevelSize := reqBody["size"]
	require.False(t, hasTopLevelSize)
}

func TestNormalizeOpenAIResponsesImageOnlyModel_IgnoresNonImageModel(t *testing.T) {
	reqBody := map[string]any{"model": "gpt-5.6", "input": "hi"}
	require.False(t, normalizeOpenAIResponsesImageOnlyModel(reqBody, true))
	require.Equal(t, "gpt-5.6", reqBody["model"])
}
