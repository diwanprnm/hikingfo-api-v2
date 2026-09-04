// Package interfaces owns the HTTP handlers and route registration for the
// identity bounded context. It is the only package that touches gin.Context
// from the identity side; the application layer is framework-free.
package interfaces

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"hikingfo/backend/internal/identity/application"
	"hikingfo/backend/internal/identity/domain"
	plathttp "hikingfo/backend/internal/platform/http"
	"hikingfo/backend/internal/shared/ids"
	"hikingfo/backend/internal/shared/kerr"
	"hikingfo/backend/internal/shared/langtext"
)

// Handler holds the identity application service and cookie config.
type Handler struct {
	svc        *application.Service
	cookieName string
	// googleVerifier is nil when Google auth is unconfigured.
	googleVerifier application.GoogleVerifier
	// experience derives the user's level from distinct-mountain count.
	experience domain.ExperienceProvider
}

// Config carries non-service wiring for the identity handlers.
type Config struct {
	CookieName    string
	GoogleVerifier application.GoogleVerifier
	Experience    domain.ExperienceProvider
}

// NewHandler wires the handler.
func NewHandler(svc *application.Service, cfg Config) *Handler {
	cookie := cfg.CookieName
	if cookie == "" {
		cookie = "hikingfo_session"
	}
	return &Handler{
		svc:            svc,
		cookieName:     cookie,
		googleVerifier: cfg.GoogleVerifier,
		experience:     cfg.Experience,
	}
}

// RegisterRoutes mounts all identity routes under /api/v1.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	auth := rg.Group("/auth")
	{
		auth.POST("/register", h.register)
		auth.POST("/login", h.login)
		auth.POST("/google", h.googleLogin)
		auth.POST("/logout", plathttp.RequireAuth(), h.logout)
		auth.POST("/verify-email", h.verifyEmail)
		auth.POST("/forgot-password", h.forgotPassword)
		auth.POST("/reset-password", h.resetPassword)
		auth.GET("/me", plathttp.RequireAuth(), h.me)
	}

	// Profile + contacts — authenticated.
	me := rg.Group("/me", plathttp.RequireAuth())
	{
		me.PATCH("/profile", h.updateProfile)
		me.PUT("/contacts", h.updateContacts)
	}
}

// ---- request / response shapes --------------------------------------------

type registerReq struct {
	Email       string `json:"email" binding:"required"`
	Password    string `json:"password" binding:"required"`
	DisplayName string `json:"display_name" binding:"required"`
}

type loginReq struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type googleReq struct {
	Credential string `json:"credential" binding:"required"`
}

type verifyEmailReq struct {
	Token string `json:"token" binding:"required"`
}

type forgotPasswordReq struct {
	Email string `json:"email" binding:"required"`
}

type resetPasswordReq struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

type updateProfileReq struct {
	DisplayName *string `json:"display_name"`
	AvatarKey   *string `json:"avatar_key"`
	BioID       *string `json:"bio_id"`
	BioEN       *string `json:"bio_en"`
	HomeRegion  *string `json:"home_region"`
}

type updateContactsReq struct {
	Phone     *string `json:"phone"`
	WhatsApp  *string `json:"whatsapp"`
	Instagram *string `json:"instagram"`
}

// ---- handlers -------------------------------------------------------------

func (h *Handler) register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		plathttp.WriteError(c, kerrFromBinding(err))
		return
	}
	u, err := h.svc.Register(c.Request.Context(), application.RegisterParams{
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":           u.ID,
		"email":        u.Email,
		"display_name": u.DisplayName,
	})
}

func (h *Handler) login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		plathttp.WriteError(c, kerrFromBinding(err))
		return
	}
	sess, err := h.svc.Login(c.Request.Context(), application.LoginParams{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	h.setSessionCookie(c, sess.Token, sess.Session.ExpiresAt)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) googleLogin(c *gin.Context) {
	var req googleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		plathttp.WriteError(c, kerrFromBinding(err))
		return
	}
	if h.googleVerifier == nil {
		plathttp.WriteError(c, kerr.Internal("Google sign-in is not configured"))
		return
	}
	sess, _, err := h.svc.GoogleLogin(c.Request.Context(), application.GoogleParams{
		Credential: req.Credential,
	}, h.googleVerifier)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	h.setSessionCookie(c, sess.Token, sess.Session.ExpiresAt)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) logout(c *gin.Context) {
	token, _ := c.Cookie(h.cookieName)
	_ = h.svc.Logout(c.Request.Context(), plathttp.HashToken(token))
	h.clearSessionCookie(c)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) verifyEmail(c *gin.Context) {
	var req verifyEmailReq
	if err := c.ShouldBindJSON(&req); err != nil {
		plathttp.WriteError(c, kerrFromBinding(err))
		return
	}
	if err := h.svc.VerifyEmail(c.Request.Context(), req.Token); err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) forgotPassword(c *gin.Context) {
	var req forgotPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		plathttp.WriteError(c, kerrFromBinding(err))
		return
	}
	if err := h.svc.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		plathttp.WriteError(c, err)
		return
	}
	// Always return success — never reveal whether the email exists.
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) resetPassword(c *gin.Context) {
	var req resetPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		plathttp.WriteError(c, kerrFromBinding(err))
		return
	}
	if err := h.svc.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) me(c *gin.Context) {
	uid := plathttp.UserID(c)
	if uid == ids.Nil {
		plathttp.WriteError(c, kerr.Internal("missing user in context"))
		return
	}
	// The achievement context provides the distinct count; wire via dependency
	// injection. For now, pass 0/unknown — the composition root will call an
	// achievement query port here. (Deferred to T021/T063 when achievement
	// lands.)
	distinctCount := 0
	level := domain.LevelPemula
	if h.experience != nil {
		level = h.experience.LevelFor(distinctCount)
	}
	v, err := h.svc.Me(c.Request.Context(), uid, distinctCount, level)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, meViewToJSON(v))
}

func (h *Handler) updateProfile(c *gin.Context) {
	uid := plathttp.UserID(c)
	var req updateProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		plathttp.WriteError(c, kerrFromBinding(err))
		return
	}
	p := application.UpdateProfileParams{
		DisplayName: deref(req.DisplayName, ""),
		AvatarKey:   deref(req.AvatarKey, ""),
		BioID:       deref(req.BioID, ""),
		BioEN:       deref(req.BioEN, ""),
		HomeRegion:  deref(req.HomeRegion, ""),
	}
	u, err := h.svc.UpdateProfile(c.Request.Context(), uid, p)
	if err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":           u.ID,
		"display_name": u.DisplayName,
		"avatar_key":   u.AvatarKey,
		"bio":          u.Bio,
		"home_region":  u.HomeRegion,
	})
}

func (h *Handler) updateContacts(c *gin.Context) {
	uid := plathttp.UserID(c)
	var req updateContactsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		plathttp.WriteError(c, kerrFromBinding(err))
		return
	}
	cc := domain.ContactChannel{
		Phone:     deref(req.Phone, ""),
		WhatsApp:  deref(req.WhatsApp, ""),
		Instagram: deref(req.Instagram, ""),
	}
	if err := h.svc.UpdateContacts(c.Request.Context(), uid, cc); err != nil {
		plathttp.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ---- helpers --------------------------------------------------------------

func (h *Handler) setSessionCookie(c *gin.Context, token string, expiresAt time.Time) {
	// SameSite=Lax per contracts; httpOnly; path scoped to API.
	secure := !isLoopback(c.Request.Host)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(h.cookieName, token, int(time.Until(expiresAt).Seconds()), "/api", "", secure, true)
}

func (h *Handler) clearSessionCookie(c *gin.Context) {
	c.SetCookie(h.cookieName, "", -1, "/api", "", false, true)
}

// isLoopback returns true when the request host looks like localhost / 127.0.0.1
// (dev mode — cookie Secure flag off).
func isLoopback(host string) bool {
	h, _, _ := strings.Cut(host, ":")
	return h == "localhost" || h == "127.0.0.1" || h == "::1"
}

// deref returns the pointed-to value or fallback.
func deref(p *string, fallback string) string {
	if p == nil {
		return fallback
	}
	return *p
}

// kerrFromBinding translates gin binding errors into the uniform kerr shape.
func kerrFromBinding(err error) error {
	msg := err.Error()
	if strings.Contains(msg, "required") {
		return kerr.Validation("validation failed")
	}
	return kerr.Validation(msg)
}

// meViewToJSON serialises a MeView for the API response.
func meViewToJSON(v *application.MeView) gin.H {
	return gin.H{
		"id":               v.ID,
		"email":            v.Email,
		"email_verified":   v.EmailVerified,
		"display_name":     v.DisplayName,
		"avatar_key":       v.AvatarKey,
		"home_region":      v.HomeRegion,
		"bio":              langtextToFallback(v.Bio),
		"role":             v.Role,
		"status":           v.Status,
		"joined_at":        v.JoinedAt,
		"distinct_count":   v.DistinctCount,
		"experience_level": v.ExperienceLevel,
	}
}

func langtextToFallback(t langtext.Text) map[string]string {
	if !t.Has() {
		return nil
	}
	return map[string]string{"id": t.ID, "en": t.EN}
}

