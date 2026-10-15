package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type imageUsageBillingRepo struct {
	service.UsageBillingRepository
	evidence *service.RequestBillingEvidence
}

func (r *imageUsageBillingRepo) FindRequestBillingEvidence(_ context.Context, _, _ int64, _ []string) (*service.RequestBillingEvidence, error) {
	return r.evidence, nil
}

func TestRequestImageUsageSettlesOnlyReliableImageBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	amount := "0.0125000000"
	evidence := &service.RequestBillingEvidence{Model: "gpt-image-1", BillingMode: "image", ImageCount: 2, ActualCostUSD: &amount, TransactionConfirmed: true}
	usageRepo := &imageUsageBillingRepo{evidence: evidence}
	openAI := service.NewOpenAIGatewayService(nil, nil, usageRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	handler := &GatewayHandler{openAIGatewayService: openAI}

	groupID := int64(5)
	apiKey := &service.APIKey{ID: 7, UserID: 11, GroupID: &groupID, Group: &service.Group{ID: groupID, Platform: service.PlatformOpenAI}}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/sub2api/usage?request_id=req-1", nil)
	ctx.Set(string(middleware2.ContextKeyAPIKey), apiKey)
	handler.RequestImageUsage(ctx)

	var response requestUsageResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "settled", response.Status)
	require.Equal(t, "USD", response.Currency)
	require.Equal(t, amount, *response.ActualCostUSD)
	require.Equal(t, 2, *response.ImageCount)
	require.Empty(t, response.PendingReason)
}

func TestReliableImageBillingEvidenceRejectsUnverifiedOrInvalidAmounts(t *testing.T) {
	validAmount := "0"
	negative := "-0.01"
	tooLarge := "1e9999"
	valid := func(amount *string) *service.RequestBillingEvidence {
		return &service.RequestBillingEvidence{Model: "gpt-image-1", BillingMode: "per_request", ImageCount: 1, ActualCostUSD: amount, TransactionConfirmed: true}
	}
	tests := []struct {
		name     string
		evidence *service.RequestBillingEvidence
		want     bool
	}{
		{name: "zero is a valid confirmed free charge", evidence: valid(&validAmount), want: true},
		{name: "not confirmed", evidence: &service.RequestBillingEvidence{Model: "gpt-image-1", BillingMode: "image", ImageCount: 1, ActualCostUSD: &validAmount}},
		{name: "token billing mode", evidence: &service.RequestBillingEvidence{Model: "gpt-image-1", BillingMode: "token", ImageCount: 1, ActualCostUSD: &validAmount, TransactionConfirmed: true}},
		{name: "missing model", evidence: &service.RequestBillingEvidence{BillingMode: "image", ImageCount: 1, ActualCostUSD: &validAmount, TransactionConfirmed: true}},
		{name: "no images", evidence: &service.RequestBillingEvidence{Model: "gpt-image-1", BillingMode: "image", ImageCount: 0, ActualCostUSD: &validAmount, TransactionConfirmed: true}},
		{name: "negative amount", evidence: valid(&negative)},
		{name: "infinite amount", evidence: valid(&tooLarge)},
		{name: "missing amount", evidence: valid(nil)},
		{name: "nil evidence", evidence: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, reliableImageBillingEvidence(tt.evidence))
		})
	}
}

func TestRequestImageUsageKeepsUnverifiedEvidencePending(t *testing.T) {
	gin.SetMode(gin.TestMode)
	amount := "0.04"
	usageRepo := &imageUsageBillingRepo{evidence: &service.RequestBillingEvidence{Model: "gpt-image-1", BillingMode: "token", ImageCount: 1, ActualCostUSD: &amount, TransactionConfirmed: true}}
	openAI := service.NewOpenAIGatewayService(nil, nil, usageRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	handler := &GatewayHandler{openAIGatewayService: openAI, cfg: &config.Config{}}
	groupID := int64(5)
	apiKey := &service.APIKey{ID: 7, UserID: 11, GroupID: &groupID, Group: &service.Group{ID: groupID, Platform: service.PlatformOpenAI}}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/sub2api/usage?request_id=req-1", nil)
	ctx.Set(string(middleware2.ContextKeyAPIKey), apiKey)
	handler.RequestImageUsage(ctx)

	var response requestUsageResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, "pending", response.Status)
	require.Nil(t, response.ActualCostUSD)
	require.Equal(t, "transaction_confirmed_image_amount_unverified", response.PendingReason)
}
