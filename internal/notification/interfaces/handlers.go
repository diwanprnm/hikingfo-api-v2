// Package interfaces owns the HTTP handlers and route registration for the
// notification bounded context.
package interfaces

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"hikingfo/backend/internal/notification/application"
	plathttp "hikingfo/backend/internal/platform/http"
	"hikingfo/backend/internal/shared/ids"
	"hikingfo/backend/internal/shared/kerr"
)

// Handler holds the notification application service.
type Handler struct{ svc *application.Service }

// NewHandler wires the handler.
func NewHandler(svc *application.Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts notification routes under /api/v1 (all authenticated).
func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	g := rg.Group("/me", plathttp.RequireAuth())
	{
		g.GET("/notifications", h.list)
		g.POST("/notifications/:id/read", h.markRead)
		g.GET("/notifications/unread_count", h.unreadCount)
	}
}

type listReq struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

func (h *Handler) list(c *gin.Context) {
	uid := plathttp.UserID(c)
	var req listReq
	_ = c.ShouldBindQuery(&req)
	items, total, err := h.svc.List(c.Request.Context(), uid, req.Page, req.PageSize)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"total": total,
	})
}

func (h *Handler) markRead(c *gin.Context) {
	uid := plathttp.UserID(c)
	nid, err := ids.Parse(c.Param("id"))
	if err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid notification id"))
		return
	}
	if err := h.svc.MarkRead(c.Request.Context(), uid, nid); err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) unreadCount(c *gin.Context) {
	uid := plathttp.UserID(c)
	count, err := h.svc.UnreadCount(c.Request.Context(), uid)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": count})
}
