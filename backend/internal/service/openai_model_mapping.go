package service

import "strings"

// resolveOpenAIForwardModel resolves the account/group mapping result for
// OpenAI-compatible forwarding. Group-level default mapping only applies when
// the account itself did not match any explicit model_mapping rule.
func resolveOpenAIForwardModel(account *Account, requestedModel, defaultMappedModel string) string {
	if account == nil {
		if defaultMappedModel != "" {
			return defaultMappedModel
		}
		return requestedModel
	}

	mappedModel, matched := account.ResolveMappedModel(requestedModel)
	if !matched && defaultMappedModel != "" {
		return defaultMappedModel
	}
	return mappedModel
}

func resolveOpenAIUpstreamModel(model string) string {
	if isBareGPT53CodexSparkModel(model) {
		return "gpt-5.3-codex-spark"
	}
	return normalizeCodexModel(strings.TrimSpace(model))
}

// resolveOpenAIUpstreamModelForAccount 只对 OAuth 账号执行 Codex 模型名规范化。
// OAuth 走 ChatGPT 官方内部 backend，需要 canonical codex 名；
// apikey 账号指向任意 OpenAI 兼容上游（含第三方聚合站），必须保留用户/映射给定的
// 模型原值，否则会把上游真实支持的模型（如 gpt-5.6-sol）错误改写成 gpt-5.1 导致 404。
func resolveOpenAIUpstreamModelForAccount(account *Account, model string) string {
	if account != nil && account.Type == AccountTypeOAuth {
		return resolveOpenAIUpstreamModel(model)
	}
	return strings.TrimSpace(model)
}

func isBareGPT53CodexSparkModel(model string) bool {
	modelID := strings.TrimSpace(model)
	if modelID == "" {
		return false
	}
	if strings.Contains(modelID, "/") {
		parts := strings.Split(modelID, "/")
		modelID = parts[len(parts)-1]
	}
	normalized := strings.ToLower(strings.TrimSpace(modelID))
	return normalized == "gpt-5.3-codex-spark" || normalized == "gpt 5.3 codex spark"
}
