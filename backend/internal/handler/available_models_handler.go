package handler

import (
	"strconv"

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

	out := h.buildGroupsResponse(c, groups)

	response.Success(c, out)
}

// Refresh invalidates and recalculates one group without changing its account
// model configuration. The group must be available to the current user.
// POST /api/v1/groups/available-models/:id/refresh
func (h *AvailableModelsHandler) Refresh(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || groupID <= 0 {
		response.BadRequest(c, "Invalid group ID")
		return
	}

	groups, err := h.apiKeyService.GetAvailableGroups(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	for i := range groups {
		if groups[i].ID != groupID {
			continue
		}

		h.gatewayService.InvalidateAvailableModelsCache(&groupID, groups[i].Platform)
		response.Success(c, h.buildGroupResponse(c, &groups[i]))
		return
	}

	// Do not reveal whether an inaccessible group exists.
	response.NotFound(c, "Group not found")
}

func (h *AvailableModelsHandler) buildGroupsResponse(c *gin.Context, groups []service.Group) []userAvailableModelsGroup {
	out := make([]userAvailableModelsGroup, 0, len(groups))
	for i := range groups {
		out = append(out, h.buildGroupResponse(c, &groups[i]))
	}
	return out
}

func (h *AvailableModelsHandler) buildGroupResponse(c *gin.Context, group *service.Group) userAvailableModelsGroup {
	return userAvailableModelsGroup{
		ID:       group.ID,
		Name:     group.Name,
		Platform: group.Platform,
		Models:   h.gatewayService.GetEffectiveModels(c.Request.Context(), group),
	}
}
