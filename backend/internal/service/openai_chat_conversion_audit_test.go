package service

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func openAIChatConversionAuditLog(t *testing.T, fields []zap.Field) (map[string]any, string) {
	t.Helper()
	var output bytes.Buffer
	logger := zap.New(zapcore.NewCore(zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		MessageKey: "message",
	}), zapcore.AddSync(&output), zap.DebugLevel))
	logger.Info("conversion_audit", fields...)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(output.Bytes(), &decoded))
	return decoded, output.String()
}

func TestOpenAIChatConversionAuditSequence(t *testing.T) {
	audit := newOpenAIChatConversionAudit()
	state := apicompat.NewResponsesEventToChatState()
	state.IncludeUsage = true
	sequence := []string{
		`{"type":"response.created","response":{"id":"private-id","model":"private-model"}}`,
		`{"type":"response.output_text.delta","delta":"\u4f60"}`,
		`{"type":"response.reasoning_text.delta","delta":"private-thought"}`,
		`{"type":"response.reasoning_summary_text.done","text":"private-summary"}`,
		`{"type":"response.output_item.added","item":{"type":"custom_tool_call","call_id":"private-call","name":"tool"}}`,
		`{"type":"response.custom_tool_call_input.delta","delta":"{\"x\":1}"}`,
		`{"type":"response.refusal.delta","delta":"private-refusal"}`,
		`{"type":"response.incomplete","response":{"status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},"usage":{"input_tokens":7,"output_tokens":2,"output_tokens_details":{"reasoning_tokens":0}}}}`,
	}
	for _, raw := range sequence {
		var event apicompat.ResponsesStreamEvent
		require.NoError(t, json.Unmarshal([]byte(raw), &event))
		before, err := json.Marshal(event)
		require.NoError(t, err)
		audit.ObserveEvent(&event)
		audit.ObserveRawUsageJSON([]byte(raw))
		for _, chunk := range apicompat.ResponsesEventToChatChunks(&event, state) {
			audit.ObserveChunk(chunk)
		}
		after, err := json.Marshal(event)
		require.NoError(t, err)
		require.Equal(t, before, after, "observation must not mutate input")
	}
	fields, _ := openAIChatConversionAuditLog(t, audit.Fields())
	for name, want := range map[string]float64{
		"responses_events": 8, "responses_terminal_events": 1,
		"responses_text_events": 1, "responses_reasoning_events": 2,
		"responses_refusal_events": 1, "responses_tool_events": 2,
		"responses_input_tokens": 7, "responses_output_tokens": 2, "responses_reasoning_tokens": 0,
		"chat_content_deltas": 2, "chat_content_bytes": 3,
		"chat_reasoning_deltas": 1, "chat_reasoning_bytes": 15,
		"chat_tool_deltas": 2, "chat_tool_bytes": 11,
	} {
		require.Equal(t, want, fields[name], name)
	}
	require.Equal(t, "response.incomplete", fields["responses_terminal_type"])
	require.Equal(t, "incomplete", fields["responses_terminal_status"])
	require.Equal(t, "max_output_tokens", fields["responses_incomplete_reason"])
	require.Equal(t, "length", fields["chat_finish_reason"])
	require.Equal(t, true, fields["responses_usage_presence_exact"])
}

func TestOpenAIChatConversionAuditRawUsagePresence(t *testing.T) {
	for _, tc := range []struct {
		name, raw                string
		present                  bool
		input, output, reasoning any
	}{
		{name: "absent", raw: `{}`},
		{name: "null", raw: `{"usage":null}`},
		{name: "empty", raw: `{"usage":{}}`, present: true},
		{name: "null tokens", raw: `{"usage":{"input_tokens":null,"output_tokens":null,"output_tokens_details":{"reasoning_tokens":null}}}`, present: true},
		{name: "zero", raw: `{"usage":{"input_tokens":0,"output_tokens":0,"output_tokens_details":{"reasoning_tokens":0}}}`, present: true, input: float64(0), output: float64(0), reasoning: float64(0)},
		{name: "missing reasoning", raw: `{"response":{"usage":{"input_tokens":8,"output_tokens":3,"output_tokens_details":{}}}}`, present: true, input: float64(8), output: float64(3)},
		{name: "details missing", raw: `{"usage":{"input_tokens":8}}`, present: true, input: float64(8)},
		{name: "nested priority", raw: `{"usage":{"input_tokens":99},"response":{"usage":{"output_tokens":0}}}`, present: true, output: float64(0)},
		{name: "no alias defaults", raw: `{"usage":{"prompt_tokens":9,"completion_tokens":4,"completion_tokens_details":{"reasoning_tokens":6}}}`, present: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			audit := newOpenAIChatConversionAudit()
			var event apicompat.ResponsesStreamEvent
			require.NoError(t, json.Unmarshal([]byte(tc.raw), &event))
			audit.ObserveEvent(&event)
			audit.ObserveRawUsageJSON([]byte(tc.raw))
			fields, _ := openAIChatConversionAuditLog(t, audit.Fields())
			require.Equal(t, tc.present, fields["responses_usage_present"])
			require.Equal(t, tc.present, fields["responses_usage_presence_exact"])
			require.Equal(t, tc.input, fields["responses_input_tokens"])
			require.Equal(t, tc.output, fields["responses_output_tokens"])
			require.Equal(t, tc.reasoning, fields["responses_reasoning_tokens"])
		})
	}
}

func TestOpenAIChatConversionAuditUsageSnapshots(t *testing.T) {
	audit := newOpenAIChatConversionAudit()
	event := &apicompat.ResponsesStreamEvent{Type: "response.completed", Response: &apicompat.ResponsesResponse{
		Status: "completed", Usage: &apicompat.ResponsesUsage{InputTokens: 12, OutputTokens: 4,
			OutputTokensDetails: &apicompat.ResponsesOutputTokensDetails{ReasoningTokens: 2}},
	}}
	audit.ObserveEvent(event)
	event.Response.Usage.InputTokens = 999
	event.Response.Usage.OutputTokensDetails.ReasoningTokens = 999
	fields, _ := openAIChatConversionAuditLog(t, audit.Fields())
	require.Equal(t, false, fields["responses_usage_presence_exact"])
	require.Equal(t, float64(12), fields["responses_input_tokens"])
	require.Equal(t, float64(2), fields["responses_reasoning_tokens"])

	audit.ObserveRawUsageJSON([]byte(`{"usage":{"input_tokens":0,"output_tokens":5}}`))
	snapshot := audit.Fields()
	audit.ObserveRawUsageJSON([]byte(`{"usage":{"input_tokens":3}}`))
	audit.ObserveEvent(&apicompat.ResponsesStreamEvent{Type: "response.done"})
	fields, _ = openAIChatConversionAuditLog(t, audit.Fields())
	require.Equal(t, float64(3), fields["responses_input_tokens"], "latest snapshot, never sum")
	require.Nil(t, fields["responses_output_tokens"], "new partial usage must not inherit old fields")
	require.Nil(t, fields["responses_reasoning_tokens"])
	require.Equal(t, "response.done", fields["responses_terminal_type"])
	require.Equal(t, "", fields["responses_terminal_status"])
	require.Equal(t, float64(2), fields["responses_terminal_events"])
	old, _ := openAIChatConversionAuditLog(t, snapshot)
	require.Equal(t, float64(0), old["responses_input_tokens"])
	require.Equal(t, float64(5), old["responses_output_tokens"])
}

func TestOpenAIChatConversionAuditBufferedAndEmptyDeltas(t *testing.T) {
	audit := newOpenAIChatConversionAudit()
	empty, text := "", "abc"
	audit.ObserveChunk(apicompat.ChatCompletionsChunk{Choices: []apicompat.ChatChunkChoice{
		{Delta: apicompat.ChatDelta{Role: "assistant"}},
		{Delta: apicompat.ChatDelta{Content: &empty, ReasoningContent: &empty}},
		{Delta: apicompat.ChatDelta{Content: &text}},
	}})
	audit.ObserveResponse(&apicompat.ChatCompletionsResponse{Choices: []apicompat.ChatChoice{
		{Message: apicompat.ChatMessage{Content: json.RawMessage(`"\u4f60"`), ReasoningContent: "think",
			ToolCalls: []apicompat.ChatToolCall{{Function: apicompat.ChatFunctionCall{Name: "f", Arguments: "{}"}}}}, FinishReason: "tool_calls"},
		{Message: apicompat.ChatMessage{Content: json.RawMessage(`[{"type":"text","text":"ok"},{"type":"image_url","text":"ignored"}]`)}, FinishReason: "stop"},
		{Message: apicompat.ChatMessage{Content: json.RawMessage(`null`)}, FinishReason: "stop"},
		{Message: apicompat.ChatMessage{Content: json.RawMessage(`invalid-json`)}, FinishReason: "stop"},
	}})
	fields, _ := openAIChatConversionAuditLog(t, audit.Fields())
	require.Equal(t, float64(4), fields["chat_content_deltas"])
	require.Equal(t, float64(8), fields["chat_content_bytes"])
	require.Equal(t, float64(2), fields["chat_reasoning_deltas"])
	require.Equal(t, float64(5), fields["chat_reasoning_bytes"])
	require.Equal(t, float64(1), fields["chat_tool_deltas"])
	require.Equal(t, float64(3), fields["chat_tool_bytes"])
	require.Equal(t, "stop", fields["chat_finish_reason"])
}

func TestOpenAIChatConversionAuditKnownEnums(t *testing.T) {
	for _, terminal := range []string{"response.completed", "response.done", "response.incomplete", "response.failed"} {
		for _, status := range []string{"completed", "incomplete", "failed", "in_progress", "queued", "cancelled"} {
			audit := newOpenAIChatConversionAudit()
			audit.ObserveEvent(&apicompat.ResponsesStreamEvent{Type: terminal, Response: &apicompat.ResponsesResponse{Status: status}})
			fields, _ := openAIChatConversionAuditLog(t, audit.Fields())
			require.Equal(t, terminal, fields["responses_terminal_type"])
			require.Equal(t, status, fields["responses_terminal_status"])
		}
	}
	for _, reason := range []string{"max_output_tokens", "content_filter"} {
		audit := newOpenAIChatConversionAudit()
		audit.ObserveEvent(&apicompat.ResponsesStreamEvent{Type: "response.incomplete", Response: &apicompat.ResponsesResponse{
			IncompleteDetails: &apicompat.ResponsesIncompleteDetails{Reason: reason},
		}})
		fields, _ := openAIChatConversionAuditLog(t, audit.Fields())
		require.Equal(t, reason, fields["responses_incomplete_reason"])
	}
	for _, finish := range []string{"stop", "length", "tool_calls", "content_filter", "function_call"} {
		audit := newOpenAIChatConversionAudit()
		audit.ObserveChunk(apicompat.ChatCompletionsChunk{Choices: []apicompat.ChatChunkChoice{{FinishReason: &finish}}})
		fields, _ := openAIChatConversionAuditLog(t, audit.Fields())
		require.Equal(t, finish, fields["chat_finish_reason"])
	}
}

func TestOpenAIChatConversionAuditTypedLifecycleEvents(t *testing.T) {
	audit := newOpenAIChatConversionAudit()
	for _, event := range []apicompat.ResponsesStreamEvent{
		{Type: "response.content_part.added", Part: &apicompat.ResponsesContentPart{Type: "output_text"}},
		{Type: "response.content_part.done", Part: &apicompat.ResponsesContentPart{Type: "refusal"}},
		{Type: "response.output_item.done", Item: &apicompat.ResponsesOutput{Type: "reasoning"}},
		{Type: "response.output_item.added", Item: &apicompat.ResponsesOutput{Type: "function_call"}},
		{Type: "response.function_call_arguments.done"},
		{Type: "response.reasoning_summary_part.done"},
		{Type: "response.output_item.done", Item: &apicompat.ResponsesOutput{Type: "unknown-private-tool"}},
		{Type: "response.unknown-private-event"},
		{Type: "response.content_part.added"},
		{Type: "response.output_item.added"},
	} {
		audit.ObserveEvent(&event)
	}
	fields, log := openAIChatConversionAuditLog(t, audit.Fields())
	require.Equal(t, float64(1), fields["responses_text_events"])
	require.Equal(t, float64(1), fields["responses_refusal_events"])
	require.Equal(t, float64(2), fields["responses_reasoning_events"])
	require.Equal(t, float64(2), fields["responses_tool_events"])
	require.NotContains(t, log, "private")
}

func TestOpenAIChatConversionAuditPrivacy(t *testing.T) {
	const secret = "DO-NOT-RETAIN-private-prompt-thought-argument"
	audit := newOpenAIChatConversionAudit()
	audit.ObserveEvent(&apicompat.ResponsesStreamEvent{Type: "response.failed", Delta: secret, Text: secret,
		Name: secret, Arguments: secret, Input: secret, ItemID: secret, CallID: secret, Code: secret, Param: secret,
		Response: &apicompat.ResponsesResponse{ID: secret, Model: secret, Status: secret,
			Error:             &apicompat.ResponsesError{Code: secret, Message: secret},
			IncompleteDetails: &apicompat.ResponsesIncompleteDetails{Reason: secret},
			Output:            []apicompat.ResponsesOutput{{EncryptedContent: secret, Arguments: secret}},
		},
	})
	audit.ObserveEvent(&apicompat.ResponsesStreamEvent{Type: secret})
	audit.ObserveRawUsageJSON([]byte(`{"instructions":"` + secret + `","input":"` + secret + `","usage":{"input_tokens":1,"untrusted":"` + secret + `"}}`))
	for _, malformed := range []string{
		`{"usage":{"input_tokens":"` + secret + `"}}`,
		`{"usage":{"output_tokens_details":{"reasoning_tokens":"` + secret + `"}}}`,
		`{"usage":"` + secret + `"}`, secret,
	} {
		audit.ObserveRawUsageJSON([]byte(malformed))
	}
	audit.ObserveChunk(apicompat.ChatCompletionsChunk{ID: secret, Model: secret, Choices: []apicompat.ChatChunkChoice{{
		Delta: apicompat.ChatDelta{Content: ptrForOpenAIChatConversionAudit(secret), ReasoningContent: ptrForOpenAIChatConversionAudit(secret),
			ToolCalls: []apicompat.ChatToolCall{{ID: secret, Type: secret, Function: apicompat.ChatFunctionCall{Name: secret, Arguments: secret}}}},
		FinishReason: ptrForOpenAIChatConversionAudit(secret),
	}}})
	fields, log := openAIChatConversionAuditLog(t, audit.Fields())
	require.NotContains(t, log, secret)
	require.Equal(t, "unknown", fields["responses_terminal_status"])
	require.Equal(t, "unknown", fields["responses_incomplete_reason"])
	require.Equal(t, "unknown", fields["chat_finish_reason"])
	require.Equal(t, float64(1), fields["responses_input_tokens"])
	require.Equal(t, true, fields["responses_raw_usage_parse_failed"])

	// Inspect retained state as well as log output: a redacted logger must not
	// conceal an event/body reference retained by the collector itself.
	var check func(reflect.Value)
	check = func(value reflect.Value) {
		switch value.Kind() {
		case reflect.Pointer:
			if !value.IsNil() {
				check(value.Elem())
			}
		case reflect.Struct:
			for i := 0; i < value.NumField(); i++ {
				check(value.Field(i))
			}
		case reflect.String:
			require.Contains(t, []string{"", "unknown", "response.failed"}, value.String())
		case reflect.Bool, reflect.Int64:
		default:
			t.Fatalf("unexpected retained state type: %s", value.Type())
		}
	}
	check(reflect.ValueOf(audit))
}

func ptrForOpenAIChatConversionAudit(value string) *string { return &value }

func TestOpenAIChatConversionAuditNilAndZeroValue(t *testing.T) {
	var audit *openAIChatConversionAudit
	audit.ObserveEvent(nil)
	audit.ObserveChunk(apicompat.ChatCompletionsChunk{})
	audit.ObserveResponse(nil)
	audit.ObserveRawUsageJSON(nil)
	require.Nil(t, audit.Fields())
	audit = &openAIChatConversionAudit{}
	audit.ObserveEvent(nil)
	audit.ObserveResponse(nil)
	audit.ObserveRawUsageJSON(nil)
	fields, _ := openAIChatConversionAuditLog(t, audit.Fields())
	require.Equal(t, float64(0), fields["responses_events"])
	require.Nil(t, fields["responses_input_tokens"])
	require.Nil(t, fields["responses_reasoning_tokens"])
	require.Equal(t, "", fields["chat_finish_reason"])
}
