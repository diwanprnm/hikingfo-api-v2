// Package interfaces owns the moderation HTTP handlers: public report filing
// and the admin queue (contracts §9).
package interfaces

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"hikingfo/backend/internal/moderation/application"
	"hikingfo/backend/internal/moderation/domain"
	"hikingfo/backend/internal/shared/ids"
	"hikingfo/backend/internal/shared/kerr"
	plathttp "hikingfo/backend/internal/platform/http"
)

// Handler holds the moderation application service.
type Handler struct{ svc *application.Service }

// reportReq is the shared body for report-filing endpoints.
type reportReq struct {
	Reason string `json:"reason" binding:"required"`
	Detail string `json:"detail"`
}

// NewHandler wires the handler.
func NewHandler(svc *application.Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts the public report-filing endpoint.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	g := rg.Group("")
	g.POST("/reports", h.create)

	// Profile reports (public, contracts §5) + blocks (authenticated).
	g.POST("/users/:id/reports", h.reportProfile)
	auth := rg.Group("", plathttp.RequireAuth())
	auth.POST("/users/:id/blocks", h.block)
	auth.DELETE("/users/:id/blocks", h.unblock)
}

// RegisterAdminRoutes mounts the admin queue under /admin (RequireAdmin).
func RegisterAdminRoutes(rg *gin.RouterGroup, h *Handler) {
	g := rg.Group("/admin", plathttp.RequireAdmin())
	{
		g.GET("/queue/reports", h.list)
		g.PATCH("/queue/reports/:id", h.resolve)
		g.GET("/queue/summary", h.summary)
	}
}

// ---- public ----------------------------------------------------------------

type createReq struct {
	TargetType string `json:"target_type" binding:"required"`
	TargetID   string `json:"target_id" binding:"required"`
	FieldRef   string `json:"field_ref"`
	Reason     string `json:"reason" binding:"required"`
	Detail     string `json:"detail"`
}

func (h *Handler) create(c *gin.Context) {
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		plathttp.WriteError(c, kerr.Validation("target_type, target_id and reason are required"))
		return
	}
	if !ids.Valid(req.TargetID) {
		plathttp.WriteError(c, kerr.Validation("invalid target_id"))
		return
	}
	var reporter *ids.ID
	if uid := plathttp.UserID(c); !uid.IsNil() {
		reporter = &uid
	}
	v, err := h.svc.Create(c.Request.Context(), reporter, req.TargetType, req.TargetID, req.FieldRef, req.Reason, req.Detail)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, v)
}

// reportProfile files a user_profile report against {id} (contracts §5).
// Public — anonymous visitors may report, matching POST /reports.
func (h *Handler) reportProfile(c *gin.Context) {
	targetID := c.Param("id")
	if !ids.Valid(targetID) {
		plathttp.WriteError(c, kerr.Validation("invalid user id"))
		return
	}
	var req reportReq
	if err := c.ShouldBindJSON(&req); err != nil {
		plathttp.WriteError(c, kerr.Validation("reason is required"))
		return
	}
	var reporter *ids.ID
	if uid := plathttp.UserID(c); !uid.IsNil() {
		reporter = &uid
	}
	v, err := h.svc.Create(c.Request.Context(), reporter,
		string(domain.TargetUserProfile), targetID, "", req.Reason, req.Detail)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, v)
}

// block records the session user blocking {id}.
func (h *Handler) block(c *gin.Context) {
	blocked, err := ids.Parse(c.Param("id"))
	if err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid user id"))
		return
	}
	if err := h.svc.Block(c.Request.Context(), plathttp.UserID(c), blocked); err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"ok": true, "blocked": string(blocked)})
}

// unblock removes the session user's block on {id}.
func (h *Handler) unblock(c *gin.Context) {
	blocked, err := ids.Parse(c.Param("id"))
	if err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid user id"))
		return
	}
	if err := h.svc.Unblock(c.Request.Context(), plathttp.UserID(c), blocked); err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ---- admin -----------------------------------------------------------------

func (h *Handler) list(c *gin.Context) {
	items, err := h.svc.List(c.Request.Context(), c.Query("status"), 100)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

type resolveReq struct {
	Action     string         `json:"action" binding:"required"`
	Resolution string         `json:"resolution"`
	EditField  map[string]any `json:"edit_field"`
}

func (h *Handler) resolve(c *gin.Context) {
	id, err := ids.Parse(c.Param("id"))
	if err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid report id"))
		return
	}
	var req resolveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		plathttp.WriteError(c, kerr.Validation("action is required"))
		return
	}
	v, err := h.svc.Resolve(c.Request.Context(), id, application.ResolveInput{
		Action:     req.Action,
		Resolution: req.Resolution,
		EditField:  req.EditField,
	})
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) summary(c *gin.Context) {
	counts, err := h.svc.Counts(c.Request.Context())
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"reports": counts})
}
