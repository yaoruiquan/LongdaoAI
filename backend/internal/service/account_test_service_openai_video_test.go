//go:build unit

package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type videoAccountTestRepo struct {
	AccountRepository
	account *Account
}

func (r videoAccountTestRepo) GetByID(context.Context, int64) (*Account, error) {
	return r.account, nil
}

func newVideoAccountTestFixture(response string, status int) (*AccountTestService, *Account, *queuedHTTPUpstream) {
	account := &Account{ID: 38, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{
		"api_key": "provider-secret", "base_url": "https://video.example/v1", "openai_capabilities": []any{"videos"},
		"model_mapping": map[string]any{"alias": "minimax-h3"},
	}}
	upstream := &queuedHTTPUpstream{responses: []*http.Response{newJSONResponse(status, response)}}
	svc := &AccountTestService{accountRepo: videoAccountTestRepo{account: account}, httpUpstream: upstream, cfg: &config.Config{}}
	return svc, account, upstream
}

func TestAccountTestService_VideoModeSubmitsNativeTask(t *testing.T) {
	for _, base := range []string{"https://video.example", "https://video.example/v1", "https://video.example/v1/videos", "https://video.example/v1/video/generations"} {
		t.Run(base, func(t *testing.T) {
			svc, account, upstream := newVideoAccountTestFixture(`{"id":"task_123","status":"queued"}`, http.StatusAccepted)
			account.Credentials["base_url"] = base
			c, recorder := newTestContext()
			err := svc.TestAccountConnection(c, account.ID, "alias", "paper boat", "video", AccountTestVideoOptions{Seconds: 10, Size: "1920x1080"})
			require.NoError(t, err)
			require.Len(t, upstream.requests, 1)
			req := upstream.requests[0]
			require.Equal(t, "https://video.example/v1/videos", req.URL.String())
			require.Equal(t, http.MethodPost, req.Method)
			require.Equal(t, HTTPUpstreamProfileOpenAI, HTTPUpstreamProfileFromContext(req.Context()))
			require.Equal(t, "Bearer provider-secret", req.Header.Get("Authorization"))
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			require.Equal(t, "minimax-h3", gjson.GetBytes(body, "model").String())
			require.Equal(t, "paper boat", gjson.GetBytes(body, "prompt").String())
			require.Equal(t, "10", gjson.GetBytes(body, "seconds").String())
			require.Equal(t, "1920x1080", gjson.GetBytes(body, "size").String())
			require.False(t, gjson.GetBytes(body, "messages").Exists())
			require.Contains(t, recorder.Body.String(), `"type":"video_task","status":"queued","task_id":"task_123"`)
			require.Contains(t, recorder.Body.String(), `"success":true`)
			require.NotContains(t, recorder.Body.String(), "provider-secret")
		})
	}
}

func TestAccountTestService_VideoOnlyDefaultAndBackgroundDoNotSubmit(t *testing.T) {
	svc, account, upstream := newVideoAccountTestFixture(`{"id":"task_123"}`, 200)
	for _, mode := range []string{"default", "compact"} {
		c, recorder := newTestContext()
		err := svc.TestAccountConnection(c, account.ID, "minimax-h3", "", mode)
		require.ErrorContains(t, err, "select video mode")
		require.NotContains(t, recorder.Body.String(), `"success":true`)
	}
	result, err := svc.RunTestBackground(context.Background(), account.ID, "minimax-h3")
	require.NoError(t, err)
	require.Equal(t, "failed", result.Status)
	require.Empty(t, upstream.requests, "scheduled/default tests must not create billable video tasks")
}

func TestAccountTestService_VideoDefaultsAndTaskIDAlias(t *testing.T) {
	svc, account, upstream := newVideoAccountTestFixture(`{"task_id":"provider_task","status":"processing"}`, 200)
	c, recorder := newTestContext()
	require.NoError(t, svc.TestAccountConnection(c, account.ID, "minimax-h3", "", "video"))
	body, err := io.ReadAll(upstream.requests[0].Body)
	require.NoError(t, err)
	require.Equal(t, "6", gjson.GetBytes(body, "seconds").String())
	require.Equal(t, "1280x720", gjson.GetBytes(body, "size").String())
	require.NotEmpty(t, gjson.GetBytes(body, "prompt").String())
	require.Contains(t, recorder.Body.String(), `"task_id":"provider_task"`)
}

func TestAccountTestService_VideoValidationPreventsUpstreamCall(t *testing.T) {
	for _, tc := range []struct {
		name    string
		edit    func(*Account)
		model   string
		options AccountTestVideoOptions
	}{
		{name: "missing_model"},
		{name: "negative_seconds", model: "m", options: AccountTestVideoOptions{Seconds: -1}},
		{name: "invalid_size", model: "m", options: AccountTestVideoOptions{Size: "720p"}},
		{name: "oauth", model: "m", edit: func(a *Account) { a.Type = AccountTypeOAuth }},
		{name: "capability_disabled", model: "m", edit: func(a *Account) { a.Credentials["openai_capabilities"] = []string{"chat_completions"} }},
		{name: "missing_key", model: "m", edit: func(a *Account) { delete(a.Credentials, "api_key") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, account, upstream := newVideoAccountTestFixture(`{"id":"task_123"}`, 200)
			if tc.edit != nil {
				tc.edit(account)
			}
			c, _ := newTestContext()
			require.Error(t, svc.TestAccountConnection(c, account.ID, tc.model, "", "video", tc.options))
			require.Empty(t, upstream.requests)
		})
	}
}

func TestAccountTestService_VideoRejectsErrorsWithoutRetry(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"upstream_error", `{"error":{"message":"provider-secret unavailable"}}`, 400},
		{"invalid_json", `not-json`, 200},
		{"missing_task", `{"status":"queued"}`, 200},
		{"failed_task", `{"id":"task_failed","status":"failed"}`, 200},
		{"error_with_id", `{"id":"task_error","error":{"message":"no"}}`, 200},
		{"too_large", strings.Repeat("x", (1<<20)+1), 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, account, upstream := newVideoAccountTestFixture(tc.body, tc.status)
			c, recorder := newTestContext()
			require.Error(t, svc.TestAccountConnection(c, account.ID, "minimax-h3", "", "video"))
			require.Len(t, upstream.requests, 1)
			require.NotContains(t, recorder.Body.String(), `"success":true`)
			require.NotContains(t, recorder.Body.String(), "provider-secret")
		})
	}
}
