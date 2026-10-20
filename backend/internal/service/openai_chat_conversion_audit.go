package service

import (
	"encoding/json"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"go.uber.org/zap"
)

// openAIChatConversionAudit is request-local, zero-value usable and not concurrency
// safe. It retains only counters, numeric usage and allowlisted enum values, never
// events, responses, identifiers, content, tool arguments or raw JSON.
type openAIChatConversionAudit struct {
	events, terminalEvents                                int64
	textEvents, reasoningEvents, refusalEvents            int64
	toolEvents                                            int64
	terminalType, terminalStatus, incompleteReason        string
	usagePresent, usagePresenceExact, rawUsageParseFailed bool
	usage                                                 openAIChatConversionAuditUsage
	contentDeltas, contentBytes                           int64
	reasoningDeltas, reasoningBytes                       int64
	toolDeltas, toolBytes                                 int64
	finishReason                                          string
}

// Dedicated wire types deliberately exclude arbitrary payloads. Pointers preserve
// absent/null versus explicit zero without ResponsesUsage's defaulting/aliases.
type openAIChatConversionAuditUsage struct {
	InputTokens         *int64 `json:"input_tokens"`
	OutputTokens        *int64 `json:"output_tokens"`
	OutputTokensDetails *struct {
		ReasoningTokens *int64 `json:"reasoning_tokens"`
	} `json:"output_tokens_details"`
}

func newOpenAIChatConversionAudit() *openAIChatConversionAudit {
	return &openAIChatConversionAudit{}
}

// ObserveEvent must run before normalization. Counts include known delta/done and
// typed item/part lifecycle events, not just events yielding converted output.
// Typed usage cannot preserve individual JSON field presence: call
// ObserveRawUsageJSON immediately afterwards when the original JSON is available.
func (a *openAIChatConversionAudit) ObserveEvent(event *apicompat.ResponsesStreamEvent) {
	if a == nil || event == nil {
		return
	}
	a.events++
	switch event.Type {
	case "response.completed", "response.done", "response.incomplete", "response.failed":
		a.terminalEvents++
		a.terminalType = event.Type
		a.terminalStatus, a.incompleteReason = "", ""
		if event.Response != nil {
			a.terminalStatus = openAIChatConversionAuditEnum(event.Response.Status,
				"completed", "incomplete", "failed", "in_progress", "queued", "cancelled")
			if event.Response.IncompleteDetails != nil {
				a.incompleteReason = openAIChatConversionAuditEnum(event.Response.IncompleteDetails.Reason,
					"max_output_tokens", "content_filter")
			}
		}
	case "response.output_text.delta", "response.output_text.done":
		a.textEvents++
	case "response.reasoning_text.delta", "response.reasoning_text.done",
		"response.reasoning_summary_text.delta", "response.reasoning_summary_text.done",
		"response.reasoning_summary_part.added", "response.reasoning_summary_part.done":
		a.reasoningEvents++
	case "response.refusal.delta", "response.refusal.done":
		a.refusalEvents++
	case "response.function_call_arguments.delta", "response.function_call_arguments.done",
		"response.custom_tool_call_input.delta", "response.custom_tool_call_input.done":
		a.toolEvents++
	case "response.output_item.added", "response.output_item.done":
		if event.Item != nil {
			switch event.Item.Type {
			case "reasoning":
				a.reasoningEvents++
			case "function_call", "custom_tool_call", "tool_search_call", "web_search_call",
				"file_search_call", "computer_call", "code_interpreter_call",
				"image_generation_call", "local_shell_call", "shell_call", "apply_patch_call",
				"mcp_call", "mcp_list_tools", "mcp_approval_request":
				a.toolEvents++
			}
		}
	case "response.content_part.added", "response.content_part.done":
		if event.Part != nil {
			switch event.Part.Type {
			case "output_text":
				a.textEvents++
			case "refusal":
				a.refusalEvents++
			}
		}
	}

	usage := event.Usage
	if event.Response != nil && event.Response.Usage != nil {
		usage = event.Response.Usage
	}
	if usage != nil {
		// Snapshot values, not pointers into an event that normalization may mutate.
		input, output := int64(usage.InputTokens), int64(usage.OutputTokens)
		a.usage = openAIChatConversionAuditUsage{InputTokens: &input, OutputTokens: &output}
		if usage.OutputTokensDetails != nil {
			reasoning := int64(usage.OutputTokensDetails.ReasoningTokens)
			a.usage.OutputTokensDetails = &struct {
				ReasoningTokens *int64 `json:"reasoning_tokens"`
			}{ReasoningTokens: &reasoning}
		}
		a.usagePresent, a.usagePresenceExact = true, false
	}
}

// ObserveRawUsageJSON refines the latest usage snapshot using the original JSON,
// after ObserveEvent and before any normalization. It accepts either an SSE event
// payload or a buffered Responses body. Nested response.usage takes precedence.
// Missing usage leaves the last snapshot intact; usage is never summed. Invalid
// JSON sets only a boolean, never logs or retains a decoder error or raw bytes.
func (a *openAIChatConversionAudit) ObserveRawUsageJSON(data []byte) {
	if a == nil || len(data) == 0 {
		return
	}
	var wire struct {
		Usage    *openAIChatConversionAuditUsage `json:"usage"`
		Response *struct {
			Usage *openAIChatConversionAuditUsage `json:"usage"`
		} `json:"response"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		a.rawUsageParseFailed = true
		return
	}
	usage := wire.Usage
	if wire.Response != nil && wire.Response.Usage != nil {
		usage = wire.Response.Usage
	}
	if usage != nil {
		a.usage = *usage
		a.usagePresent, a.usagePresenceExact = true, true
	}
}

// ObserveChunk records actual emitted choices, including empty-but-present text
// deltas. Each tool call entry counts as one tool delta; its bytes are the UTF-8
// lengths of name plus arguments, excluding IDs, type and JSON/SSE framing.
func (a *openAIChatConversionAudit) ObserveChunk(chunk apicompat.ChatCompletionsChunk) {
	if a == nil {
		return
	}
	for _, choice := range chunk.Choices {
		if choice.Delta.Content != nil {
			a.contentDeltas++
			a.contentBytes += int64(len(*choice.Delta.Content))
		}
		if choice.Delta.ReasoningContent != nil {
			a.reasoningDeltas++
			a.reasoningBytes += int64(len(*choice.Delta.ReasoningContent))
		}
		a.observeTools(choice.Delta.ToolCalls)
		if choice.FinishReason != nil {
			a.observeFinishReason(*choice.FinishReason)
		}
	}
}

// ObserveResponse is the buffered-output counterpart. Each present string content
// (or text part), nonempty reasoning field and tool call counts as one logical
// delta. Use either this method or ObserveChunk for the same output, not both.
func (a *openAIChatConversionAudit) ObserveResponse(response *apicompat.ChatCompletionsResponse) {
	if a == nil || response == nil {
		return
	}
	for _, choice := range response.Choices {
		var text *string
		if err := json.Unmarshal(choice.Message.Content, &text); err == nil {
			if text != nil {
				a.contentDeltas++
				a.contentBytes += int64(len(*text))
			}
		} else {
			var parts []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}
			if json.Unmarshal(choice.Message.Content, &parts) == nil {
				for _, part := range parts {
					if part.Type == "text" {
						a.contentDeltas++
						a.contentBytes += int64(len(part.Text))
					}
				}
			}
		}
		if choice.Message.ReasoningContent != "" {
			a.reasoningDeltas++
			a.reasoningBytes += int64(len(choice.Message.ReasoningContent))
		}
		a.observeTools(choice.Message.ToolCalls)
		a.observeFinishReason(choice.FinishReason)
	}
}

func (a *openAIChatConversionAudit) observeTools(tools []apicompat.ChatToolCall) {
	for _, tool := range tools {
		a.toolDeltas++
		a.toolBytes += int64(len(tool.Function.Name)) + int64(len(tool.Function.Arguments))
	}
}

func (a *openAIChatConversionAudit) observeFinishReason(reason string) {
	a.finishReason = openAIChatConversionAuditEnum(reason,
		"stop", "length", "tool_calls", "content_filter", "function_call")
}

func openAIChatConversionAuditEnum(value string, known ...string) string {
	if value == "" {
		return ""
	}
	for _, candidate := range known {
		if value == candidate {
			return candidate
		}
	}
	return "unknown"
}

// Fields is a scalar-only snapshot. Missing usage numbers are logged as null;
// usage_presence_exact distinguishes raw JSON presence from typed fallbacks.
// The caller supplies trusted request/account/model context separately.
func (a *openAIChatConversionAudit) Fields() []zap.Field {
	if a == nil {
		return nil
	}
	var reasoning *int64
	if a.usage.OutputTokensDetails != nil {
		reasoning = a.usage.OutputTokensDetails.ReasoningTokens
	}
	return []zap.Field{
		zap.Int64("responses_events", a.events),
		zap.Int64("responses_terminal_events", a.terminalEvents),
		zap.String("responses_terminal_type", a.terminalType),
		zap.String("responses_terminal_status", a.terminalStatus),
		zap.String("responses_incomplete_reason", a.incompleteReason),
		zap.Bool("responses_usage_present", a.usagePresent),
		zap.Bool("responses_usage_presence_exact", a.usagePresenceExact),
		zap.Bool("responses_raw_usage_parse_failed", a.rawUsageParseFailed),
		zap.Int64p("responses_input_tokens", a.usage.InputTokens),
		zap.Int64p("responses_output_tokens", a.usage.OutputTokens),
		zap.Int64p("responses_reasoning_tokens", reasoning),
		zap.Int64("responses_text_events", a.textEvents),
		zap.Int64("responses_reasoning_events", a.reasoningEvents),
		zap.Int64("responses_refusal_events", a.refusalEvents),
		zap.Int64("responses_tool_events", a.toolEvents),
		zap.Int64("chat_content_deltas", a.contentDeltas),
		zap.Int64("chat_content_bytes", a.contentBytes),
		zap.Int64("chat_reasoning_deltas", a.reasoningDeltas),
		zap.Int64("chat_reasoning_bytes", a.reasoningBytes),
		zap.Int64("chat_tool_deltas", a.toolDeltas),
		zap.Int64("chat_tool_bytes", a.toolBytes),
		zap.String("chat_finish_reason", a.finishReason),
	}
}

func (a *openAIChatConversionAudit) IsSuspicious() bool {
	return a != nil && ((a.reasoningBytes > 0 && a.contentDeltas == 0 && a.toolDeltas == 0) ||
		a.terminalStatus == "incomplete" || a.terminalStatus == "unknown" ||
		a.finishReason == "unknown")
}
