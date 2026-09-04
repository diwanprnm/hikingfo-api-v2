// Package interfaces owns the HTTP handlers and route registration for the
// partner matching bounded context.
package interfaces

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"hikingfo/backend/internal/partner/application"
	"hikingfo/backend/internal/partner/domain"
	plathttp "hikingfo/backend/internal/platform/http"
	"hikingfo/backend/internal/shared/kerr"
	"hikingfo/backend/internal/shared/page"
	"hikingfo/backend/internal/shared/ids"
)

// Handler holds the partner application service.
type Handler struct{ svc *application.Service }

// NewHandler wires the handler.
func NewHandler(svc *application.Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts all partner routes under /api/v1.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	auth := rg.Group("", plathttp.RequireAuth())
	pub := rg.Group("")

	auth.GET("/partners/search", h.search)
	auth.POST("/partners/notices", h.createNotice)
	auth.GET("/partners/notices", h.listNotices)
	auth.DELETE("/partners/notices/:id", h.withdrawNotice)
	auth.POST("/partners/requests", h.sendRequest)
	auth.GET("/partners/requests", h.listRequests)
	auth.POST("/partners/requests/:id/accept", h.acceptRequest)
	auth.POST("/partners/requests/:id/decline", h.declineRequest)
	auth.DELETE("/partners/requests/:id", h.withdrawRequest)
	pub.POST("/partners/requests/:id/reports", h.reportRequest)
}

// ---- search ----------------------------------------------------------------

type searchReq struct {
	MountainID string `form:"mountain_id" binding:"required"`
	TripStart  string `form:"trip_start" binding:"required"`
	TripEnd    string `form:"trip_end" binding:"required"`
	Page       string `form:"page"`
	PageSize   string `form:"page_size"`
}

func (h *Handler) search(c *gin.Context) {
	var req searchReq
	if err := c.ShouldBindQuery(&req); err != nil {
		plathttp.WriteError(c, kerr.Validation("mountain_id, trip_start, trip_end are required"))
		return
	}

	mountainID, err := ids.Parse(req.MountainID)
	if err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid mountain_id"))
		return
	}
	tripStart, err := time.Parse("2006-01-02", req.TripStart)
	if err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid trip_start (use YYYY-MM-DD)"))
		return
	}
	tripEnd, err := time.Parse("2006-01-02", req.TripEnd)
	if err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid trip_end (use YYYY-MM-DD)"))
		return
	}

	cur := page.ParseCursor(req.Page, req.PageSize)
	result, err := h.svc.SearchCandidates(c.Request.Context(), plathttp.UserID(c), domain.CandidateFilter{
		MountainID: mountainID,
		TripStart:  tripStart,
		TripEnd:    tripEnd,
		Limit:      cur.PageSize,
		Offset:     cur.Offset(),
	})
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// ---- notices ---------------------------------------------------------------

type createNoticeReq struct {
	MountainID string `json:"mountain_id" binding:"required"`
	TripStart  string `json:"trip_start" binding:"required"`
	TripEnd    string `json:"trip_end" binding:"required"`
	Note       string `json:"note"`
}

func (h *Handler) createNotice(c *gin.Context) {
	var req createNoticeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		plathttp.WriteError(c, kerr.Validation("mountain_id, trip_start, trip_end are required"))
		return
	}

	mountainID, err := ids.Parse(req.MountainID)
	if err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid mountain_id"))
		return
	}
	tripStart, err := time.Parse("2006-01-02", req.TripStart)
	if err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid trip_start (use YYYY-MM-DD)"))
		return
	}
	tripEnd, err := time.Parse("2006-01-02", req.TripEnd)
	if err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid trip_end (use YYYY-MM-DD)"))
		return
	}

	n, err := h.svc.PublishNotice(c.Request.Context(), plathttp.UserID(c), mountainID, tripStart, tripEnd, req.Note)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": n.ID, "status": n.Status})
}

type listNoticesReq struct {
	MountainID string `form:"mountain_id"`
	Status     string `form:"status"`
	Page       string `form:"page"`
	PageSize   string `form:"page_size"`
}

func (h *Handler) listNotices(c *gin.Context) {
	var req listNoticesReq
	_ = c.ShouldBindQuery(&req)

	cur := page.ParseCursor(req.Page, req.PageSize)
	filter := domain.NoticeFilter{
		Limit:  cur.PageSize,
		Offset: cur.Offset(),
	}
	if req.MountainID != "" {
		id, err := ids.Parse(req.MountainID)
		if err == nil {
			filter.MountainID = &id
		}
	}
	if req.Status != "" {
		s := domain.NoticeStatus(req.Status)
		filter.Status = &s
	}

	items, total, err := h.svc.ListNotices(c.Request.Context(), plathttp.UserID(c), filter)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

func (h *Handler) withdrawNotice(c *gin.Context) {
	noticeID, err := ids.Parse(c.Param("id"))
	if err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid notice id"))
		return
	}
	if err := h.svc.WithdrawNotice(c.Request.Context(), noticeID, plathttp.UserID(c)); err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ---- requests --------------------------------------------------------------

type sendRequestReq struct {
	ToUserID  string `json:"to_user_id" binding:"required"`
	MountainID string `json:"mountain_id" binding:"required"`
	TripStart string `json:"trip_start" binding:"required"`
	TripEnd   string `json:"trip_end" binding:"required"`
	NoticeID  string `json:"notice_id"`
	Message   string `json:"message"`
}

func (h *Handler) sendRequest(c *gin.Context) {
	var req sendRequestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		plathttp.WriteError(c, kerr.Validation("to_user_id, mountain_id, trip_start, trip_end are required"))
		return
	}

	toUserID, err := ids.Parse(req.ToUserID)
	if err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid to_user_id"))
		return
	}
	mountainID, err := ids.Parse(req.MountainID)
	if err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid mountain_id"))
		return
	}
	tripStart, err := time.Parse("2006-01-02", req.TripStart)
	if err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid trip_start (use YYYY-MM-DD)"))
		return
	}
	tripEnd, err := time.Parse("2006-01-02", req.TripEnd)
	if err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid trip_end (use YYYY-MM-DD)"))
		return
	}

	var noticeID *ids.ID
	if req.NoticeID != "" {
		nid, err := ids.Parse(req.NoticeID)
		if err != nil {
			plathttp.WriteError(c, kerr.Validation("invalid notice_id"))
			return
		}
		noticeID = &nid
	}

	result, err := h.svc.SendRequest(c.Request.Context(), plathttp.UserID(c), toUserID, mountainID, tripStart, tripEnd, noticeID, req.Message)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": result.ID, "status": result.Status})
}

type listRequestsReq struct {
	Direction string `form:"direction"`
	Status    string `form:"status"`
	Page      string `form:"page"`
	PageSize  string `form:"page_size"`
}

func (h *Handler) listRequests(c *gin.Context) {
	var req listRequestsReq
	_ = c.ShouldBindQuery(&req)

	cur := page.ParseCursor(req.Page, req.PageSize)
	filter := domain.RequestFilter{
		Direction: req.Direction,
		Limit:     cur.PageSize,
		Offset:    cur.Offset(),
	}
	if req.Status != "" {
		s := domain.RequestStatus(req.Status)
		filter.Status = &s
	}

	items, total, err := h.svc.ListRequests(c.Request.Context(), plathttp.UserID(c), filter)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

func (h *Handler) acceptRequest(c *gin.Context) {
	requestID, err := ids.Parse(c.Param("id"))
	if err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid request id"))
		return
	}
	if err := h.svc.AcceptRequest(c.Request.Context(), requestID, plathttp.UserID(c)); err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "matched": true})
}

func (h *Handler) declineRequest(c *gin.Context) {
	requestID, err := ids.Parse(c.Param("id"))
	if err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid request id"))
		return
	}
	if err := h.svc.DeclineRequest(c.Request.Context(), requestID, plathttp.UserID(c)); err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) withdrawRequest(c *gin.Context) {
	requestID, err := ids.Parse(c.Param("id"))
	if err != nil {
		plathttp.WriteError(c, kerr.Validation("invalid request id"))
		return
	}
	if err := h.svc.WithdrawRequest(c.Request.Context(), requestID, plathttp.UserID(c)); err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ---- reports (public, stub) -----------------------------------------------

type reportReq struct {
	Reason string `json:"reason" binding:"required"`
	Detail string `json:"detail"`
}

func (h *Handler) reportRequest(c *gin.Context) {
	var req reportReq
	if err := c.ShouldBindJSON(&req); err != nil {
		plathttp.WriteError(c, kerr.Validation("reason is required"))
		return
	}
	// Report is created via the moderation context. Acknowledge for now.
	c.JSON(http.StatusCreated, gin.H{"ok": true})
}
