//go:build unit

package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

const sessionPolicyTestBody = `{"error":{"code":"session_blocked_by_cyber_policy","type":"session_blocked_by_cyber_policy","message":"This session is blocked; start a new session."}}`

func TestOpenAISessionPolicy_ForwardDoesNotFailoverOrBlockSharedAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, passthrough := range []bool{false, true} {
		t.Run(fmt.Sprintf("passthrough_%t", passthrough), func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(nil))
			body := &passthroughCloseTrackingReadCloser{Reader: strings.NewReader(sessionPolicyTestBody)}
			upstream := &httpUpstreamRecorder{resp: &http.Response{
				StatusCode: http.StatusForbidden,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       body,
			}}
			repo := &rateLimitAccountRepoStub{}
			counter := &openAI403CounterCacheStub{counts: []int64{1}}
			rateLimiter := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
			rateLimiter.SetOpenAI403CounterCache(counter)
			svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream, rateLimitService: rateLimiter}
			rateLimiter.SetAccountRuntimeBlocker(svc)
			account := &Account{
				ID: 31, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Concurrency: 1,
				Status: StatusActive, Schedulable: true,
				Credentials: map[string]any{"api_key": "sk-test", "base_url": "https://api.example.test"},
				Extra:       map[string]any{"openai_passthrough": passthrough, "openai_responses_supported": true},
			}
			result, err := svc.Forward(context.Background(), c, account, []byte(`{"model":"gpt-6.1-sol","stream":false,"input":"hello"}`))
			require.Nil(t, result)
			require.Error(t, err)
			var failover *UpstreamFailoverError
			require.False(t, errors.As(err, &failover), "a blocked session must stop this request rather than retry other accounts")
			require.Equal(t, http.StatusForbidden, rec.Code)
			require.JSONEq(t, sessionPolicyTestBody, rec.Body.String())
			require.True(t, IsResponseCommitted(c), "handler must not append a generic fallback error")
			require.Nil(t, GetOpsCyberPolicy(c), "session rejection must not be counted as a new moderation event")
			require.True(t, body.closed)
			require.Zero(t, repo.tempCalls)
			require.Zero(t, repo.setErrorCalls)
			require.Len(t, counter.counts, 1)
			require.False(t, svc.isOpenAIAccountRuntimeBlocked(account))
		})
	}
}

func TestOpenAISessionPolicy_CompactCommittedStreamEndsWithFailure(t *testing.T) {
	for _, passthrough := range []bool{false, true} {
		t.Run(fmt.Sprintf("passthrough_%t", passthrough), func(t *testing.T) {
			c, rec := newCompactBridgeTestContext(t, true)
			stop := StartOpenAICompactSSEKeepalive(c, keepaliveTestInterval)
			defer stop()
			waitForKeepaliveBeats()
			svc := &OpenAIGatewayService{cfg: &config.Config{}}
			resp := &http.Response{StatusCode: http.StatusForbidden, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(sessionPolicyTestBody))}
			account := &Account{ID: 31, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
			var err error
			if passthrough {
				err = svc.handleErrorResponsePassthrough(context.Background(), resp, c, account, nil, []byte(sessionPolicyTestBody))
			} else {
				_, err = svc.handleErrorResponse(context.Background(), resp, c, account, nil)
			}
			require.Error(t, err)
			require.Equal(t, http.StatusOK, rec.Code, "the keepalive already committed HTTP 200")
			events := parseCompactBridgeSSE(t, stripKeepaliveComments(rec.Body.String()))
			require.Len(t, events, 1)
			require.Equal(t, "response.failed", events[0][0])
			require.Equal(t, "session_blocked_by_cyber_policy", gjson.Get(events[0][1], "response.error.code").String())
			streamErr, ok := GetOpsStreamError(c)
			require.True(t, ok)
			require.Equal(t, http.StatusForbidden, streamErr.IntendedStatus)
			require.True(t, IsResponseCommitted(c))
		})
	}
}

func TestOpenAISessionPolicy_CompatPreservesForbiddenStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	svc := &OpenAIGatewayService{cfg: &config.Config{}}
	resp := &http.Response{StatusCode: http.StatusForbidden, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(sessionPolicyTestBody))}
	writeError := func(c *gin.Context, status int, errType, message string) {
		c.JSON(status, gin.H{"error": gin.H{"type": errType, "message": message}})
	}
	result, err := svc.handleCompatErrorResponse(resp, c, &Account{ID: 31, Platform: PlatformOpenAI}, writeError)
	require.Nil(t, result)
	require.Error(t, err)
	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Equal(t, "permission_error", gjson.Get(rec.Body.String(), "error.type").String())
	require.Contains(t, gjson.Get(rec.Body.String(), "error.message").String(), "start a new session")
	require.True(t, IsResponseCommitted(c))
	require.Nil(t, GetOpsCyberPolicy(c))
}
