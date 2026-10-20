package apicompat

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResponsesChatTerminalPreservesIncompleteEvent(t *testing.T) {
	for _, status := range []string{"", "incomplete"} {
		for _, reason := range []string{"max_output_tokens", "content_filter"} {
			t.Run(status+"/"+reason, func(t *testing.T) {
				state := NewResponsesEventToChatState()
				state.SawToolCall = true
				state.IncludeUsage = true
				event := &ResponsesStreamEvent{Type: "response.incomplete", Response: &ResponsesResponse{
					Status: status, IncompleteDetails: &ResponsesIncompleteDetails{Reason: reason},
					Usage: &ResponsesUsage{OutputTokens: 4096},
				}}
				chunks := ResponsesEventToChatChunks(event, state)
				require.Len(t, chunks, 2)
				expected := "length"
				if reason == "content_filter" {
					expected = "content_filter"
				}
				require.Equal(t, expected, *chunks[0].Choices[0].FinishReason)
				require.Equal(t, 4096, chunks[1].Usage.CompletionTokens)
				require.Equal(t, status, event.Response.Status, "conversion must not overwrite raw evidence")
			})
		}
	}
}

func TestResponsesChatTerminalPreservesIncompleteStatusInCompletedEnvelope(t *testing.T) {
	for _, reason := range []string{"max_output_tokens", "content_filter"} {
		t.Run(reason, func(t *testing.T) {
			state := NewResponsesEventToChatState()
			event := &ResponsesStreamEvent{Type: "response.completed", Response: &ResponsesResponse{
				Status: "incomplete", IncompleteDetails: &ResponsesIncompleteDetails{Reason: reason},
			}}
			chunks := ResponsesEventToChatChunks(event, state)
			require.Len(t, chunks, 1)
			expected := "length"
			if reason == "content_filter" {
				expected = "content_filter"
			}
			require.Equal(t, expected, *chunks[0].Choices[0].FinishReason)
		})
	}
}

func TestResponsesChatTerminalRejectsAmbiguousSuccess(t *testing.T) {
	for _, tc := range []struct{ typ, status, reason string }{
		{"response.incomplete", "", ""},
		{"response.incomplete", "incomplete", "unknown"},
		{"response.incomplete", "completed", "max_output_tokens"},
		{"response.completed", "", ""},
		{"response.completed", "completed", "max_output_tokens"},
		{"response.completed", "in_progress", ""},
		{"response.done", "", ""},
		{"response.done", "cancelled", ""},
		{"response.failed", "failed", ""},
	} {
		t.Run(tc.typ+"/"+tc.status+"/"+tc.reason, func(t *testing.T) {
			state := NewResponsesEventToChatState()
			event := &ResponsesStreamEvent{Type: tc.typ, Response: &ResponsesResponse{Status: tc.status}}
			if tc.reason != "" {
				event.Response.IncompleteDetails = &ResponsesIncompleteDetails{Reason: tc.reason}
			}
			require.Empty(t, ResponsesEventToChatChunks(event, state), "must not manufacture a successful finish")
			require.Empty(t, FinalizeResponsesChatStream(state))
		})
	}
}

func TestResponsesChatTerminalDoesNotDuplicateFinish(t *testing.T) {
	state := NewResponsesEventToChatState()
	event := &ResponsesStreamEvent{Type: "response.completed", Response: &ResponsesResponse{Status: "completed"}}
	require.Len(t, ResponsesEventToChatChunks(event, state), 1)
	require.Empty(t, ResponsesEventToChatChunks(event, state))
}

func TestResponsesToChatUnknownStatusDoesNotBecomeStop(t *testing.T) {
	for _, status := range []string{"", "in_progress", "failed", "incomplete", "cancelled"} {
		chat := ResponsesToChatCompletions(&ResponsesResponse{Status: status}, "deepseek-v4.1-flash")
		require.Empty(t, chat.Choices[0].FinishReason)
	}
}
