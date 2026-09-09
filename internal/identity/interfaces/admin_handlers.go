package interfaces

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"hikingfo/backend/internal/shared/ids"
	"hikingfo/backend/internal/shared/kerr"
	plathttp "hikingfo/backend/internal/platform/http"
)

// RegisterAdminRoutes mounts identity admin endpoints under /admin
// (contracts §9 → users, stats).
func RegisterAdminRoutes(rg *gin.RouterGroup, h *Handler) {
	g := rg.Group("/admin", plathttp.RequireAdmin())
	{
		g.GET("/users/:id", h.adminGetUser)
		g.PATCH("/users/:id", h.adminSetUserStatus)
		g.GET("/stats", h.adminStats)
	}
}

type adminStatusReq struct {
	Status string `json:"status" binding:"required"`
}

func (h *Handler) adminGetUser(c *gin.Context) {
	id, err := ids.Parse(c.Param("id"))
	if err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid user id"))
		return
	}
	v, err := h.svc.AdminGetUser(c.Request.Context(), id)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) adminSetUserStatus(c *gin.Context) {
	id, err := ids.Parse(c.Param("id"))
	if err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid user id"))
		return
	}
	var req adminStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		plathttp.WriteError(c, kerr.Validation("status is required"))
		return
	}
	v, err := h.svc.AdminSetUserStatus(c.Request.Context(), id, req.Status)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

// adminStats reports catalogue completeness + report aging (contracts §9).
// The moderation counts come through the handler's stats provider (wired at
// the composition root from the moderation context).
func (h *Handler) adminStats(c *gin.Context) {
	if h.stats == nil {
		plathttp.WriteError(c, kerr.Internal("stats not wired"))
		return
	}
	stats, err := h.stats(c.Request.Context())
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, stats)
}
