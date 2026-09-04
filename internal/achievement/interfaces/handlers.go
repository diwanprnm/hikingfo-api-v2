// Package interfaces owns the HTTP handlers and route registration for the
// achievement bounded context.
package interfaces

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"hikingfo/backend/internal/achievement/application"
	plathttp "hikingfo/backend/internal/platform/http"
)

// Handler holds the achievement application service.
type Handler struct{ svc *application.Service }

// NewHandler wires the handler.
func NewHandler(svc *application.Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts all achievement routes under /api/v1.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	g := rg.Group("/me", plathttp.RequireAuth())
	{
		g.GET("/badges", h.getBadges)
	}
}

func (h *Handler) getBadges(c *gin.Context) {
	userID := plathttp.UserID(c)
	view, err := h.svc.GetBadges(c.Request.Context(), userID)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}
