package apicompat

import "errors"

// NormalizeResponsesChatTerminal resolves a terminal envelope without changing
// the raw event. Ambiguous states must not become successful Chat completions.
func NormalizeResponsesChatTerminal(event *ResponsesStreamEvent) (*ResponsesResponse, error) {
	if event == nil || event.Response == nil {
		return nil, errors.New("upstream terminal event is missing response")
	}
	response := *event.Response
	expected := ""
	switch event.Type {
	case "response.completed":
	case "response.incomplete":
		expected = "incomplete"
	case "response.failed":
		expected = "failed"
	case "response.done":
	default:
		return nil, errors.New("unrecognized upstream terminal event")
	}
	if response.Status == "" {
		response.Status = expected
	}
	if expected != "" && response.Status != expected {
		return nil, errors.New("upstream terminal event conflicts with response status")
	}
	switch response.Status {
	case "completed":
		if response.IncompleteDetails != nil || response.Error != nil {
			return nil, errors.New("completed upstream response contains incomplete details or error")
		}
	case "incomplete":
		if response.IncompleteDetails == nil || response.Error != nil {
			return nil, errors.New("upstream incomplete response is missing a supported reason")
		}
		switch response.IncompleteDetails.Reason {
		case "max_output_tokens", "content_filter":
		default:
			return nil, errors.New("upstream incomplete response has an unsupported reason")
		}
	case "failed":
	default:
		return nil, errors.New("upstream terminal response has a missing or unsupported status")
	}
	return &response, nil
}
