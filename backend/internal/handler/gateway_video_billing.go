package handler

import (
	"encoding/json"
	"io"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/Wei-Shaw/sub2api/internal/config"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type videoPricingRequest struct {
	Model           string `json:"model"`
	Operation       string `json:"operation"`
	Resolution      string `json:"resolution"`
	DurationSeconds int    `json:"duration_seconds"`
}

type videoPricingResponse struct {
	Object        string `json:"object"`
	SchemaVersion int    `json:"schema_version"`
	Currency      string `json:"currency"`
	videoPricingRequest
	BillingMode             service.BillingMode `json:"billing_mode"`
	PricingStatus           string              `json:"pricing_status"`
	PricingSource           string              `json:"pricing_source"`
	PricingAuthority        string              `json:"pricing_authority"`
	AmountUSD               *string             `json:"amount_usd"`
	PricingVersion          string              `json:"pricing_version"`
	EffectiveRateMultiplier float64             `json:"effective_rate_multiplier"`
	ObservedAt              time.Time           `json:"observed_at"`
	ExpiresAt               time.Time           `json:"expires_at"`
}

func (h *GatewayHandler) videoBillingKey(c *gin.Context) *service.APIKey {
	key, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok || key == nil || key.ID <= 0 || key.UserID <= 0 {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return nil
	}
	if h.cfg != nil && h.cfg.RunMode == config.RunModeSimple {
		h.errorResponse(c, http.StatusNotFound, "not_found_error", "Billing information is unavailable")
		return nil
	}
	if key.GroupID == nil || key.Group == nil || *key.GroupID != key.Group.ID || key.Group.Platform != service.PlatformOpenAI {
		h.errorResponse(c, http.StatusForbidden, "permission_error", "Video billing is unavailable for this key")
		return nil
	}
	return key
}

func validVideoPricingRequest(request videoPricingRequest) bool {
	return validVideoBillingModel(request.Model) && (request.Operation == "image-to-video" || request.Operation == "text-to-video") &&
		service.ValidVideoBillingResolution(request.Resolution) && request.DurationSeconds > 0 && request.DurationSeconds <= 600
}

func validVideoBillingModel(model string) bool {
	return model != "" && len(model) <= 160 && !strings.ContainsFunc(model, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r) || unicode.Is(unicode.Cf, r)
	})
}

func (h *GatewayHandler) VideoPricing(c *gin.Context) {
	key := h.videoBillingKey(c)
	if key == nil {
		return
	}
	var request videoPricingRequest
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 4096))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil || !validVideoPricingRequest(request) {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Invalid video pricing request")
		return
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Expected one JSON object")
		return
	}
	if !service.GroupAllowsImageGeneration(key.Group) || h.gatewayService == nil || !h.gatewayService.ImageBillingModelAllowed(c.Request.Context(), key, request.Model) {
		h.errorResponse(c, http.StatusForbidden, "permission_error", "Model is not available for this key")
		return
	}
	now := time.Now().UTC()
	quote, err := h.openAIGatewayService.QuoteVideoBilling(c.Request.Context(), key, request.Model, request.Resolution, request.DurationSeconds, now)
	if err != nil || quote == nil || math.IsNaN(quote.Multiplier) || math.IsInf(quote.Multiplier, 0) || quote.Multiplier < 0 {
		h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "Video pricing is unavailable")
		return
	}
	response := videoPricingResponse{
		Object: "sub2api.video_pricing", SchemaVersion: 1, Currency: "USD", videoPricingRequest: request,
		BillingMode: quote.Mode, PricingStatus: "unavailable", PricingSource: quote.Source,
		PricingAuthority: service.PricingAuthoritySub2APICustomerBilling, AmountUSD: quote.Amount,
		EffectiveRateMultiplier: quote.Multiplier, ObservedAt: now, ExpiresAt: now.Add(time.Minute),
	}
	if quote.Amount != nil {
		response.PricingStatus = "quoted"
	}
	response.PricingVersion, err = quote.Version(key, request.Model, request.Operation, request.Resolution, request.DurationSeconds)
	if err != nil {
		h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "Video pricing is unavailable")
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, response)
}

type videoRequestUsageResponse struct {
	Object          string  `json:"object"`
	SchemaVersion   int     `json:"schema_version"`
	Currency        string  `json:"currency"`
	RequestID       string  `json:"request_id"`
	Status          string  `json:"status"`
	Model           *string `json:"model"`
	ActualCostUSD   *string `json:"actual_cost_usd"`
	VideoCount      *int    `json:"video_count"`
	DurationSeconds *int    `json:"duration_seconds"`
	Resolution      *string `json:"resolution"`
}

var videoBillingDecimal = regexp.MustCompile(`^[0-9]+(?:\.[0-9]{1,12})?$`)
var videoBillingRequestID = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,256}$`)
var videoPricingVersion = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,128}$`)

// Legacy callers remain compatible. SEP opts into price protection with both
// headers; only this local, pre-dispatch path may emit the rejection marker.
func (h *OpenAIGatewayHandler) prepareVideoBilling(c *gin.Context, key *service.APIKey, account *service.Account, request *service.OpenAIVideoRequest) (*service.FrozenVideoBilling, bool) {
	versions, operations := c.Request.Header.Values("X-Video-Pricing-Version"), c.Request.Header.Values("X-Video-Operation")
	if len(versions) == 0 && len(operations) == 0 {
		return nil, true
	}
	if len(versions) == 1 && len(operations) == 1 && videoPricingVersion.MatchString(versions[0]) &&
		(operations[0] == "text-to-video" || operations[0] == "image-to-video") {
		frozen, err := h.gatewayService.PrepareVideoBilling(c.Request.Context(), key, request, operations[0], versions[0])
		if err == nil && frozen.MatchesAccount(account) {
			return frozen, true
		}
	}
	c.Header("Cache-Control", "no-store")
	c.Header("X-Video-Dispatch", "rejected")
	c.JSON(http.StatusConflict, gin.H{"error": gin.H{
		"type": "invalid_request_error", "code": "video_pricing_changed",
		"message": "Video pricing or request parameters changed; obtain a new quote before submitting",
	}})
	return nil, false
}

func (h *GatewayHandler) RequestVideoUsage(c *gin.Context) {
	key := h.videoBillingKey(c)
	if key == nil {
		return
	}
	ids := c.QueryArray("request_id")
	if len(ids) != 1 || !videoBillingRequestID.MatchString(ids[0]) {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "A valid request_id is required")
		return
	}
	id := ids[0]
	evidence, err := h.openAIGatewayService.RequestVideoBilling(c.Request.Context(), key, id)
	if err != nil {
		h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "Video request billing is unavailable")
		return
	}
	response := videoRequestUsageResponse{Object: "sub2api.video_request_usage", SchemaVersion: 1, Currency: "USD", RequestID: id, Status: "pending"}
	if evidence != nil {
		// Preserve the strict nullable SEP wire shape even for incomplete or
		// malformed legacy metadata; invalid evidence never confirms a charge.
		if evidence.Model != nil && validVideoBillingModel(*evidence.Model) {
			response.Model = evidence.Model
		}
		if evidence.VideoCount != nil && *evidence.VideoCount >= 0 {
			response.VideoCount = evidence.VideoCount
		}
		if evidence.DurationSeconds != nil && *evidence.DurationSeconds > 0 && *evidence.DurationSeconds <= 600 {
			response.DurationSeconds = evidence.DurationSeconds
		}
		if evidence.Resolution != nil && service.ValidVideoBillingResolution(*evidence.Resolution) {
			response.Resolution = evidence.Resolution
		}
		if reliableVideoBillingEvidence(evidence) {
			response.Status, response.ActualCostUSD = "settled", evidence.ActualCostUSD
		}
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, response)
}

func reliableVideoBillingEvidence(evidence *service.VideoRequestBillingEvidence) bool {
	if evidence == nil || !evidence.TransactionConfirmed || evidence.BillingMode != string(service.BillingModeVideo) ||
		evidence.Model == nil || !validVideoBillingModel(*evidence.Model) || evidence.ActualCostUSD == nil ||
		evidence.VideoCount == nil || *evidence.VideoCount <= 0 || evidence.DurationSeconds == nil || *evidence.DurationSeconds <= 0 ||
		*evidence.DurationSeconds > 600 ||
		evidence.Resolution == nil || !service.ValidVideoBillingResolution(*evidence.Resolution) {
		return false
	}
	if !videoBillingDecimal.MatchString(*evidence.ActualCostUSD) {
		return false
	}
	amount, err := strconv.ParseFloat(*evidence.ActualCostUSD, 64)
	return err == nil && !math.IsNaN(amount) && !math.IsInf(amount, 0) && amount >= 0
}
