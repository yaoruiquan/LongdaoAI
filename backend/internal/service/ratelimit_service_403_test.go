//go:build unit

package service

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type runtimeBlockRecorder struct {
	accounts   []*Account
	until      []time.Time
	reasons    []string
	clearedIDs []int64
}

func (r *runtimeBlockRecorder) BlockAccountScheduling(account *Account, until time.Time, reason string) {
	r.accounts = append(r.accounts, account)
	r.until = append(r.until, until)
	r.reasons = append(r.reasons, reason)
}

func (r *runtimeBlockRecorder) ClearAccountSchedulingBlock(accountID int64) {
	r.clearedIDs = append(r.clearedIDs, accountID)
}

func TestRateLimitService_HandleUpstreamError_OpenAI403FirstHitTempUnschedulable(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	counter := &openAI403CounterCacheStub{counts: []int64{1}}
	blocker := &runtimeBlockRecorder{}
	service := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	service.SetOpenAI403CounterCache(counter)
	service.SetAccountRuntimeBlocker(blocker)
	account := &Account{
		ID:       301,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
	}

	shouldDisable := service.HandleUpstreamError(
		context.Background(),
		account,
		http.StatusForbidden,
		http.Header{},
		[]byte(`{"error":{"message":"temporary edge rejection"}}`),
	)

	require.True(t, shouldDisable)
	require.Equal(t, 0, repo.setErrorCalls)
	require.Equal(t, 1, repo.tempCalls)
	require.Contains(t, repo.lastTempReason, "temporary edge rejection")
	require.Contains(t, repo.lastTempReason, "(1/3)")
	require.Len(t, blocker.accounts, 1)
	require.Equal(t, account.ID, blocker.accounts[0].ID)
	require.Equal(t, "openai_403_temp", blocker.reasons[0])
	require.True(t, blocker.until[0].After(time.Now()))
}

func TestRateLimitService_HandleUpstreamError_OpenAI403ThresholdDisables(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	counter := &openAI403CounterCacheStub{counts: []int64{3}}
	service := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	service.SetOpenAI403CounterCache(counter)
	account := &Account{
		ID:       302,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
	}

	shouldDisable := service.HandleUpstreamError(
		context.Background(),
		account,
		http.StatusForbidden,
		http.Header{},
		[]byte(`{"error":{"message":"workspace forbidden by policy"}}`),
	)

	require.True(t, shouldDisable)
	require.Equal(t, 1, repo.setErrorCalls)
	require.Equal(t, 0, repo.tempCalls)
	require.Contains(t, repo.lastErrorMsg, "workspace forbidden by policy")
	require.Contains(t, repo.lastErrorMsg, "consecutive_403=3/3")
}

// A session policy rejection must never quarantine the shared upstream account,
// including when a broad local 403 rule would otherwise match it.
func TestRateLimitService_HandleUpstreamError_SessionPolicyDoesNotBlockAccount(t *testing.T) {
	for _, accountType := range []string{AccountTypeAPIKey, AccountTypeOAuth} {
		for _, configuredRule := range []bool{false, true} {
			t.Run(accountType+fmt.Sprint("/temp_rule=", configuredRule), func(t *testing.T) {
				repo := &rateLimitAccountRepoStub{}
				counter := &openAI403CounterCacheStub{counts: []int64{1, 2, 3}}
				gateway := &OpenAIGatewayService{}
				svc := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
				svc.SetOpenAI403CounterCache(counter)
				svc.SetAccountRuntimeBlocker(gateway)
				gateway.rateLimitService = svc
				account := &Account{ID: 31, Platform: PlatformOpenAI, Type: accountType, Status: StatusActive, Schedulable: true}
				if configuredRule {
					account.Credentials = map[string]any{
						"temp_unschedulable_enabled": true,
						"temp_unschedulable_rules": []any{map[string]any{
							"error_code": float64(403), "keywords": []any{"session_blocked_by_cyber_policy"}, "duration_minutes": float64(10),
						}},
					}
				}
				bodies := []string{
					`{"error":{"code":"session_blocked_by_cyber_policy","message":"This session is blocked; start a new session."}}`,
					`{"error":{"type":"session_blocked_by_cyber_policy","message":"This session is blocked; start a new session."}}`,
					`{"response":{"error":{"code":"session_blocked_by_cyber_policy","message":"This session is blocked; start a new session."}}}`,
					`{"response":{"error":{"type":"session_blocked_by_cyber_policy","message":"This session is blocked; start a new session."}}}`,
				}
				for _, body := range bodies {
					require.False(t, gateway.handleOpenAIAccountUpstreamError(context.Background(), account, http.StatusForbidden, http.Header{}, []byte(body), "gpt-6.1-sol"))
				}
				require.Zero(t, repo.tempCalls)
				require.Zero(t, repo.setErrorCalls)
				require.Len(t, counter.counts, 3, "session rejection must not increment the account 403 counter")
				require.Empty(t, counter.resetCalls, "session rejection must not clear real account failures")
				require.False(t, gateway.isOpenAIAccountRuntimeBlocked(account))
			})
		}
	}
}

func TestRateLimitService_CheckErrorPolicy_SessionPolicySkipsAccountRules(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	blocker := &runtimeBlockRecorder{}
	svc := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	svc.SetAccountRuntimeBlocker(blocker)
	account := &Account{ID: 31, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{
		"temp_unschedulable_enabled": true,
		"temp_unschedulable_rules": []any{map[string]any{
			"error_code": float64(403), "keywords": []any{"session_blocked_by_cyber_policy"}, "duration_minutes": float64(10),
		}},
	}}
	body := []byte(`{"error":{"code":"session_blocked_by_cyber_policy","message":"Start a new session."}}`)
	require.Equal(t, ErrorPolicySkipped, svc.CheckErrorPolicy(context.Background(), account, http.StatusForbidden, body))
	require.Zero(t, repo.tempCalls)
	require.Zero(t, repo.setErrorCalls)
	require.Empty(t, blocker.accounts)
}

func TestRateLimitService_HandleUpstreamError_SessionPolicyWithoutCounterCache(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	blocker := &runtimeBlockRecorder{}
	svc := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	svc.SetAccountRuntimeBlocker(blocker)
	account := &Account{ID: 31, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	body := []byte(`{"error":{"code":"session_blocked_by_cyber_policy","message":"Start a new session."}}`)
	require.False(t, svc.HandleUpstreamError(context.Background(), account, http.StatusForbidden, http.Header{}, body))
	require.Zero(t, repo.setErrorCalls, "a missing counter cache must not permanently disable the account")
	require.Zero(t, repo.tempCalls)
	require.Empty(t, blocker.accounts)
}

func TestRateLimitService_HandleUpstreamError_SessionPolicyMentionDoesNotHideAccount403(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	counter := &openAI403CounterCacheStub{counts: []int64{3}}
	svc := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	svc.SetOpenAI403CounterCache(counter)
	account := &Account{ID: 31, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	body := []byte(`{"error":{"code":"account_suspended","message":"Account suspended; this is not session_blocked_by_cyber_policy."}}`)
	require.True(t, svc.HandleUpstreamError(context.Background(), account, http.StatusForbidden, http.Header{}, body))
	require.Equal(t, 1, repo.setErrorCalls)
	require.Empty(t, counter.counts)
}
