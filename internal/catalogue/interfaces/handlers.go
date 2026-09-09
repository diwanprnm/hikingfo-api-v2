// Package interfaces owns the HTTP handlers and route registration for the
// catalogue bounded context.
package interfaces

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"hikingfo/backend/internal/catalogue/application"
	"hikingfo/backend/internal/catalogue/domain"
	plathttp "hikingfo/backend/internal/platform/http"
	"hikingfo/backend/internal/shared/ids"
	"hikingfo/backend/internal/shared/kerr"
)

// Handler holds the catalogue application service.
type Handler struct {
	svc *application.Service
	// fieldReport writes FR-003 field-error reports into the moderation
	// context (composition-root supplied closure). Nil → acknowledges only.
	fieldReport func(ctx context.Context, mountainID ids.ID, field, reason, detail string) error
}

// NewHandler wires the handler.
func NewHandler(svc *application.Service) *Handler {
	return &Handler{svc: svc}
}

// SetFieldReport wires the moderation-context report writer (T033/T034 —
// "report error" on a mountain field). Called by the composition root.
func (h *Handler) SetFieldReport(fn func(ctx context.Context, mountainID ids.ID, field, reason, detail string) error) {
	h.fieldReport = fn
}

// RegisterRoutes mounts all catalogue routes under /api/v1 (all public).
func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	g := rg.Group("")
	{
		g.GET("/mountains", h.search)
		g.GET("/regions", h.regions)
		g.GET("/mountains/:slug", h.profile)
		g.GET("/mountains/:slug/weather", h.weather)
		g.POST("/mountains/:id/fields/:field/reports", h.reportField)
	}
}

// ---- search & browse ------------------------------------------------------

type searchReq struct {
	Query      string `form:"q"`
	Region     string `form:"region"`
	Province   string `form:"province"`
	Difficulty int    `form:"difficulty"`
	MinHeight  int    `form:"min_height"`
	MaxHeight  int    `form:"max_height"`
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
}

func (h *Handler) search(c *gin.Context) {
	var req searchReq
	_ = c.ShouldBindQuery(&req)

	f := domain.SearchFilter{
		Query:      req.Query,
		Region:     domain.Region(req.Region),
		Province:   req.Province,
		Difficulty: req.Difficulty,
		MinHeight:  req.MinHeight,
		MaxHeight:  req.MaxHeight,
		Page:       req.Page,
		PageSize:   req.PageSize,
	}
	result, err := h.svc.SearchAndFilter(c.Request.Context(), f)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) regions(c *gin.Context) {
	regions, err := h.svc.Regions(c.Request.Context())
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": regions})
}

// ---- mountain profile -----------------------------------------------------

func (h *Handler) profile(c *gin.Context) {
	slug := c.Param("slug")
	p, err := h.svc.GetProfile(c.Request.Context(), slug)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

// ---- weather --------------------------------------------------------------

func (h *Handler) weather(c *gin.Context) {
	slug := c.Param("slug")
	w, err := h.svc.GetWeather(c.Request.Context(), slug)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, w)
}

// ---- field report (any visitor) -------------------------------------------

type reportFieldReq struct {
	Reason string `json:"reason" binding:"required"`
	Detail string `json:"detail"`
}

func (h *Handler) reportField(c *gin.Context) {
	mountainID, err := ids.Parse(c.Param("id"))
	if err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid mountain id"))
		return
	}
	field := c.Param("field")
	var req reportFieldReq
	if err := c.ShouldBindJSON(&req); err != nil {
		plathttp.WriteError(c, kerr.Validation("reason is required"))
		return
	}
	// FR-003: field-error reports land in the moderation queue as
	// mountain_field reports so the admin can edit + annotate provenance.
	if h.fieldReport != nil {
		if err := h.fieldReport(c.Request.Context(), mountainID, field, req.Reason, req.Detail); err != nil {
			plathttp.WriteError(c, err)
			return
		}
	}
	c.JSON(http.StatusAccepted, gin.H{"ok": true, "status": "submitted"})
}

// ---- helpers --------------------------------------------------------------
