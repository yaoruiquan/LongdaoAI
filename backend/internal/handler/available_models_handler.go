package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// AvailableModelsHandler exposes the effective /v1/models list for groups the
// authenticated user is allowed to bind to an API key.
type AvailableModelsHandler struct {
	apiKeyService  *service.APIKeyService
	gatewayService *service.GatewayService
}

func NewAvailableModelsHandler(
	apiKeyService *service.APIKeyService,
	gatewayService *service.GatewayService,
) *AvailableModelsHandler {
	return &AvailableModelsHandler{
		apiKeyService:  apiKeyService,
		gatewayService: gatewayService,
	}
}

type userAvailableModelsGroup struct {
	ID       int64    `json:"id"`
	Name     string   `json:"name"`
	Platform string   `json:"platform"`
	Models   []string `json:"models"`
}

// List returns the same effective model IDs that a key assigned to each group
// receives from the standard GET /v1/models endpoint.
// GET /api/v1/groups/available-models
func (h *AvailableModelsHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	groups, err := h.apiKeyService.GetAvailableGroups(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]userAvailableModelsGroup, 0, len(groups))
	for i := range groups {
		group := &groups[i]
		out = append(out, userAvailableModelsGroup{
			ID:       group.ID,
			Name:     group.Name,
			Platform: group.Platform,
			Models:   h.gatewayService.GetEffectiveModels(c.Request.Context(), group),
		})
	}

	response.Success(c, out)
}
