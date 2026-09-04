package service

import (
	"regexp"
	"strings"

	"github.com/tidwall/gjson"
)

const (
	// maxChatImageContentScanBytes 是文本摘要的硬上限，纯保护性质：base64 尾巴已被
	// 流式裁掉，正常响应远不会触到这个值。
	maxChatImageContentScanBytes = 256 << 10

	// maxKeptBase64Chars 是每段内联图片保留的 base64 前缀长度。尺寸信息全在头部，
	// 保留 1000 字符（解码约 750 字节）足够 image.DecodeConfig 判定，其余直接丢弃，
	// 避免把数 MB 的整张图留在内存里。
	maxKeptBase64Chars = 1000
)

var (
	// chatImageMarkdownPattern 匹配 markdown 图片语法，捕获目标 URL 或 data URI。
	chatImageMarkdownPattern = regexp.MustCompile(`!\[[^\]]*\]\(\s*([^)\s]+)`)
	// chatImageDataURIPattern 匹配内联 base64 图片，捕获紧随其后的一段 base64
	// 头部（够 image.DecodeConfig 判定尺寸即可）。1000 是 Go regexp 单次重复上限，
	// 解码后约 750 字节，远超各格式尺寸头所需。
	chatImageDataURIPattern = regexp.MustCompile(`data:image/[A-Za-z0-9.+-]+;base64,([A-Za-z0-9+/=]{1,1000})`)
)

// openAIChatImageScanner 从 Chat Completions 响应文本中识别生成图片，为 CC 直转
// 路径补齐图片计费所需的张数与尺寸（result.ImageCount / ImageSize）。
//
// 存在的原因：CC 直转不做协议转换，上游返回的图片藏在 message.content 的 markdown
// 里（或非标准的 image_urls 数组），既不是 Images API 的 data[]，也不是 Responses
// 的 output[]，openAIImageOutputCounter 原有的两种识别路径都覆盖不到。
type openAIChatImageScanner struct {
	// digest 是裁掉长 base64 尾巴后的文本摘要：标记（markdown 图片语法、data URI
	// 前缀）全部保留，因此张数统计不受图片体积影响，尺寸也仍能从保留的头部解出。
	digest     strings.Builder
	inBase64   bool
	base64Kept int
	urlCount   int
	urlSeen    map[string]struct{}
}

func newOpenAIChatImageScanner() *openAIChatImageScanner {
	return &openAIChatImageScanner{urlSeen: make(map[string]struct{})}
}

// FeedText 累积助手输出文本（流式 delta 或非流式完整内容），过程中裁掉内联图片的
// base64 尾巴。跨 chunk 的 base64 run 状态由 inBase64 / base64Kept 维持，因此
// data URI 被切成多个 delta 也能正确还原成一段。
func (s *openAIChatImageScanner) FeedText(text string) {
	if s == nil || text == "" || s.digest.Len() >= maxChatImageContentScanBytes {
		return
	}
	for i := 0; i < len(text); i++ {
		ch := text[i]
		if s.inBase64 {
			if isBase64Char(ch) {
				if s.base64Kept < maxKeptBase64Chars {
					s.digest.WriteByte(ch)
					s.base64Kept++
				}
				continue
			}
			s.inBase64 = false
		}
		s.digest.WriteByte(ch)
		if s.digest.Len() >= maxChatImageContentScanBytes {
			return
		}
		if !s.inBase64 && strings.HasSuffix(s.digest.String(), "base64,") {
			s.inBase64 = true
			s.base64Kept = 0
		}
	}
}

func isBase64Char(ch byte) bool {
	switch {
	case ch >= 'A' && ch <= 'Z', ch >= 'a' && ch <= 'z', ch >= '0' && ch <= '9':
		return true
	case ch == '+', ch == '/', ch == '=':
		return true
	default:
		return false
	}
}

// FeedImageURLs 记录上游非标准的 image_urls 数组，作为张数的可靠来源。
func (s *openAIChatImageScanner) FeedImageURLs(value gjson.Result) {
	if s == nil || !value.IsArray() {
		return
	}
	value.ForEach(func(_, item gjson.Result) bool {
		url := strings.TrimSpace(item.String())
		if url == "" {
			if nested := strings.TrimSpace(item.Get("url").String()); nested != "" {
				url = nested
			} else if nested := strings.TrimSpace(item.Get("image_url.url").String()); nested != "" {
				url = nested
			}
		}
		if url == "" {
			return true
		}
		if _, ok := s.urlSeen[url]; ok {
			return true
		}
		s.urlSeen[url] = struct{}{}
		s.urlCount++
		return true
	})
}

// Count 返回识别到的图片张数：文本里去重后的图片引用数与上游显式给出的
// image_urls 数量取大者。base64 尾巴在累积时就被裁掉，标记本身不会因图片体积
// 而丢失，所以这里不需要"截断兜底"——避免把超长纯文本误判成图片。
func (s *openAIChatImageScanner) Count() int {
	if s == nil {
		return 0
	}
	count := len(s.imageTargets())
	if s.urlCount > count {
		count = s.urlCount
	}
	return count
}

// Sizes 返回可测量的图片尺寸（形如 "1024x1024"）。只有内联 data URI 能就地测量；
// 外链图片测不了，返回空表示交由 NormalizeImageBillingTierOrDefault 取默认档。
func (s *openAIChatImageScanner) Sizes() []string {
	if s == nil {
		return nil
	}
	matches := chatImageDataURIPattern.FindAllStringSubmatch(s.digest.String(), -1)
	if len(matches) == 0 {
		return nil
	}
	sizes := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		size := detectOpenAIImageResultSize(match[1])
		if size == "" {
			continue
		}
		if _, ok := seen[size]; ok {
			continue
		}
		seen[size] = struct{}{}
		sizes = append(sizes, size)
	}
	if len(sizes) == 0 {
		return nil
	}
	return sizes
}

// imageTargets 汇总文本中出现的图片引用并去重。markdown 里包裹的 data URI 只算一次。
func (s *openAIChatImageScanner) imageTargets() []string {
	content := s.digest.String()
	if content == "" {
		return nil
	}
	targets := make([]string, 0, 2)
	seen := make(map[string]struct{}, 2)
	addTarget := func(target string) {
		target = strings.TrimSpace(target)
		if target == "" {
			return
		}
		// data URI 用头部片段作为身份，避免整段 base64 参与比较。
		if len(target) > 128 {
			target = target[:128]
		}
		if _, ok := seen[target]; ok {
			return
		}
		seen[target] = struct{}{}
		targets = append(targets, target)
	}
	for _, match := range chatImageMarkdownPattern.FindAllStringSubmatch(content, -1) {
		addTarget(match[1])
	}
	for _, match := range chatImageDataURIPattern.FindAllStringSubmatch(content, -1) {
		addTarget(match[0])
	}
	return targets
}

// AddJSON 从非流式 Chat Completions 响应中识别生成图片。
func (s *openAIChatImageScanner) AddJSON(body []byte) {
	if s == nil || len(body) == 0 || !gjson.ValidBytes(body) {
		return
	}
	root := gjson.ParseBytes(body)
	s.FeedImageURLs(root.Get("image_urls"))
	root.Get("choices").ForEach(func(_, choice gjson.Result) bool {
		message := choice.Get("message")
		s.FeedText(message.Get("content").String())
		s.FeedImageURLs(message.Get("image_urls"))
		s.FeedImageURLs(message.Get("images"))
		return true
	})
}

// AddSSEData 从流式 Chat Completions chunk 中累积图片内容。
func (s *openAIChatImageScanner) AddSSEData(data []byte) {
	if s == nil || len(data) == 0 || strings.TrimSpace(string(data)) == "[DONE]" || !gjson.ValidBytes(data) {
		return
	}
	root := gjson.ParseBytes(data)
	s.FeedImageURLs(root.Get("image_urls"))
	root.Get("choices").ForEach(func(_, choice gjson.Result) bool {
		delta := choice.Get("delta")
		s.FeedText(delta.Get("content").String())
		s.FeedImageURLs(delta.Get("image_urls"))
		s.FeedImageURLs(delta.Get("images"))
		// 少数上游把整段内容放在 message 而非 delta 里。
		s.FeedText(choice.Get("message.content").String())
		s.FeedImageURLs(choice.Get("message.image_urls"))
		return true
	})
}

// isRawChatImageModel 判断 CC 直转请求是否涉及图片模型。入站模型、计费模型与上游
// 模型三者任一命中即算，覆盖账号级模型映射改名的情况。
func isRawChatImageModel(models ...string) bool {
	for _, model := range models {
		if IsGPTImageGenerationModel(model) {
			return true
		}
	}
	return false
}

// scanRawChatCompletionsImages 在涉及图片模型时执行一次扫描，返回张数与可测尺寸。
// 非图片模型直接返回零值，不构造扫描器。
func scanRawChatCompletionsImages(originalModel, billingModel, upstreamModel string, feed func(*openAIChatImageScanner)) (int, []string) {
	if !isRawChatImageModel(originalModel, billingModel, upstreamModel) {
		return 0, nil
	}
	scanner := newOpenAIChatImageScanner()
	feed(scanner)
	return scanner.Count(), scanner.Sizes()
}
