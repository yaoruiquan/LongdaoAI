package service

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newImageBillingServiceForTest(pricing *ChannelModelPricing) (*OpenAIGatewayService, *APIKey) {
	groupID := int64(100)
	key := &APIKey{ID: 7, UserID: 11, GroupID: &groupID, Group: &Group{
		ID: groupID, Platform: PlatformOpenAI, RateMultiplier: 1, ImageRateMultiplier: 1, VideoRateMultiplier: 1,
	}}
	bs := &BillingService{}
	cache := newEmptyChannelCache()
	cache.loadedAt = time.Now()
	cache.groupPlatform[groupID] = PlatformOpenAI
	cache.channelByGroupID[groupID] = &Channel{ID: 1, Status: StatusActive, BillingModelSource: BillingModelSourceRequested}
	if pricing != nil {
		cache.pricingByGroupModel[channelModelKey{groupID: groupID, platform: PlatformOpenAI, model: "gpt-image-2"}] = pricing
	}
	cs := &ChannelService{}
	cs.cache.Store(cache)
	return &OpenAIGatewayService{billingService: bs, resolver: NewModelPricingResolver(cs, bs), channelService: cs}, key
}

func requireImageQuoteMatchesActual(t *testing.T, svc *OpenAIGatewayService, key *APIKey, size string, count int, billingModel, source string, want float64) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	quote, err := svc.QuoteImageBilling(ctx, key, "gpt-image-2", size, count, now)
	require.NoError(t, err)
	require.NotNil(t, quote.Amount)
	require.Equal(t, source, quote.Source)
	require.Equal(t, PricingAuthoritySub2APICustomerBilling, quote.PricingAuthority)
	base := svc.ResolveUserGroupRateMultiplier(ctx, key.UserID, *key.GroupID, key.Group.RateMultiplier)
	tokenMultiplier, imageMultiplier := computePeakAwareMultipliers(key, base, now)
	// Even supplied token usage is ignored by the actual image/per-request path.
	actual, err := svc.calculateOpenAIRecordUsageCost(ctx, &OpenAIForwardResult{ImageSize: size, ImageCount: count}, key,
		[]string{billingModel}, tokenMultiplier, imageMultiplier, imageMultiplier, base,
		UsageTokens{InputTokens: 50000}, "", true)
	require.NoError(t, err)
	require.InDelta(t, want, actual.ActualCost, 1e-12)
	require.Equal(t, BillingUSDString(actual.ActualCost), *quote.Amount)
	require.Equal(t, BillingMode(actual.BillingMode), quote.Mode)
}

func TestQuoteImageBillingDefaultMatchesActual(t *testing.T) {
	for _, tt := range []struct {
		size string
		unit float64
	}{
		{"1024x1024", 0.134},
		{"2048x1152", 0.134 * 1.5},
		{"3840x2160", 0.134 * 2},
	} {
		t.Run(tt.size, func(t *testing.T) {
			svc, key := newImageBillingServiceForTest(nil)
			key.Group.RateMultiplier = 0.75
			requireImageQuoteMatchesActual(t, svc, key, tt.size, 3, "gpt-image-2", PricingSourceFallback, tt.unit*3*0.75)
		})
	}
}

func TestQuoteImageBillingCommercialDefaultMatchesActual(t *testing.T) {
	svc, key := newImageBillingServiceForTest(nil)
	delete(svc.channelService.cache.Load().(*channelCache).channelByGroupID, *key.GroupID)
	key.Group.ImageRateIndependent, key.Group.ImageRateMultiplier = true, 2.19
	requireImageQuoteMatchesActual(t, svc, key, "2048x2048", 1, "gpt-image-2", PricingSourceFallback, 0.134*1.5*2.19)
}

func TestQuoteImageBillingGroupOverridesChannelAndPreservesMultipliers(t *testing.T) {
	groupPrice, channelPrice := 0.02, 0.9
	for _, tt := range []struct {
		name        string
		independent bool
		userRate    *float64
		wantRate    float64
	}{
		{name: "group base ignores peak", wantRate: 0.75},
		{name: "user overrides group", userRate: imageBillingFloatPtr(0.5), wantRate: 0.5},
		{name: "independent image overrides user", independent: true, userRate: imageBillingFloatPtr(0.5), wantRate: 0.25},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc, key := newImageBillingServiceForTest(&ChannelModelPricing{BillingMode: BillingModeImage, PerRequestPrice: &channelPrice})
			key.Group.RateMultiplier = 0.75
			key.Group.ImagePrice2K = &groupPrice
			key.Group.ImageRateIndependent, key.Group.ImageRateMultiplier = tt.independent, 0.25
			key.Group.SubscriptionType = SubscriptionTypeSubscription
			key.Group.PeakRateEnabled, key.Group.PeakStart, key.Group.PeakEnd, key.Group.PeakRateMultiplier = true, "00:00", "23:59", 10
			if tt.userRate != nil {
				svc.userGroupRateResolver = newUserGroupRateResolver(&userGroupRateResolverRepoStub{rate: tt.userRate}, nil, time.Minute, nil, "test")
			}
			requireImageQuoteMatchesActual(t, svc, key, "2048x2048", 2, "gpt-image-2", "group", groupPrice*2*tt.wantRate)
		})
	}
}

func imageBillingFloatPtr(value float64) *float64 { return &value }

func TestQuoteImageBillingChannelTierFallbackMatchesActual(t *testing.T) {
	for _, mode := range []BillingMode{BillingModeImage, BillingModePerRequest} {
		for _, tt := range []struct {
			name    string
			pricing ChannelModelPricing
			want    float64
		}{
			{name: "flat", pricing: ChannelModelPricing{PerRequestPrice: imageBillingFloatPtr(0.3)}, want: 0.3},
			{name: "matched tier overrides default", pricing: ChannelModelPricing{PerRequestPrice: imageBillingFloatPtr(0.3), Intervals: []PricingInterval{{TierLabel: "2K", PerRequestPrice: imageBillingFloatPtr(0.2)}}}, want: 0.2},
			{name: "missing tier uses channel default", pricing: ChannelModelPricing{PerRequestPrice: imageBillingFloatPtr(0.3), Intervals: []PricingInterval{{TierLabel: "1K", PerRequestPrice: imageBillingFloatPtr(0.2)}}}, want: 0.3},
			{name: "zero tier uses channel default", pricing: ChannelModelPricing{PerRequestPrice: imageBillingFloatPtr(0.3), Intervals: []PricingInterval{{TierLabel: "2K", PerRequestPrice: imageBillingFloatPtr(0)}}}, want: 0.3},
			{name: "missing tier without default is free", pricing: ChannelModelPricing{Intervals: []PricingInterval{{TierLabel: "1K", PerRequestPrice: imageBillingFloatPtr(0.2)}}}, want: 0},
			{name: "missing price is free", want: 0},
			{name: "context interval does not use upstream tokens", pricing: ChannelModelPricing{PerRequestPrice: imageBillingFloatPtr(0.3), Intervals: []PricingInterval{{MinTokens: 0, PerRequestPrice: imageBillingFloatPtr(0.9)}}}, want: 0.3},
		} {
			t.Run(string(mode)+"/"+tt.name, func(t *testing.T) {
				tt.pricing.BillingMode = mode
				svc, key := newImageBillingServiceForTest(&tt.pricing)
				key.Group.RateMultiplier = 0.5
				requireImageQuoteMatchesActual(t, svc, key, "2K", 2, "gpt-image-2", PricingSourceChannel, tt.want)
			})
		}
	}
}

func TestQuoteImageBillingRejectsInvalidAmounts(t *testing.T) {
	for _, value := range []float64{-0.1, math.NaN(), math.Inf(1), math.MaxFloat64} {
		for _, source := range []string{"group", "channel", "channel_tier", "multiplier"} {
			t.Run(source+"/"+BillingUSDString(value), func(t *testing.T) {
				svc, key := newImageBillingServiceForTest(nil)
				wantSource := PricingSourceFallback
				switch source {
				case "group":
					key.Group.ImagePrice2K, wantSource = &value, "group"
				case "channel", "channel_tier":
					pricing := &ChannelModelPricing{BillingMode: BillingModeImage, PerRequestPrice: &value}
					if source == "channel_tier" {
						pricing.PerRequestPrice = imageBillingFloatPtr(0.1)
						pricing.Intervals = []PricingInterval{{TierLabel: "2K", PerRequestPrice: &value}}
					}
					svc, key = newImageBillingServiceForTest(pricing)
					wantSource = PricingSourceChannel
				case "multiplier":
					key.Group.RateMultiplier = value
				}
				if source != "multiplier" && value < 0 {
					key.Group.RateMultiplier = 0 // Invalid prices must not become trusted negative zero.
				}
				if source == "multiplier" && value == math.MaxFloat64 {
					key.Group.ImagePrice2K = imageBillingFloatPtr(2)
					wantSource = "group"
				}
				quote, err := svc.QuoteImageBilling(context.Background(), key, "gpt-image-2", "2K", 4, time.Now())
				require.NoError(t, err)
				require.Equal(t, wantSource, quote.Source)
				require.Nil(t, quote.Amount)
				require.Empty(t, quote.PricingAuthority)
			})
		}
	}
}

func TestQuoteImageBillingRejectsUnresolvedAndTokenBilling(t *testing.T) {
	for _, tt := range []struct {
		name   string
		mode   BillingMode
		source string
	}{
		{name: "token", mode: BillingModeToken, source: PricingSourceChannel},
		{name: "upstream", mode: BillingModeImage, source: "upstream_model_unresolved"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc, key := newImageBillingServiceForTest(&ChannelModelPricing{BillingMode: tt.mode})
			key.Group.ImagePrice2K = imageBillingFloatPtr(0.02)
			if tt.name == "upstream" {
				svc.channelService.cache.Load().(*channelCache).channelByGroupID[*key.GroupID].BillingModelSource = BillingModelSourceUpstream
			}
			quote, err := svc.QuoteImageBilling(context.Background(), key, "gpt-image-2", "2K", 1, time.Now())
			require.NoError(t, err)
			require.Equal(t, tt.mode, quote.Mode)
			require.Equal(t, tt.source, quote.Source)
			require.Nil(t, quote.Amount)
			require.Empty(t, quote.PricingAuthority)
		})
	}
}

func TestQuoteImageBillingChannelMappedDefaultMatchesActual(t *testing.T) {
	svc, key := newImageBillingServiceForTest(nil)
	cache := svc.channelService.cache.Load().(*channelCache)
	cache.channelByGroupID[*key.GroupID].BillingModelSource = BillingModelSourceChannelMapped
	cache.mappingByGroupModel[channelModelKey{groupID: *key.GroupID, platform: PlatformOpenAI, model: "gpt-image-2"}] = "grok-imagine-image"
	want := svc.billingService.CalculateImageCost("grok-imagine-image", "2K", 2, nil, 1).ActualCost
	requireImageQuoteMatchesActual(t, svc, key, "2K", 2, "grok-imagine-image", PricingSourceFallback, want)
}
