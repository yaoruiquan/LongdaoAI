package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// tinyPNGBase64 是一张 7x3 的 PNG，用于验证尺寸能从 base64 头部解出。
const tinyPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAcAAAADCAIAAADQoYKSAAAAEklEQVR4nGP4z8CAibAI4RQFAMeWFOx1QjWwAAAAAElFTkSuQmCC"

func TestOpenAIChatImageScanner_InlineDataURI(t *testing.T) {
	scanner := newOpenAIChatImageScanner()
	scanner.AddJSON([]byte(`{"choices":[{"message":{"role":"assistant","content":"![image](data:image/png;base64,` + tinyPNGBase64 + `)"}}]}`))

	require.Equal(t, 1, scanner.Count())
	require.Equal(t, []string{"7x3"}, scanner.Sizes())
}

func TestOpenAIChatImageScanner_MarkdownExternalURL(t *testing.T) {
	scanner := newOpenAIChatImageScanner()
	scanner.AddJSON([]byte(`{"choices":[{"message":{"content":"![generated image 1](https://ig.example/media/abc.png)"}}]}`))

	require.Equal(t, 1, scanner.Count())
	require.Nil(t, scanner.Sizes(), "外链图片测不了尺寸，交给默认档")
}

func TestOpenAIChatImageScanner_ImageURLsArray(t *testing.T) {
	scanner := newOpenAIChatImageScanner()
	scanner.AddJSON([]byte(`{"image_urls":["https://ig.example/a.png","https://ig.example/b.png"],"choices":[{"message":{"content":""}}]}`))

	require.Equal(t, 2, scanner.Count())
}

func TestOpenAIChatImageScanner_DeduplicatesMarkdownWrappedDataURI(t *testing.T) {
	scanner := newOpenAIChatImageScanner()
	content := "![image](data:image/png;base64," + tinyPNGBase64 + ")"
	scanner.AddJSON([]byte(`{"choices":[{"message":{"content":` + jsonQuote(content) + `}}]}`))

	require.Equal(t, 1, scanner.Count(), "markdown 包裹的 data URI 只能算一张")
}

// 流式 delta 会把 data URI 切成任意长度的碎片，跨 chunk 的 base64 run 状态必须能续上。
func TestOpenAIChatImageScanner_StreamingSplitDataURI(t *testing.T) {
	scanner := newOpenAIChatImageScanner()
	content := "![image](data:image/png;base64," + tinyPNGBase64 + ")"
	for _, piece := range splitEvery(content, 7) {
		scanner.AddSSEData([]byte(`{"choices":[{"delta":{"content":` + jsonQuote(piece) + `}}]}`))
	}

	require.Equal(t, 1, scanner.Count())
	require.Equal(t, []string{"7x3"}, scanner.Sizes())
}

// 超长纯文本不能被误判成图片：base64 尾巴裁剪只影响尺寸测量，不影响标记计数。
func TestOpenAIChatImageScanner_LongPlainTextIsNotAnImage(t *testing.T) {
	scanner := newOpenAIChatImageScanner()
	scanner.FeedText(strings.Repeat("这是一段很长的普通回复。", 40000))

	require.Equal(t, 0, scanner.Count())
	require.Nil(t, scanner.Sizes())
}

func TestOpenAIChatImageScanner_TrimsLargeBase64(t *testing.T) {
	scanner := newOpenAIChatImageScanner()
	huge := "![image](data:image/png;base64," + tinyPNGBase64 + strings.Repeat("A", 4<<20) + ")"
	scanner.FeedText(huge)

	require.Equal(t, 1, scanner.Count())
	require.Equal(t, []string{"7x3"}, scanner.Sizes())
	require.Less(t, scanner.digest.Len(), 4096, "base64 尾巴必须在累积时就被裁掉")
}

func TestScanRawChatCompletionsImages_SkipsNonImageModels(t *testing.T) {
	called := false
	count, sizes := scanRawChatCompletionsImages("gpt-5.6", "gpt-5.6", "gpt-5.6", func(*openAIChatImageScanner) {
		called = true
	})

	require.False(t, called, "非图片模型不应构造扫描器")
	require.Zero(t, count)
	require.Nil(t, sizes)
}

func TestScanRawChatCompletionsImages_CountsForImageModels(t *testing.T) {
	count, sizes := scanRawChatCompletionsImages("gpt-image-2", "gpt-image-2", "gpt-image-2", func(scanner *openAIChatImageScanner) {
		scanner.AddJSON([]byte(`{"choices":[{"message":{"content":"![image](data:image/png;base64,` + tinyPNGBase64 + `)"}}]}`))
	})

	require.Equal(t, 1, count)
	require.Equal(t, []string{"7x3"}, sizes)
}

func TestIsRawChatImageModel(t *testing.T) {
	require.True(t, isRawChatImageModel("gpt-5.6", "gpt-image-2", ""), "映射后的模型命中也算")
	require.True(t, isRawChatImageModel("gpt-image-2-2k", "", ""))
	require.False(t, isRawChatImageModel("gpt-5.6", "gpt-5.6", "gpt-5.6"))
}

func jsonQuote(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

func splitEvery(s string, n int) []string {
	out := make([]string, 0, len(s)/n+1)
	for len(s) > n {
		out = append(out, s[:n])
		s = s[n:]
	}
	if s != "" {
		out = append(out, s)
	}
	return out
}

// 第三方聚合上游对同一个图片模型可能返回两种形状：官方的 output[].result，
// 或把 markdown 图片塞进 output_text。后者必须也算成一张图，否则会退化成按 token 计费。
func TestOpenAIImageOutputCounter_ResponsesOutputTextMarkdown(t *testing.T) {
	counter := newOpenAIImageOutputCounter()
	counter.AddJSONResponse([]byte(`{"model":"gpt-image-2","status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"![generated](https://pic.example/p/img/abc.png)"}]}]}`))

	require.Equal(t, 1, counter.Count())
}

func TestOpenAIImageOutputCounter_ResponsesOutputTextInlineBase64(t *testing.T) {
	counter := newOpenAIImageOutputCounter()
	counter.AddJSONResponse([]byte(`{"model":"gpt-image-2","output":[{"type":"message","content":[{"type":"output_text","text":"![img](data:image/png;base64,` + tinyPNGBase64 + `)"}]}]}`))

	require.Equal(t, 1, counter.Count())
	require.Equal(t, []string{"7x3"}, counter.Sizes())
}

// 普通文本模型的回答里出现 markdown 图片链接是正常内容，不能当成生成图计费。
func TestOpenAIImageOutputCounter_NonImageModelTextIsNotBilled(t *testing.T) {
	counter := newOpenAIImageOutputCounter()
	counter.AddJSONResponse([]byte(`{"model":"gpt-5.6","output":[{"type":"message","content":[{"type":"output_text","text":"文档里可以这样写：![logo](https://example.com/logo.png)"}]}]}`))

	require.Zero(t, counter.Count())
	require.Nil(t, counter.Sizes())
}

func TestOpenAIImageOutputCounter_StreamingOutputTextDelta(t *testing.T) {
	counter := newOpenAIImageOutputCounter()
	counter.AddSSEData([]byte(`{"type":"response.created","response":{"model":"gpt-image-2","status":"in_progress"}}`))
	content := "![img](data:image/png;base64," + tinyPNGBase64 + ")"
	for _, piece := range splitEvery(content, 9) {
		counter.AddSSEData([]byte(`{"type":"response.output_text.delta","delta":` + jsonQuote(piece) + `}`))
	}

	require.Equal(t, 1, counter.Count())
	require.Equal(t, []string{"7x3"}, counter.Sizes())
}

// 官方 output[].result 形状仍按原逻辑计数，不受文本扫描影响。
func TestOpenAIImageOutputCounter_OfficialResultShapeStillCounted(t *testing.T) {
	counter := newOpenAIImageOutputCounter()
	counter.AddJSONResponse([]byte(`{"model":"gpt-image-2","output":[{"type":"image_generation_call","id":"ig_1","result":"` + tinyPNGBase64 + `"}]}`))

	require.Equal(t, 1, counter.Count())
}
