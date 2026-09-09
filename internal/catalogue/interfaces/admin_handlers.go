package interfaces

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"hikingfo/backend/internal/catalogue/application"
	"hikingfo/backend/internal/catalogue/domain"
	"hikingfo/backend/internal/shared/ids"
	"hikingfo/backend/internal/shared/kerr"
	plathttp "hikingfo/backend/internal/platform/http"
)

// RegisterAdminRoutes mounts the catalogue admin API under /admin, guarded by
// RequireAdmin (contracts §9).
func RegisterAdminRoutes(rg *gin.RouterGroup, h *Handler) {
	g := rg.Group("/admin", plathttp.RequireAdmin())
	{
		g.GET("/mountains", h.adminList)
		g.POST("/mountains", h.adminCreate)
		g.GET("/mountains/:id", h.adminGet)
		g.GET("/mountains/:id/profile", h.adminProfile)
		g.PATCH("/mountains/:id", h.adminUpdate)
		g.DELETE("/mountains/:id", h.adminDelete)
		g.GET("/mountains/:id/revisions", h.adminRevisions)
		g.POST("/mountains/:id/revisions/:rev/rollback", h.adminRollback)

		g.POST("/mountains/:id/routes", h.adminCreateRoute)
		g.PATCH("/admin-routes/:rid", h.adminUpdateRoute)
		g.DELETE("/admin-routes/:rid", h.adminDeleteRoute)
		g.POST("/mountains/:id/basecamps", h.adminCreateBasecamp)
		g.PATCH("/admin-basecamps/:bid", h.adminUpdateBasecamp)
		g.DELETE("/admin-basecamps/:bid", h.adminDeleteBasecamp)

		// 002 §Photo — cover photo + attribution trail.
		g.PUT("/mountains/:id/photo", h.adminSetPhoto)
		g.DELETE("/mountains/:id/photo", h.adminRemovePhoto)
		g.GET("/mountains/:id/edit-log", h.adminEditLog)

		// 004 §Gallery — curated photos (contracts §1).
		g.POST("/mountains/:id/photos", h.adminAddGalleryPhoto)
		g.DELETE("/mountains/:id/photos/:photo_id", h.adminDeleteGalleryPhoto)
	}
}

func parseID(c *gin.Context, name string) (ids.ID, bool) {
	id, err := ids.Parse(c.Param(name))
	if err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid "+name))
		return ids.Nil, false
	}
	return id, true
}

func adminID(c *gin.Context) ids.ID {
	return plathttp.UserID(c)
}

// ---- mountains -------------------------------------------------------------

func (h *Handler) adminList(c *gin.Context) {
	items, err := h.svc.AdminList(c.Request.Context())
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) adminGet(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	m, err := h.svc.AdminGet(c.Request.Context(), id)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, m)
}

func (h *Handler) adminProfile(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	pv, err := h.svc.AdminProfile(c.Request.Context(), id)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, pv)
}

func (h *Handler) adminCreate(c *gin.Context) {
	var req application.CreateMountainInput
	if err := c.ShouldBindJSON(&req); err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid request body"))
		return
	}
	m, err := h.svc.AdminCreate(c.Request.Context(), req)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusCreated, m)
}

func (h *Handler) adminUpdate(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req application.UpdateMountainInput
	if err := c.ShouldBindJSON(&req); err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid request body"))
		return
	}
	m, err := h.svc.AdminUpdate(c.Request.Context(), id, req, adminID(c))
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, m)
}

func (h *Handler) adminDelete(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.AdminDelete(c.Request.Context(), id); err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) adminRevisions(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	revs, err := h.svc.AdminRevisions(c.Request.Context(), id)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": revs})
}

func (h *Handler) adminRollback(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	rev, ok := parseID(c, "rev")
	if !ok {
		return
	}
	if err := h.svc.AdminRollback(c.Request.Context(), id, rev, adminID(c)); err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ---- routes & basecamps ----------------------------------------------------

func (h *Handler) adminCreateRoute(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req application.RouteInput
	if err := c.ShouldBindJSON(&req); err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid request body"))
		return
	}
	r, err := h.svc.AdminCreateRoute(c.Request.Context(), id, adminID(c), req)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusCreated, r)
}

func (h *Handler) adminUpdateRoute(c *gin.Context) {
	rid, ok := parseID(c, "rid")
	if !ok {
		return
	}
	var req application.RouteInput
	if err := c.ShouldBindJSON(&req); err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid request body"))
		return
	}
	r, err := h.svc.AdminUpdateRoute(c.Request.Context(), rid, adminID(c), req)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, r)
}

func (h *Handler) adminDeleteRoute(c *gin.Context) {
	rid, ok := parseID(c, "rid")
	if !ok {
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req) // body optional; ignore parse failure
	if err := h.svc.AdminDeleteRoute(c.Request.Context(), rid, adminID(c), req.Reason); err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) adminCreateBasecamp(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req application.BasecampInput
	if err := c.ShouldBindJSON(&req); err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid request body"))
		return
	}
	b, err := h.svc.AdminCreateBasecamp(c.Request.Context(), id, adminID(c), req)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusCreated, b)
}

func (h *Handler) adminUpdateBasecamp(c *gin.Context) {
	bid, ok := parseID(c, "bid")
	if !ok {
		return
	}
	var req application.BasecampInput
	if err := c.ShouldBindJSON(&req); err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid request body"))
		return
	}
	b, err := h.svc.AdminUpdateBasecamp(c.Request.Context(), bid, adminID(c), req)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, b)
}

func (h *Handler) adminDeleteBasecamp(c *gin.Context) {
	bid, ok := parseID(c, "bid")
	if !ok {
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req) // body optional; ignore parse failure
	if err := h.svc.AdminDeleteBasecamp(c.Request.Context(), bid, adminID(c), req.Reason); err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ---- photo + edit log (002) -------------------------------------------------

type setPhotoReq struct {
	Key    string `json:"key" binding:"required"`
	Reason string `json:"reason"`
}

func (h *Handler) adminSetPhoto(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req setPhotoReq
	if err := c.ShouldBindJSON(&req); err != nil {
		plathttp.WriteError(c, kerr.Validation("key is required").WithField("key", "required"))
		return
	}
	m, err := h.svc.AdminSetPhoto(c.Request.Context(), id, adminID(c), req.Key, req.Reason)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, m)
}

func (h *Handler) adminRemovePhoto(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req) // body optional per contracts §Photo
	if err := h.svc.AdminRemovePhoto(c.Request.Context(), id, adminID(c), req.Reason); err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) adminEditLog(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	items, err := h.svc.AdminEditLog(c.Request.Context(), id)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	if items == nil {
		items = []domain.EditLogEntry{}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// ---- gallery (004) ----------------------------------------------------------

// adminAddGalleryPhoto POSTs {photo_key} onto a mountain's gallery.
func (h *Handler) adminAddGalleryPhoto(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req struct {
		PhotoKey string `json:"photo_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PhotoKey == "" {
		plathttp.WriteError(c, kerr.Validation("photo_key is required").WithField("photo_key", "required"))
		return
	}
	p, err := h.svc.AdminAddPhoto(c.Request.Context(), id, adminID(c), req.PhotoKey)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": p.ID})
}

// adminDeleteGalleryPhoto removes one gallery photo (204; 404 when it is not
// on this mountain).
func (h *Handler) adminDeleteGalleryPhoto(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	photoID, ok := parseID(c, "photo_id")
	if !ok {
		return
	}
	if err := h.svc.AdminDeletePhoto(c.Request.Context(), photoID, id, adminID(c)); err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
