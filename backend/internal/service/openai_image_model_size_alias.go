package service

import (
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// openAIImageModelSizeAliases 把图片模型名的档位后缀映射成上游出图尺寸。
//
// 上游只售卖不带档位的图片模型（如 gpt-image-2），档位后缀是中转侧的语法糖：
// 第三方客户端（Codex 等）往往只能配一个模型名、没有地方传 size，
// 带上后缀就能选尺寸。三档取值都已实测上游可用。
var openAIImageModelSizeAliases = map[string]string{
	"1k": "1024x1024",
	"2k": "2048x2048",
	"4k": "4096x4096",
}

// ParseOpenAIImageModelSizeAlias 解析带档位后缀的图片模型别名。
//
//	gpt-image-2-2k → ("gpt-image-2", "2048x2048", true)
//	gpt-image-2    → ("gpt-image-2", "", false)
//	gpt-image-1.5  → ("gpt-image-1.5", "", false)
//
// 去掉后缀后必须仍是合法图片模型，否则视为非别名原样返回，避免误伤
// "gpt-image-1k" 这类本身就带 k 结尾的模型名。
func ParseOpenAIImageModelSizeAlias(model string) (string, string, bool) {
	trimmed := strings.TrimSpace(model)
	if !IsGPTImageGenerationModel(trimmed) {
		return trimmed, "", false
	}
	idx := strings.LastIndexByte(trimmed, '-')
	if idx <= 0 {
		return trimmed, "", false
	}
	size, ok := openAIImageModelSizeAliases[strings.ToLower(trimmed[idx+1:])]
	if !ok {
		return trimmed, "", false
	}
	base := trimmed[:idx]
	if !IsGPTImageGenerationModel(base) {
		return trimmed, "", false
	}
	return base, size, true
}

// ExpandOpenAIImageModelSizeAlias 把 JSON 请求体里的图片模型档位别名展开成
// "基础模型 + size"，让下游账号调度、模型白名单、上游转发都只看见基础模型。
//
// 客户端已显式给出非空 size 时以客户端为准，只改写模型名——别名是兜底语法糖，
// 不应覆盖显式参数。返回值 changed 为 false 时 body 未被修改。
func ExpandOpenAIImageModelSizeAlias(body []byte) ([]byte, string, string, bool) {
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return body, "", "", false
	}
	modelResult := gjson.GetBytes(body, "model")
	if modelResult.Type != gjson.String {
		return body, "", "", false
	}
	base, size, ok := ParseOpenAIImageModelSizeAlias(modelResult.String())
	if !ok {
		return body, "", "", false
	}
	updated, err := sjson.SetBytes(body, "model", base)
	if err != nil {
		return body, "", "", false
	}
	appliedSize := ""
	if existing := gjson.GetBytes(updated, "size"); strings.TrimSpace(existing.String()) == "" {
		withSize, sizeErr := sjson.SetBytes(updated, "size", size)
		if sizeErr == nil {
			updated = withSize
			appliedSize = size
		}
	}
	return updated, base, appliedSize, true
}
