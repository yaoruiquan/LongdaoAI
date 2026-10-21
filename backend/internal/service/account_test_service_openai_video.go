package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/gin-gonic/gin"
)

type AccountTestVideoOptions struct {
	Seconds int    `json:"seconds"`
	Size    string `json:"size"`
}

var accountTestVideoSizePattern = regexp.MustCompile(`^[1-9][0-9]{1,4}x[1-9][0-9]{1,4}$`)
var accountTestVideoTaskIDPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,256}$`)

// AccountTestVideoTask exposes only the fields needed by the admin test preview.
type AccountTestVideoTask struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// GetOpenAIVideoTestTask queries an upstream task using the selected account's credentials.
func (s *AccountTestService) GetOpenAIVideoTestTask(ctx context.Context, accountID int64, taskID string) (*AccountTestVideoTask, error) {
	resp, err := s.doOpenAIVideoTestRequest(ctx, accountID, taskID, false, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if err != nil || len(body) > 1<<20 {
		return nil, infraerrors.New(502, "video_status_invalid", "Failed to read video task status")
	}
	var result struct {
		Status string          `json:"status"`
		Error  json.RawMessage `json:"error"`
	}
	if json.Unmarshal(body, &result) != nil || strings.TrimSpace(result.Status) == "" {
		return nil, infraerrors.New(502, "video_status_invalid", "Upstream returned no valid video task status")
	}
	task := &AccountTestVideoTask{ID: taskID, Status: strings.ToLower(strings.TrimSpace(result.Status))}
	if len(result.Error) > 0 && string(result.Error) != "null" {
		var message string
		var object struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(result.Error, &message) != nil {
			_ = json.Unmarshal(result.Error, &object)
			message = object.Message
		}
		task.Error = truncateString(sanitizeUpstreamErrorMessage(message), 4096)
	}
	return task, nil
}

// GetOpenAIVideoTestContent streams the upstream task content using the selected account's credentials.
func (s *AccountTestService) GetOpenAIVideoTestContent(ctx context.Context, accountID int64, taskID string, rangeHeader string) (*http.Response, error) {
	return s.doOpenAIVideoTestRequest(ctx, accountID, taskID, true, rangeHeader)
}

func (s *AccountTestService) doOpenAIVideoTestRequest(ctx context.Context, accountID int64, taskID string, content bool, rangeHeader string) (*http.Response, error) {
	if accountID <= 0 || taskID == "." || taskID == ".." || !accountTestVideoTaskIDPattern.MatchString(taskID) {
		return nil, infraerrors.New(400, "invalid_video_task", "Invalid video task ID or account ID")
	}
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil || account == nil {
		return nil, infraerrors.New(404, "account_not_found", "Account not found")
	}
	if !account.IsOpenAI() || account.Type != AccountTypeAPIKey || !account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityVideos) {
		return nil, infraerrors.New(400, "video_capability_required", "Video testing requires an OpenAI API Key account with videos capability enabled")
	}
	authToken := account.GetOpenAIApiKey()
	if authToken == "" {
		return nil, infraerrors.New(400, "api_key_required", "No API key available")
	}
	baseURL, err := s.validateUpstreamBaseURL(account.GetOpenAIBaseURL())
	if err != nil {
		return nil, infraerrors.New(400, "invalid_upstream_url", "Invalid upstream base URL")
	}
	if s.httpUpstream == nil {
		return nil, infraerrors.New(503, "upstream_unavailable", "HTTP upstream not configured")
	}
	requestCtx, cancel := context.WithTimeout(WithHTTPUpstreamProfile(ctx, HTTPUpstreamProfileOpenAI), 3*time.Minute)
	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, buildOpenAIVideoURL(baseURL, taskID, content), nil)
	if err != nil {
		cancel()
		return nil, infraerrors.New(502, "video_request_failed", "Failed to create video task request")
	}
	req.Header.Set("Authorization", "Bearer "+authToken)
	if content {
		req.Header.Set("Accept", "*/*")
	} else {
		req.Header.Set("Accept", "application/json")
	}
	if ua := account.GetOpenAIUserAgent(); ua != "" {
		req.Header.Set("User-Agent", ua)
	}
	account.ApplyHeaderOverrides(req.Header)
	if content && rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}
	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	resp, err := s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, s.tlsFPProfileService.ResolveTLSProfile(account))
	if err != nil {
		cancel()
		return nil, infraerrors.New(502, "video_request_failed", "Video upstream request failed; refresh the existing task later")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer cancel()
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		message := strings.ReplaceAll(sanitizeUpstreamErrorMessage(string(body)), authToken, "[REDACTED]")
		status := resp.StatusCode
		// Upstream authentication failures must not log the administrator out.
		if status < 400 || status > 599 || status == http.StatusUnauthorized || status == http.StatusForbidden {
			status = 502
		}
		return nil, infraerrors.New(status, "video_upstream_error", fmt.Sprintf("Video upstream returned %d: %s", resp.StatusCode, message))
	}
	if !content {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
		_ = resp.Body.Close()
		if readErr != nil || len(body) > 1<<20 {
			cancel()
			return nil, infraerrors.New(502, "video_status_invalid", "Failed to read video task status")
		}
		var decoded any
		if json.Unmarshal(body, &decoded) != nil {
			cancel()
			return nil, infraerrors.New(502, "video_status_invalid", "Upstream returned no valid video task status")
		}
		redactVideoTaskSecrets(decoded, authToken)
		redactedBody, err := json.Marshal(decoded)
		if err != nil {
			cancel()
			return nil, infraerrors.New(502, "video_status_invalid", "Failed to sanitize video task status")
		}
		resp.Body = io.NopCloser(strings.NewReader(string(redactedBody)))
	} else {
		mediaType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
		if mediaType != "" && mediaType != "application/octet-stream" && !strings.HasPrefix(mediaType, "video/") {
			_ = resp.Body.Close()
			cancel()
			return nil, infraerrors.New(502, "video_content_invalid", "Upstream returned non-video content; refresh the existing task later")
		}
		if mediaType == "" {
			resp.Header.Set("Content-Type", "application/octet-stream")
		}
	}
	// Keep the request context alive while the caller streams the response body.
	resp.Body = &cancelOnCloseBody{ReadCloser: resp.Body, cancel: cancel}
	return resp, nil
}

func redactVideoTaskSecrets(value any, secret string) {
	if secret == "" {
		return
	}
	switch item := value.(type) {
	case map[string]any:
		for key, child := range item {
			redactVideoTaskSecrets(child, secret)
			if text, ok := child.(string); ok {
				item[key] = strings.ReplaceAll(text, secret, "[REDACTED]")
			}
		}
	case []any:
		for index, child := range item {
			redactVideoTaskSecrets(child, secret)
			if text, ok := child.(string); ok {
				item[index] = strings.ReplaceAll(text, secret, "[REDACTED]")
			}
		}
	}
}

type cancelOnCloseBody struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (b *cancelOnCloseBody) Close() error {
	err := b.ReadCloser.Close()
	b.cancel()
	return err
}

// This probe checks task acceptance only. It never retries a billable submission.
func (s *AccountTestService) testOpenAIVideoAccountConnection(c *gin.Context, account *Account, model, prompt string, options AccountTestVideoOptions) error {
	if account == nil || !account.IsOpenAI() || account.Type != AccountTypeAPIKey ||
		!account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityVideos) {
		return s.sendErrorAndEnd(c, "Video testing requires an OpenAI API Key account with videos capability enabled")
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return s.sendErrorAndEnd(c, "Select a video model before submitting a test task")
	}
	model = account.GetMappedModel(model)
	if options.Seconds == 0 {
		options.Seconds = 6
	}
	if options.Seconds < 1 {
		return s.sendErrorAndEnd(c, "Video seconds must be a positive integer")
	}
	options.Size = strings.TrimSpace(options.Size)
	if options.Size == "" {
		options.Size = "1280x720"
	}
	if !accountTestVideoSizePattern.MatchString(options.Size) {
		return s.sendErrorAndEnd(c, "Video size must use WIDTHxHEIGHT format, for example 1280x720")
	}
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		prompt = "A small paper boat gently floating on a calm pond, soft daylight, static camera, no people, no text."
	}
	authToken := account.GetOpenAIApiKey()
	if authToken == "" {
		return s.sendErrorAndEnd(c, "No API key available")
	}
	baseURL, err := s.validateUpstreamBaseURL(account.GetOpenAIBaseURL())
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Invalid base URL: %s", err))
	}
	if s.httpUpstream == nil {
		return s.sendErrorAndEnd(c, "HTTP upstream not configured")
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Minute)
	defer cancel()
	payload, err := json.Marshal(map[string]any{
		"model": model, "prompt": prompt, "seconds": strconv.Itoa(options.Seconds), "size": options.Size,
	})
	if err != nil {
		return s.sendErrorAndEnd(c, "Failed to build video test request")
	}
	req, err := http.NewRequestWithContext(WithHTTPUpstreamProfile(ctx, HTTPUpstreamProfileOpenAI), http.MethodPost, buildOpenAIVideoURL(baseURL, "", false), bytes.NewReader(payload))
	if err != nil {
		return s.sendErrorAndEnd(c, "Failed to create video test request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)
	if ua := account.GetOpenAIUserAgent(); ua != "" {
		req.Header.Set("User-Agent", ua)
	}
	account.ApplyHeaderOverrides(req.Header)
	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
	c.Writer.Flush()
	s.sendEvent(c, TestEvent{Type: "test_start", Model: model})
	s.sendEvent(c, TestEvent{Type: "status", Text: "正在通过 POST /v1/videos 提交视频测试任务"})
	resp, err := s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, s.tlsFPProfileService.ResolveTLSProfile(account))
	if err != nil {
		return s.sendErrorAndEnd(c, "Video API (/v1/videos) request failed; the task may already have been accepted, check the upstream before retrying: "+strings.ReplaceAll(sanitizeUpstreamErrorMessage(err.Error()), authToken, "[REDACTED]"))
	}
	defer resp.Body.Close()
	const maxResponseSize = 1 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil || len(body) > maxResponseSize {
		return s.sendErrorAndEnd(c, "Failed to read video task response; check the upstream before retrying")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		detail := truncateString(strings.ReplaceAll(sanitizeUpstreamErrorMessage(string(body)), authToken, "[REDACTED]"), 4096)
		return s.sendErrorAndEnd(c, fmt.Sprintf("Video API (/v1/videos) returned %d: %s", resp.StatusCode, detail))
	}
	var result struct {
		ID        string          `json:"id"`
		TaskID    string          `json:"task_id"`
		RequestID string          `json:"request_id"`
		Status    string          `json:"status"`
		Error     json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return s.sendErrorAndEnd(c, "Invalid video task response; check the upstream before retrying")
	}
	taskID := strings.TrimSpace(firstNonEmptyString(result.ID, result.TaskID, result.RequestID))
	status := strings.ToLower(strings.TrimSpace(result.Status))
	if taskID == "" || (len(result.Error) > 0 && string(result.Error) != "null") {
		return s.sendErrorAndEnd(c, "Video API did not return an accepted task ID; check the upstream before retrying")
	}
	s.sendEvent(c, TestEvent{Type: "video_task", TaskID: taskID, Status: status})
	switch status {
	case "failed", "error", "cancelled", "canceled", "expired":
		return s.sendErrorAndEnd(c, "Video task "+taskID+" returned status: "+status)
	}
	s.sendEvent(c, TestEvent{Type: "status", Text: "视频任务已受理，任务 ID：" + taskID + "；本次仅验证提交，不等待视频生成完成"})
	s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
	return nil
}
