package service

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

func videoBillingPtr[T any](value T) *T { return &value }

func newVideoBillingServiceForTest(pricing *ChannelModelPricing) (*OpenAIGatewayService, *APIKey) {
	svc, key := newImageBillingServiceForTest(nil)
	if pricing != nil {
		cache := svc.channelService.cache.Load().(*channelCache)
		cache.pricingByGroupModel[channelModelKey{groupID: *key.GroupID, platform: PlatformOpenAI, model: "minimax-h3"}] = pricing
	}
	return svc, key
}

func TestQuoteVideoBillingMatchesActualVideoCostBranch(t *testing.T) {
	for _, tc := range []struct {
		name       string
		pricing    *ChannelModelPricing
		groupPrice *float64
		base       float64
		userRate   *float64
		videoRate  *float64
		duration   int
		want       float64
		mode       BillingMode
		source     string
	}{
		{name: "group per second", groupPrice: videoBillingPtr(0.0324), base: 1, duration: 4, want: 0.1296, mode: BillingModeVideo, source: "group"},
		{name: "long non grok duration preserved", groupPrice: videoBillingPtr(0.02), base: 1, duration: 30, want: 0.6, mode: BillingModeVideo, source: "group"},
		{name: "user rate overrides group", groupPrice: videoBillingPtr(0.03), base: 2, userRate: videoBillingPtr(0.5), duration: 4, want: 0.06, mode: BillingModeVideo, source: "group"},
		{name: "independent video ignores user image and peak", groupPrice: videoBillingPtr(0.03), base: 2, userRate: videoBillingPtr(0.5), videoRate: videoBillingPtr(0.25), duration: 4, want: 0.03, mode: BillingModeVideo, source: "group"},
		{name: "group overrides request channel", pricing: &ChannelModelPricing{BillingMode: BillingModePerRequest, PerRequestPrice: videoBillingPtr(9.0)}, groupPrice: videoBillingPtr(0.03), base: 1, duration: 4, want: 0.12, mode: BillingModeVideo, source: "group"},
		{name: "request channel not multiplied by duration", pricing: &ChannelModelPricing{BillingMode: BillingModePerRequest, PerRequestPrice: videoBillingPtr(0.3)}, base: 0.5, duration: 4, want: 0.15, mode: BillingModePerRequest, source: PricingSourceChannel},
		{name: "image channel uses request semantics", pricing: &ChannelModelPricing{BillingMode: BillingModeImage, PerRequestPrice: videoBillingPtr(0.4)}, base: 0.5, duration: 4, want: 0.2, mode: BillingModePerRequest, source: PricingSourceChannel},
		{name: "channel size tier", pricing: &ChannelModelPricing{BillingMode: BillingModePerRequest, PerRequestPrice: videoBillingPtr(0.9), Intervals: []PricingInterval{{TierLabel: "720p", PerRequestPrice: videoBillingPtr(0.2)}}}, base: 1, duration: 4, want: 0.2, mode: BillingModePerRequest, source: PricingSourceChannel},
		{name: "explicit zero group", groupPrice: videoBillingPtr(0.0), base: 1, duration: 4, mode: BillingModeVideo, source: "group"},
		{name: "explicit zero channel", pricing: &ChannelModelPricing{BillingMode: BillingModePerRequest, PerRequestPrice: videoBillingPtr(0.0)}, base: 1, duration: 4, mode: BillingModePerRequest, source: PricingSourceChannel},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, key := newVideoBillingServiceForTest(tc.pricing)
			key.Group.RateMultiplier, key.Group.VideoPrice720P = tc.base, tc.groupPrice
			key.Group.PeakRateEnabled, key.Group.PeakStart, key.Group.PeakEnd, key.Group.PeakRateMultiplier = true, "00:00", "23:59", 9
			key.Group.ImageRateIndependent, key.Group.ImageRateMultiplier = true, 17
			if tc.userRate != nil {
				svc.userGroupRateResolver = newUserGroupRateResolver(&userGroupRateResolverRepoStub{rate: tc.userRate}, nil, time.Minute, nil, "test")
			}
			if tc.videoRate != nil {
				key.Group.VideoRateIndependent, key.Group.VideoRateMultiplier = true, *tc.videoRate
			}
			ctx := context.Background()
			quote, err := svc.QuoteVideoBilling(ctx, key, "minimax-h3", "720p", tc.duration, time.Now())
			require.NoError(t, err)
			require.NotNil(t, quote.Amount)
			require.Equal(t, tc.mode, quote.Mode)
			require.Equal(t, tc.source, quote.Source)
			base := svc.ResolveUserGroupRateMultiplier(ctx, key.UserID, *key.GroupID, tc.base)
			videoMultiplier := resolveVideoRateMultiplier(key, base)
			actual, err := svc.calculateOpenAIRecordUsageCost(ctx, &OpenAIForwardResult{
				VideoCount: 1, VideoDurationSeconds: tc.duration, VideoResolution: "720p",
			}, key, []string{"minimax-h3"}, 999, 888, videoMultiplier, base, UsageTokens{InputTokens: 99999}, "", true)
			require.NoError(t, err)
			require.Equal(t, string(BillingModeVideo), actual.BillingMode)
			require.InDelta(t, tc.want, actual.ActualCost, 1e-12)
			require.Equal(t, BillingUSDString(actual.ActualCost), *quote.Amount)
			require.Equal(t, videoMultiplier, quote.Multiplier)
		})
	}
}

func TestQuoteVideoBillingUnavailableWithoutDeterministicConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name    string
		pricing *ChannelModelPricing
		change  func(*OpenAIGatewayService, *APIKey)
		source  string
	}{
		{name: "unknown vendor fallback", source: PricingSourceFallback},
		{name: "known grok fallback still not configured", source: PricingSourceFallback},
		{name: "bare request mode not free", pricing: &ChannelModelPricing{BillingMode: BillingModePerRequest}, source: PricingSourceChannel},
		{name: "token even when group configured", pricing: &ChannelModelPricing{BillingMode: BillingModeToken}, change: func(_ *OpenAIGatewayService, key *APIKey) { key.Group.VideoPrice720P = videoBillingPtr(0.03) }, source: "channel_token_unquotable"},
		{name: "upstream mapping even when group configured", change: func(svc *OpenAIGatewayService, key *APIKey) {
			key.Group.VideoPrice720P = videoBillingPtr(0.03)
			svc.channelService.cache.Load().(*channelCache).channelByGroupID[*key.GroupID].BillingModelSource = BillingModelSourceUpstream
		}, source: "upstream_model_unresolved"},
		{name: "wrong resolution group price", change: func(_ *OpenAIGatewayService, key *APIKey) { key.Group.VideoPrice480P = videoBillingPtr(0.03) }, source: PricingSourceFallback},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, key := newVideoBillingServiceForTest(tc.pricing)
			if tc.change != nil {
				tc.change(svc, key)
			}
			model := "minimax-h3"
			if tc.name == "known grok fallback still not configured" {
				model = "grok-imagine-video"
			}
			quote, err := svc.QuoteVideoBilling(context.Background(), key, model, "720p", 4, time.Now())
			require.NoError(t, err)
			require.Nil(t, quote.Amount)
			require.Equal(t, tc.source, quote.Source)
		})
	}
}

func TestQuoteVideoBillingRejectsInvalidMoney(t *testing.T) {
	for _, value := range []float64{-0.1, math.NaN(), math.Inf(1), math.MaxFloat64} {
		for _, source := range []string{"group", "channel", "channel tier", "multiplier"} {
			t.Run(source+"/"+BillingUSDString(value), func(t *testing.T) {
				svc, key := newVideoBillingServiceForTest(nil)
				key.Group.VideoPrice720P = videoBillingPtr(0.1)
				switch source {
				case "group":
					key.Group.VideoPrice720P = &value
				case "channel", "channel tier":
					pricing := &ChannelModelPricing{BillingMode: BillingModePerRequest, PerRequestPrice: &value}
					if source == "channel tier" {
						pricing.Intervals = []PricingInterval{{TierLabel: "720p", PerRequestPrice: &value}}
					}
					svc, key = newVideoBillingServiceForTest(pricing)
					if value == math.MaxFloat64 {
						key.Group.RateMultiplier = 2
					}
				case "multiplier":
					key.Group.RateMultiplier = value
					key.Group.VideoPrice720P = videoBillingPtr(2.0)
				}
				if source != "multiplier" && value < 0 {
					key.Group.RateMultiplier = 0
				}
				quote, err := svc.QuoteVideoBilling(context.Background(), key, "minimax-h3", "720p", 4, time.Now())
				require.NoError(t, err)
				require.Nil(t, quote.Amount)
			})
		}
	}
}

func TestQuoteVideoBillingMappingAndDurationFollowActual(t *testing.T) {
	svc, key := newVideoBillingServiceForTest(nil)
	key.Group.VideoPrice720P = videoBillingPtr(0.03)
	cache := svc.channelService.cache.Load().(*channelCache)
	cache.channelByGroupID[*key.GroupID].BillingModelSource = BillingModelSourceChannelMapped
	cache.mappingByGroupModel[channelModelKey{groupID: *key.GroupID, platform: PlatformOpenAI, model: "video-alias"}] = "grok-imagine-video"
	quote, err := svc.QuoteVideoBilling(context.Background(), key, "video-alias", "720p", 4, time.Now())
	require.NoError(t, err)
	require.Equal(t, "0.1200000000", *quote.Amount)
	quote, err = svc.QuoteVideoBilling(context.Background(), key, "video-alias", "720p", 30, time.Now())
	require.NoError(t, err)
	require.Nil(t, quote.Amount)
	require.Equal(t, "duration_unresolved", quote.Source)
}

type videoBillingUnavailableChannelRepo struct{ ChannelRepository }

func (*videoBillingUnavailableChannelRepo) ListAll(context.Context) ([]Channel, error) {
	return nil, errors.New("channel storage unavailable")
}

func TestQuoteVideoBillingMappingFailureAndHydration(t *testing.T) {
	svc, key := newVideoBillingServiceForTest(nil)
	key.Group.VideoPrice720P = videoBillingPtr(0.03)
	svc.channelService = NewChannelService(&videoBillingUnavailableChannelRepo{}, nil, nil, nil)
	quote, err := svc.QuoteVideoBilling(context.Background(), key, "minimax-h3", "720p", 4, time.Now())
	require.NoError(t, err)
	require.Nil(t, quote.Amount)
	require.Equal(t, "model_mapping_unavailable", quote.Source)
	quote, err = svc.QuoteVideoBilling(context.Background(), key, "minimax-h3", "720p", 4, time.Now())
	require.NoError(t, err)
	require.Nil(t, quote.Amount, "cached storage failure must not become a trusted group quote")
	require.Equal(t, "model_mapping_unavailable", quote.Source)

	svc, key = newVideoBillingServiceForTest(nil)
	key.Group.ImageRateMultiplier, key.Group.VideoRateMultiplier = 0, 0
	svc.channelService.groupRepo = &openAIMediaPriceGroupRepoStub{group: &Group{
		ID: *key.GroupID, Platform: PlatformOpenAI, RateMultiplier: 1, VideoPrice720P: videoBillingPtr(0.03),
	}}
	quote, err = svc.QuoteVideoBilling(context.Background(), key, "minimax-h3", "720p", 4, time.Now())
	require.NoError(t, err)
	require.NotNil(t, quote.Amount)
	require.Equal(t, "0.1200000000", *quote.Amount)
	require.Equal(t, "group", quote.Source)
}

type videoEvidenceReaderStub struct {
	UsageBillingRepository
	keyID, userID int64
	ids           []string
	err           error
}

func (r *videoEvidenceReaderStub) FindVideoRequestBillingEvidence(_ context.Context, keyID, userID int64, ids []string) (*VideoRequestBillingEvidence, error) {
	r.keyID, r.userID, r.ids = keyID, userID, ids
	return nil, r.err
}

func TestRequestVideoBillingScopedAliasesAndReaderFailure(t *testing.T) {
	reader := &videoEvidenceReaderStub{}
	svc := &OpenAIGatewayService{usageBillingRepo: reader}
	key := &APIKey{ID: 7, UserID: 11}
	for id, want := range map[string][]string{
		"req-1": {"req-1", "client:req-1", "local:req-1"}, "client:req-1": {"client:req-1"},
		"local:req-1": {"local:req-1"}, "generated:req-1": {"generated:req-1"},
	} {
		_, err := svc.RequestVideoBilling(context.Background(), key, id)
		require.NoError(t, err)
		require.Equal(t, int64(7), reader.keyID)
		require.Equal(t, int64(11), reader.userID)
		require.Equal(t, want, reader.ids)
	}
	reader.err = errors.New("database unavailable")
	_, err := svc.RequestVideoBilling(context.Background(), key, "req-1")
	require.ErrorIs(t, err, reader.err)
	_, err = (&OpenAIGatewayService{}).RequestVideoBilling(context.Background(), key, "req-1")
	require.Error(t, err)
}

func TestPrepareVideoBillingVersionInvalidation(t *testing.T) {
	for _, change := range []string{"price", "rate", "mapping", "operation", "duration", "resolution", "key", "unconfigured"} {
		t.Run(change, func(t *testing.T) {
			svc, key := newVideoBillingServiceForTest(nil)
			key.Group.VideoPrice720P = videoBillingPtr(0.03)
			request := &OpenAIVideoRequest{Model: "minimax-h3", Resolution: "720p", Seconds: 4}
			quote, err := svc.QuoteVideoBilling(context.Background(), key, request.Model, request.Resolution, request.Seconds, time.Now())
			require.NoError(t, err)
			version, err := quote.Version(key, request.Model, "text-to-video", request.Resolution, request.Seconds)
			require.NoError(t, err)
			require.Regexp(t, `^[A-Za-z0-9_.:-]{1,128}$`, version)
			_, err = svc.PrepareVideoBilling(context.Background(), key, request, "text-to-video", version)
			require.NoError(t, err)
			operation := "text-to-video"
			switch change {
			case "price":
				key.Group.VideoPrice720P = videoBillingPtr(0.04)
			case "rate":
				key.Group.RateMultiplier = 2
			case "mapping":
				cache := svc.channelService.cache.Load().(*channelCache)
				cache.mappingByGroupModel[channelModelKey{groupID: *key.GroupID, platform: PlatformOpenAI, model: request.Model}] = "other-video"
			case "operation":
				operation = "image-to-video"
			case "duration":
				request.Seconds = 6
			case "resolution":
				request.Resolution = "1080p"
			case "key":
				key.ID++
			case "unconfigured":
				key.Group.VideoPrice720P = nil
			}
			_, err = svc.PrepareVideoBilling(context.Background(), key, request, operation, version)
			require.ErrorIs(t, err, ErrVideoPricingChanged)
		})
	}
}

func TestPrepareVideoBillingReadsCurrentStorageNotCaches(t *testing.T) {
	svc, key := newVideoBillingServiceForTest(nil)
	group := *key.Group
	group.VideoPrice720P = videoBillingPtr(0.03)
	svc.channelService.groupRepo = &openAIMediaPriceGroupRepoStub{group: &group}
	quote, err := svc.QuoteVideoBilling(context.Background(), key, "minimax-h3", "720p", 4, time.Now())
	require.NoError(t, err)
	require.Equal(t, "0.1200000000", *quote.Amount)
	version, err := quote.Version(key, "minimax-h3", "text-to-video", "720p", 4)
	require.NoError(t, err)
	group.VideoPrice720P = videoBillingPtr(0.09) // auth snapshot remains unchanged
	_, err = svc.PrepareVideoBilling(context.Background(), key, &OpenAIVideoRequest{Model: "minimax-h3", Resolution: "720p", Seconds: 4}, "text-to-video", version)
	require.ErrorIs(t, err, ErrVideoPricingChanged)

	rate := 0.5
	svc.userGroupRateResolver = newUserGroupRateResolver(&userGroupRateResolverRepoStub{rate: &rate}, nil, time.Minute, nil, "test")
	quote, err = svc.QuoteVideoBilling(context.Background(), key, "minimax-h3", "720p", 4, time.Now())
	require.NoError(t, err)
	version, err = quote.Version(key, "minimax-h3", "text-to-video", "720p", 4)
	require.NoError(t, err)
	rate = 2
	_, err = svc.PrepareVideoBilling(context.Background(), key, &OpenAIVideoRequest{Model: "minimax-h3", Resolution: "720p", Seconds: 4}, "text-to-video", version)
	require.ErrorIs(t, err, ErrVideoPricingChanged)
}

func TestFrozenVideoBillingRecordUsageCannotRepriceAfterDispatch(t *testing.T) {
	for _, mode := range []string{"group", "channel"} {
		t.Run(mode, func(t *testing.T) {
			pricing := &ChannelModelPricing{BillingMode: BillingModePerRequest, PerRequestPrice: videoBillingPtr(0.12)}
			svc, key := newVideoBillingServiceForTest(pricing)
			if mode == "group" {
				key.Group.VideoPrice720P = videoBillingPtr(0.03)
			}
			request := &OpenAIVideoRequest{Model: "minimax-h3", Resolution: "720p", Seconds: 4}
			quote, err := svc.QuoteVideoBilling(context.Background(), key, request.Model, request.Resolution, request.Seconds, time.Now())
			require.NoError(t, err)
			version, err := quote.Version(key, request.Model, "text-to-video", "720p", 4)
			require.NoError(t, err)
			frozen, err := svc.PrepareVideoBilling(context.Background(), key, request, "text-to-video", version)
			require.NoError(t, err)
			key.Group.VideoPrice720P, key.Group.RateMultiplier = videoBillingPtr(99.0), 99
			pricing.PerRequestPrice = videoBillingPtr(99.0)
			usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
			billingRepo := &openAIRecordUsageBillingRepoStub{}
			svc.usageLogRepo, svc.usageBillingRepo = usageRepo, billingRepo
			svc.billingCacheService, svc.deferredService = &BillingCacheService{}, &DeferredService{}
			input := &OpenAIRecordUsageInput{APIKey: key, User: &User{ID: key.UserID}, Account: &Account{ID: 9, Type: AccountTypeAPIKey}, FrozenVideoBilling: frozen,
				Result:             &OpenAIForwardResult{Model: request.Model, VideoCount: 1, VideoResolution: "1080p", VideoDurationSeconds: 600},
				ChannelUsageFields: frozen.Mapping().ToUsageFields(request.Model, request.Model)}
			ctx := context.WithValue(context.Background(), ctxkey.ClientRequestID, "relay-original-id")
			require.NoError(t, svc.RecordUsage(ctx, input))
			require.Equal(t, 1, billingRepo.calls)
			require.InDelta(t, 0.12, billingRepo.lastCmd.BalanceCost, 1e-12)
			require.Equal(t, "client:relay-original-id", billingRepo.lastCmd.RequestID)
			require.Equal(t, billingRepo.lastCmd.RequestID, usageRepo.lastLog.RequestID)
			require.Equal(t, *quote.Amount, BillingUSDString(usageRepo.lastLog.ActualCost))
			require.Equal(t, string(BillingModeVideo), *usageRepo.lastLog.BillingMode)
			require.Equal(t, 1.0, usageRepo.lastLog.RateMultiplier)
			input.APIKey = &APIKey{ID: 999, UserID: key.UserID, GroupID: key.GroupID, Group: key.Group}
			require.Error(t, svc.RecordUsage(ctx, input))
			require.Equal(t, 1, billingRepo.calls, "mismatched ownership must not bill")
		})
	}
}
