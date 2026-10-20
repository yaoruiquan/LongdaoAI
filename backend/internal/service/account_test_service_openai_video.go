package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type AccountTestVideoOptions struct {
	Seconds int    `json:"seconds"`
	Size    string `json:"size"`
}

var accountTestVideoSizePattern = regexp.MustCompile(`^[1-9][0-9]{1,4}x[1-9][0-9]{1,4}$`)

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
