package service

// featureKeyResponsesImageModelPassthrough 是 accounts.extra 中控制
// "/responses 图片模型是否原样透传上游" 的键名，值类型为 bool。
const featureKeyResponsesImageModelPassthrough = "openai_responses_image_model_passthrough"

// ResponsesImageModelPassthroughOverride 返回账号级的 /responses 图片模型透传开关。
//
// 背景：官方 OpenAI 的 gpt-image-* 不是 Responses 模型，请求必须改写成
// "主文本模型 + image_generation 工具"（见 normalizeOpenAIResponsesImageOnlyModel）。
// 但第三方聚合型上游普遍把 gpt-image-* 直接挂在 /v1/responses 上，改写后送过去的
// 主文本模型它们并不售卖，会返回 404「该模型当前没有已对接的可用供应商」。
//
// 开启本开关后只保留工具注入（上游认 tools[].image_generation.size），
// 不再改写 model，让 gpt-image-* 原样出站。
//
// 返回 nil 表示未配置，按默认行为（改写）处理，保持存量账号不受影响。
func (a *Account) ResponsesImageModelPassthroughOverride() *bool {
	if a == nil || a.Platform != PlatformOpenAI || a.Extra == nil {
		return nil
	}
	if override := boolOverrideFromMap(a.Extra, featureKeyResponsesImageModelPassthrough); override != nil {
		return override
	}
	openaiConfig, _ := a.Extra[PlatformOpenAI].(map[string]any)
	return boolOverrideFromMap(openaiConfig, featureKeyResponsesImageModelPassthrough)
}

// ResponsesImageModelPassthroughEnabled 是 ResponsesImageModelPassthroughOverride
// 的布尔化封装：未配置时返回 false（保持改写这一存量行为）。
func (a *Account) ResponsesImageModelPassthroughEnabled() bool {
	if override := a.ResponsesImageModelPassthroughOverride(); override != nil {
		return *override
	}
	return false
}
