package interfaces

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"hikingfo/backend/internal/achievement/application"
	plathttp "hikingfo/backend/internal/platform/http"
	"hikingfo/backend/internal/shared/kerr"
)

// RegisterAdminRoutes mounts badge-config admin endpoints under /admin
// (contracts §9).
func RegisterAdminRoutes(rg *gin.RouterGroup, h *Handler) {
	g := rg.Group("/admin", plathttp.RequireAdmin())
	{
		g.GET("/badge-configs", h.adminListBadges)
		g.POST("/badge-configs", h.adminUpsertBadge)
		g.PATCH("/badge-configs/:key", h.adminUpdateBadge)
	}
}

func (h *Handler) adminListBadges(c *gin.Context) {
	items, err := h.svc.AdminAll(c.Request.Context())
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) adminUpsertBadge(c *gin.Context) {
	var req application.BadgeConfigInput
	if err := c.ShouldBindJSON(&req); err != nil {
		plathttp.WriteError(c, kerr.Validation("key and threshold are required"))
		return
	}
	bc, err := h.svc.AdminUpsertBadge(c.Request.Context(), req)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusCreated, bc)
}

func (h *Handler) adminUpdateBadge(c *gin.Context) {
	var req application.BadgeConfigInput
	if err := c.ShouldBindJSON(&req); err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid request body"))
		return
	}
	req.Key = c.Param("key") // path wins over body
	bc, err := h.svc.AdminUpsertBadge(c.Request.Context(), req)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, bc)
}
