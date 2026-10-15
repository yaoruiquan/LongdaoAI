package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type imagePricingChannelRepo struct {
	service.ChannelRepository
	channel service.Channel
}

func (r *imagePricingChannelRepo) ListAll(context.Context) ([]service.Channel, error) {
	if r.channel.ID == 0 {
		return nil, nil
	}
	return []service.Channel{r.channel}, nil
}

func (r *imagePricingChannelRepo) GetGroupPlatforms(context.Context, []int64) (map[int64]string, error) {
	return map[int64]string{5: service.PlatformOpenAI}, nil
}

func newImagePricingHandlerForTest(channel *service.Channel) (*GatewayHandler, *service.APIKey) {
	groupID := int64(5)
	key := &service.APIKey{ID: 7, UserID: 11, GroupID: &groupID, Group: &service.Group{
		ID: groupID, Platform: service.PlatformOpenAI, AllowImageGeneration: true, RateMultiplier: 1,
		ImageRateIndependent: true, ImageRateMultiplier: 2.19, VideoRateMultiplier: 1,
	}}
	h := newGatewayModelsHandlerForTest(&gatewayModelsAccountRepoStub{byGroup: map[int64][]service.Account{
		groupID: {{ID: 1, Platform: service.PlatformOpenAI, Credentials: map[string]any{
			"model_mapping": map[string]any{"gpt-image-2": "gpt-image-2"},
		}}},
	}})
	repo := &imagePricingChannelRepo{}
	if channel != nil {
		channel.ID, channel.Status, channel.GroupIDs = 1, service.StatusActive, []int64{groupID}
		for i := range channel.ModelPricing {
			channel.ModelPricing[i].Platform = service.PlatformOpenAI
		}
		repo.channel = *channel
	}
	cs := service.NewChannelService(repo, nil, nil, nil)
	bs := service.NewBillingService(&config.Config{}, nil)
	h.openAIGatewayService = service.NewOpenAIGatewayService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		bs, nil, nil, nil, nil, nil, nil, service.NewModelPricingResolver(cs, bs), cs, nil, nil, nil)
	return h, key
}

func imagePricingRequestForTest(t *testing.T, h *GatewayHandler, key *service.APIKey) (imagePricingResponse, map[string]any) {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/sub2api/image-pricing", strings.NewReader(
		`{"model":"gpt-image-2","operation":"generate","size":"2048x2048","n":1}`))
	ctx.Set(string(middleware2.ContextKeyAPIKey), key)
	h.ImagePricing(ctx)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))
	var response imagePricingResponse
	var raw map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &raw))
	return response, raw
}

func TestImagePricingAuthorityContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	groupPrice, channelPrice, invalidPrice := 0.02, 0.3, -0.1
	for _, tt := range []struct {
		name       string
		channel    *service.Channel
		groupPrice *float64
		source     string
		mode       service.BillingMode
		quoted     bool
		amount     string
	}{
		{name: "commercial fallback", source: "fallback", mode: service.BillingModeImage, quoted: true, amount: "0.4401900000"},
		{name: "group", groupPrice: &groupPrice, source: "group", mode: service.BillingModeImage, quoted: true, amount: "0.0438000000"},
		{name: "channel per request", channel: &service.Channel{ModelPricing: []service.ChannelModelPricing{{Models: []string{"gpt-image-2"}, BillingMode: service.BillingModePerRequest, PerRequestPrice: &channelPrice}}}, source: "channel", mode: service.BillingModePerRequest, quoted: true, amount: "0.6570000000"},
		{name: "token", groupPrice: &groupPrice, channel: &service.Channel{ModelPricing: []service.ChannelModelPricing{{Models: []string{"gpt-image-2"}, BillingMode: service.BillingModeToken}}}, source: "channel", mode: service.BillingModeToken},
		{name: "unresolved upstream", channel: &service.Channel{BillingModelSource: service.BillingModelSourceUpstream}, source: "upstream_model_unresolved", mode: service.BillingModeImage},
		{name: "invalid amount", groupPrice: &invalidPrice, source: "group", mode: service.BillingModeImage},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h, key := newImagePricingHandlerForTest(tt.channel)
			key.Group.ImagePrice2K = tt.groupPrice
			response, raw := imagePricingRequestForTest(t, h, key)
			require.Equal(t, "sub2api.image_pricing", response.Object)
			require.Equal(t, 1, response.SchemaVersion)
			require.Equal(t, "USD", response.Currency)
			require.Equal(t, tt.mode, response.BillingMode)
			require.Equal(t, tt.source, response.PricingSource)
			require.Equal(t, "medium", response.Quality)
			if tt.mode == service.BillingModeToken {
				require.Equal(t, 1.0, response.EffectiveRateMultiplier)
			} else {
				require.Equal(t, 2.19, response.EffectiveRateMultiplier)
			}
			if tt.quoted {
				require.Equal(t, "quoted", response.PricingStatus)
				require.NotNil(t, response.AmountUSD)
				require.Equal(t, tt.amount, *response.AmountUSD)
				require.Equal(t, "sub2api_customer_billing", raw["pricing_authority"])
			} else {
				require.Equal(t, "unavailable", response.PricingStatus)
				require.Nil(t, response.AmountUSD)
				require.NotContains(t, raw, "pricing_authority")
			}
		})
	}
}

func TestImagePricingFingerprintIncludesAuthorityAndIsStable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, key := newImagePricingHandlerForTest(nil)
	response, _ := imagePricingRequestForTest(t, h, key)
	repeated, _ := imagePricingRequestForTest(t, h, key)
	require.Equal(t, response.PricingVersion, repeated.PricingVersion)
	fingerprint := struct {
		Request    imagePricingRequest
		Mode       service.BillingMode
		Source     string
		Authority  string
		Amount     *string
		Multiplier float64
	}{response.imagePricingRequest, response.BillingMode, response.PricingSource,
		"sub2api_customer_billing", response.AmountUSD, response.EffectiveRateMultiplier}
	encoded, err := json.Marshal(fingerprint)
	require.NoError(t, err)
	sum := sha256.Sum256(encoded)
	require.Equal(t, "v1:"+hex.EncodeToString(sum[:]), response.PricingVersion)
	fingerprint.Authority = ""
	encoded, err = json.Marshal(fingerprint)
	require.NoError(t, err)
	sum = sha256.Sum256(encoded)
	require.NotEqual(t, "v1:"+hex.EncodeToString(sum[:]), response.PricingVersion)
}

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
