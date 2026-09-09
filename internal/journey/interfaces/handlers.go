// Package interfaces owns the HTTP handlers and route registration for the
// journey bounded context.
package interfaces

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"hikingfo/backend/internal/journey/application"
	"hikingfo/backend/internal/journey/domain"
	plathttp "hikingfo/backend/internal/platform/http"
	"hikingfo/backend/internal/platform/storage"
	"hikingfo/backend/internal/shared/ids"
	"hikingfo/backend/internal/shared/kerr"
)

// Handler holds the journey application service.
type Handler struct {
	svc   *application.Service
	blobs storage.BlobStore // 002 §8: real POST /uploads pipeline
}

// NewHandler wires the handler. blobs is optional (nil → placeholder upload mode).
func NewHandler(svc *application.Service, blobs ...storage.BlobStore) *Handler {
	h := &Handler{svc: svc}
	if len(blobs) > 0 {
		h.blobs = blobs[0]
	}
	return h
}

// RegisterRoutes mounts all journey routes under the given group.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	auth := rg.Group("", plathttp.RequireAuth())

	// §4 Journey / hike log (authenticated)
	auth.POST("/hikes", h.recordHike)
	auth.GET("/me/hikes", h.listMyHikes)
	auth.GET("/me/hikes/:id", h.getMyHike)
	auth.DELETE("/me/hikes/:id", h.deleteMyHike)
	auth.POST("/me/journeys", h.createPost)
	auth.PATCH("/me/journeys/:id", h.updatePost)
	auth.POST("/me/journeys/:id/publish", h.publishPost)
	auth.DELETE("/me/journeys/:id", h.deletePost)

	// §8 Community feed (authenticated — 004: signed-in users only).
	// GET /mountains/:slug/journeys is the same feed scoped to one mountain
	// (contracts §2) — resolved to an ID here.
	auth.GET("/journeys", h.listFeed)
	auth.GET("/mountains/:slug/journeys", h.listMountainJourneys)
	auth.GET("/journeys/:id", h.getFeedItem)
	auth.POST("/journeys/:id/reports", h.reportPost)

	// §8 Uploads (authenticated, placeholder) — upload endpoints are
	// abuse-sensitive, throttle per IP.
	auth.POST("/uploads", plathttp.RateLimiter(0.2, 5), h.upload)
}

// ---- §4 Hike log ----------------------------------------------------------

type recordHikeReq struct {
	MountainID        string   `json:"mountain_id" binding:"required"`
	RouteID           *string  `json:"route_id"`
	ClimbDate         string   `json:"climb_date" binding:"required"`
	EvidencePhotoKeys []string `json:"evidence_photo_keys"`
	Title             *string  `json:"title"`
	Summary           *string  `json:"summary"`
	Narrative         *string  `json:"narrative"`
	PhotoKeys         *[]string `json:"photo_keys"`
	Visibility        *string  `json:"visibility"`
}

func (h *Handler) recordHike(c *gin.Context) {
	var req recordHikeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid request body"))
		return
	}

	userID := plathttp.UserID(c)
	mountainID, err := parseID(req.MountainID)
	if err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid mountain_id"))
		return
	}

	var routeID *ids.ID
	if req.RouteID != nil {
		id, err := parseID(*req.RouteID)
		if err != nil {
			plathttp.WriteError(c, kerr.Validation("invalid route_id"))
			return
		}
		routeID = &id
	}

	in := application.RecordHikeInput{
		MountainID:        mountainID,
		RouteID:           routeID,
		ClimbDate:         req.ClimbDate,
		EvidencePhotoKeys: req.EvidencePhotoKeys,
		Title:             req.Title,
		Summary:           req.Summary,
		Narrative:         req.Narrative,
		PhotoKeys:         req.PhotoKeys,
		Visibility:        req.Visibility,
	}

	result, err := h.svc.RecordHike(c.Request.Context(), userID, in)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusCreated, result)
}

func (h *Handler) listMyHikes(c *gin.Context) {
	userID := plathttp.UserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	result, err := h.svc.ListMyHikes(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) getMyHike(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid hike id"))
		return
	}
	userID := plathttp.UserID(c)

	hike, err := h.svc.GetMyHike(c.Request.Context(), id, userID)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, hike)
}

func (h *Handler) deleteMyHike(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid hike id"))
		return
	}
	userID := plathttp.UserID(c)

	if err := h.svc.DeleteMyHike(c.Request.Context(), id, userID); err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ---- §4 Journey posts -----------------------------------------------------

type createPostReq struct {
	MountainID string          `json:"mountain_id" binding:"required"`
	RouteID    *string         `json:"route_id"`
	Title      string          `json:"title" binding:"required"`
	Summary    map[string]any  `json:"summary"`
	Narrative  string          `json:"narrative"`
	PhotoKeys  []string        `json:"photo_keys"`
	Visibility string          `json:"visibility"`
}

func (h *Handler) createPost(c *gin.Context) {
	var req createPostReq
	if err := c.ShouldBindJSON(&req); err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid request body"))
		return
	}

	userID := plathttp.UserID(c)
	mountainID, err := parseID(req.MountainID)
	if err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid mountain_id"))
		return
	}

	var routeID *ids.ID
	if req.RouteID != nil {
		id, err := parseID(*req.RouteID)
		if err != nil {
			plathttp.WriteError(c, kerr.Validation("invalid route_id"))
			return
		}
		routeID = &id
	}

	in := application.CreatePostInput{
		MountainID: mountainID,
		RouteID:    routeID,
		Title:      req.Title,
		Summary:    req.Summary,
		Narrative:  req.Narrative,
		PhotoKeys:  req.PhotoKeys,
		Visibility: req.Visibility,
	}

	post, err := h.svc.CreatePost(c.Request.Context(), userID, in)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": post.ID})
}

type updatePostReq struct {
	Title      *string         `json:"title"`
	Summary    *map[string]any `json:"summary"`
	Narrative  *string         `json:"narrative"`
	PhotoKeys  *[]string       `json:"photo_keys"`
	Visibility *string         `json:"visibility"`
}

func (h *Handler) updatePost(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid journey id"))
		return
	}
	userID := plathttp.UserID(c)

	var req updatePostReq
	if err := c.ShouldBindJSON(&req); err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid request body"))
		return
	}

	in := application.UpdatePostInput{
		Title:      req.Title,
		Summary:    req.Summary,
		Narrative:  req.Narrative,
		PhotoKeys:  req.PhotoKeys,
		Visibility: req.Visibility,
	}

	post, err := h.svc.UpdatePost(c.Request.Context(), id, userID, in)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, post)
}

func (h *Handler) publishPost(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid journey id"))
		return
	}
	userID := plathttp.UserID(c)

	if err := h.svc.PublishPost(c.Request.Context(), id, userID); err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) deletePost(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid journey id"))
		return
	}
	userID := plathttp.UserID(c)

	if err := h.svc.DeletePost(c.Request.Context(), id, userID); err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ---- §8 Public feed -------------------------------------------------------

func (h *Handler) listFeed(c *gin.Context) {
	var filter domain.FeedFilter
	if v := c.Query("mountain_id"); v != "" {
		id, err := parseID(v)
		if err != nil {
			plathttp.WriteError(c, kerr.Validation("invalid mountain_id"))
			return
		}
		filter.MountainID = id
	}
	filter.Region = c.Query("region")
	filter.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	filter.PageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "20"))

	result, err := h.svc.ListFeed(c.Request.Context(), filter)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// listMountainJourneys serves GET /mountains/{slug}/journeys (contracts §2).
// Reuses the feed query after resolving the slug to a mountain id.
func (h *Handler) listMountainJourneys(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		plathttp.WriteError(c, kerr.Validation("missing mountain slug"))
		return
	}
	mountainID, err := h.svc.MountainIDBySlug(c.Request.Context(), slug)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	filter := domain.FeedFilter{
		MountainID: mountainID,
		Page:       1,
		PageSize:   20,
	}
	if v := c.Query("page"); v != "" {
		filter.Page, _ = strconv.Atoi(v)
	}
	if v := c.Query("page_size"); v != "" {
		filter.PageSize, _ = strconv.Atoi(v)
	}
	result, err := h.svc.ListFeed(c.Request.Context(), filter)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) getFeedItem(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid journey id"))
		return
	}

	detail, err := h.svc.GetFeedItem(c.Request.Context(), id)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, detail)
}

type reportPostReq struct {
	Reason string `json:"reason" binding:"required"`
	Detail string `json:"detail"`
}

func (h *Handler) reportPost(c *gin.Context) {
	_ = c.Param("id") // target of the report
	var req reportPostReq
	if err := c.ShouldBindJSON(&req); err != nil {
		plathttp.WriteError(c, kerr.Validation("reason is required"))
		return
	}
	// Report is created via moderation context. Acknowledge for now.
	c.JSON(http.StatusCreated, gin.H{"ok": true})
}

// ---- §8 Uploads -----------------------------------------------------------

func (h *Handler) upload(c *gin.Context) {
	// Shared platform pipeline (002): sniff, size-cap, store, presign.
	plathttp.UploadHandler(h.blobs)(c)
}

// ---- helpers --------------------------------------------------------------

func parseID(s string) (ids.ID, error) {
	return ids.Parse(s)
}
