package service

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// ImageBillingQuote describes sub2api's customer charge, not its supplier cost.
type ImageBillingQuote struct {
	Mode       BillingMode
	Source     string
	Amount     *string
	Multiplier float64
}

func BillingUSDString(amount float64) string {
	return strconv.FormatFloat(amount, 'f', 10, 64)
}

func validImageBillingAmount(amount float64) bool {
	return !math.IsNaN(amount) && !math.IsInf(amount, 0) && amount >= 0
}

// ImageBillingModelAllowed uses the same group-scoped discovery as /v1/models.
func (s *GatewayService) ImageBillingModelAllowed(ctx context.Context, key *APIKey, model string) bool {
	if s == nil || key == nil || key.Group == nil || key.GroupID == nil || *key.GroupID != key.Group.ID {
		return false
	}
	if s.channelService != nil && s.channelService.IsModelRestricted(ctx, *key.GroupID, model) {
		return false
	}
	return customModelsListAllowsModel(s.GetEffectiveModels(ctx, key.Group), model)
}

func (s *OpenAIGatewayService) QuoteImageBilling(ctx context.Context, key *APIKey, model, size string, count int, now time.Time) (*ImageBillingQuote, error) {
	if s == nil || s.billingService == nil || key == nil || key.Group == nil || key.GroupID == nil {
		return nil, errors.New("image billing is unavailable")
	}
	base := s.ResolveUserGroupRateMultiplier(ctx, key.UserID, *key.GroupID, key.Group.RateMultiplier)
	tokenMultiplier, imageMultiplier := computePeakAwareMultipliers(key, base, now)
	quote := &ImageBillingQuote{Mode: BillingModeImage, Source: PricingSourceFallback, Multiplier: imageMultiplier}
	billingModel := model
	if s.channelService != nil {
		mapping := s.channelService.ResolveChannelMapping(ctx, *key.GroupID, model)
		if mapping.BillingModelSource == BillingModelSourceUpstream {
			quote.Source = "upstream_model_unresolved"
			return quote, nil
		}
		if mapping.BillingModelSource == BillingModelSourceChannelMapped && mapping.MappedModel != "" {
			billingModel = mapping.MappedModel
		}
	}
	if resolved := s.resolveOpenAIChannelPricing(ctx, billingModel, key); resolved != nil && resolved.Mode == BillingModeToken {
		quote.Mode, quote.Source, quote.Multiplier = BillingModeToken, PricingSourceChannel, tokenMultiplier
		return quote, nil
	}
	cost, source, explicit := s.imageCostDecision(ctx, billingModel, key, &OpenAIForwardResult{ImageSize: size, ImageCount: count}, imageMultiplier, true)
	quote.Source = source
	if cost != nil {
		quote.Mode = BillingMode(cost.BillingMode)
		if explicit && validImageBillingAmount(cost.ActualCost) && validImageBillingAmount(imageMultiplier) {
			amount := BillingUSDString(cost.ActualCost)
			quote.Amount = &amount
		}
	}
	return quote, nil
}

// Shared by actual image billing and read-only quoting. Quotes never use defaults
// or context-dependent rates whose final input usage is unknown before execution.
func (s *OpenAIGatewayService) imageCostDecision(ctx context.Context, model string, key *APIKey, result *OpenAIForwardResult, multiplier float64, quoteOnly bool) (*CostBreakdown, string, bool) {
	tier := NormalizeImageBillingTierOrDefault(result.ImageSize)
	if !apiKeyHasConfiguredImagePrice(key, tier) {
		key = s.apiKeyWithFreshGroupMediaPricing(ctx, key)
	}
	config := imagePriceConfigFromAPIKey(key)
	if apiKeyHasConfiguredImagePrice(key, tier) {
		price := key.Group.GetImagePrice(tier)
		if quoteOnly && !validImageBillingAmount(*price) {
			return nil, "group", false
		}
		return s.billingService.CalculateImageCost(model, tier, result.ImageCount, config, multiplier), "group", true
	}
	if resolved := s.resolveOpenAIChannelPricing(ctx, model, key); resolved != nil &&
		(resolved.Mode == BillingModePerRequest || resolved.Mode == BillingModeImage) {
		if quoteOnly && !explicitImageChannelPrice(resolved, tier) {
			return &CostBreakdown{BillingMode: string(resolved.Mode)}, PricingSourceChannel, false
		}
		gid := key.Group.ID
		cost, err := s.billingService.CalculateCostUnified(CostInput{Ctx: ctx, Model: model, GroupID: &gid,
			RequestCount: result.ImageCount, SizeTier: tier, RateMultiplier: multiplier, Resolver: s.resolver, Resolved: resolved})
		if err == nil {
			return cost, PricingSourceChannel, true
		}
		logger.LegacyPrintf("service.openai_gateway", "Calculate image channel cost failed: %v", err)
	}
	if quoteOnly {
		return nil, PricingSourceFallback, false
	}
	return s.billingService.CalculateImageCost(model, tier, result.ImageCount, config, multiplier), PricingSourceFallback, false
}

func explicitImageChannelPrice(resolved *ResolvedPricing, tier string) bool {
	for _, interval := range resolved.RequestTiers {
		if interval.TierLabel == tier && interval.PerRequestPrice != nil && *interval.PerRequestPrice != 0 {
			return validImageBillingAmount(*interval.PerRequestPrice)
		}
	}
	return len(resolved.RequestTiers) == 0 && resolved.channelPricing != nil &&
		resolved.channelPricing.PerRequestPrice != nil && validImageBillingAmount(*resolved.channelPricing.PerRequestPrice)
}

// RequestBillingEvidence combines the usage log amount with the billing
// transaction marker. The amount is the customer-facing charge in USD.
type RequestBillingEvidence struct {
	Model                string
	BillingMode          string
	ImageCount           int
	ActualCostUSD        *string
	TransactionConfirmed bool
}

type RequestBillingEvidenceReader interface {
	FindRequestBillingEvidence(context.Context, int64, int64, []string) (*RequestBillingEvidence, error)
}

// RequestBillingIDs accepts a stored ID, a response X-Client-Request-ID (client:),
// or a response X-Request-ID used by legacy local: billing. It never equates two
// unrelated headers. Multiple matches are ambiguous and must remain pending.
func RequestBillingIDs(id string) []string {
	id = strings.TrimSpace(id)
	if strings.HasPrefix(id, "client:") || strings.HasPrefix(id, "local:") || strings.HasPrefix(id, "generated:") {
		return []string{id}
	}
	return []string{id, "client:" + id, "local:" + id}
}

func (s *OpenAIGatewayService) RequestImageBilling(ctx context.Context, key *APIKey, id string) (*RequestBillingEvidence, error) {
	if s == nil || key == nil || key.ID <= 0 || key.UserID <= 0 {
		return nil, errors.New("request billing is unavailable")
	}
	reader, ok := s.usageBillingRepo.(RequestBillingEvidenceReader)
	if !ok {
		return nil, errors.New("request billing is unavailable")
	}
	return reader.FindRequestBillingEvidence(ctx, key.ID, key.UserID, RequestBillingIDs(id))
}
