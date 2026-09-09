package http

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"hikingfo/backend/internal/shared/ids"
)

// ---- Recovery (panic → 500 JSON) -----------------------------------------

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic recovered", "error", r, "path", c.Request.URL.Path)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{
						"code":    "internal",
						"message": "an unexpected error occurred",
					},
				})
			}
		}()
		c.Next()
	}
}

// ---- Request ID ----------------------------------------------------------

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}
		c.Header("X-Request-ID", id)
		c.Set("request_id", id)
		c.Next()
	}
}

// ---- i18n Accept-Language ------------------------------------------------

func I18n() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("lang", parseAcceptLanguage(c.GetHeader("Accept-Language")))
		c.Next()
	}
}

func parseAcceptLanguage(header string) string {
	if header == "" {
		return "id"
	}
	for part := range strings.SplitSeq(header, ",") {
		lang := strings.TrimSpace(strings.SplitN(part, ";", 2)[0])
		switch {
		case strings.HasPrefix(lang, "en"):
			return "en"
		case strings.HasPrefix(lang, "id"):
			return "id"
		}
	}
	return "id"
}

// ---- Session Auth --------------------------------------------------------

// SessionLookup is the port the middleware uses to resolve a session token
// hash to a lightweight user snapshot.  The identity infrastructure provides
// an adapter; the platform never imports identity/domain.
type SessionLookup interface {
	FindUserByTokenHash(ctx context.Context, tokenHash string) (*SessionUser, error)
}

// SessionUser is the platform's read-only view of a user loaded from a
// session.  It avoids importing identity/domain in the platform layer.
type SessionUser struct {
	ID        ids.ID
	Role      string // "member" | "admin"
	Status    string // "active" | "suspended" | "banned"
	CSRFToken string // per-session CSRF token; "" when no session
}

// IsActive reports whether the user may use the platform.
func (u *SessionUser) IsActive() bool { return u.Status == "active" }

func SessionAuth(cookieName string, lookup SessionLookup) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(cookieName)
		if err != nil || token == "" {
			c.Next()
			return
		}
		hash := HashToken(token)
		u, err := lookup.FindUserByTokenHash(c.Request.Context(), hash)
		if err != nil {
			slog.Error("session lookup failed", "error", err, "request_id", c.GetString("request_id"))
			c.Next()
			return
		}
		if u == nil || !u.IsActive() {
			c.Next()
			return
		}
		SetUser(c, u.ID, u.Role)
		if u.CSRFToken != "" {
			// Session-scoped CSRF cookie: readable by the SPA (not httpOnly)
			// so it can echo the value in X-CSRF-Token on mutating requests.
			// Path must be "/" — document.cookie only exposes cookies whose
			// path prefixes the PAGE path; "/api" hid it from every non-API route.
			secure := !isLoopbackHost(c.Request.Host)
			c.SetSameSite(http.SameSiteLaxMode)
			c.SetCookie("hikingfo_csrf", u.CSRFToken, 0, "/", "", secure, false)
		}
		c.Next()
	}
}

// isLoopbackHost reports whether the request host looks like localhost /
// 127.0.0.1 / ::1 (dev mode — cookie Secure flag off).
func isLoopbackHost(host string) bool {
	h, _, _ := strings.Cut(host, ":")
	return h == "localhost" || h == "127.0.0.1" || h == "::1"
}

// HashToken computes base64(SHA-256(token)) — matches the form stored by
// identity/domain.HashToken without importing that package.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// ---- Auth Guard (require session) ----------------------------------------

func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if UserID(c) == ids.Nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "unauthorized",
					"message": "authentication required",
				},
			})
			return
		}
		c.Next()
	}
}

// ---- Admin Guard ---------------------------------------------------------

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if UserRole(c) != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":    "forbidden",
					"message": "admin access required",
				},
			})
			return
		}
		c.Next()
	}
}

// ---- CSRF (mutating methods) ---------------------------------------------

func CSRF(csrfCookieName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}
		cookie, err := c.Cookie(csrfCookieName)
		if err != nil || cookie == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":    "forbidden",
					"message": "CSRF token missing",
				},
			})
			return
		}
		header := c.GetHeader("X-CSRF-Token")
		if subtle.ConstantTimeCompare([]byte(cookie), []byte(header)) != 1 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":    "forbidden",
					"message": "CSRF token mismatch",
				},
			})
			return
		}
		c.Next()
	}
}

// ---- CORS (dev convenience) ----------------------------------------------

func DevCORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers",
			"Content-Type, Accept, Authorization, X-CSRF-Token, X-Request-ID, Accept-Language")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// ---- Last-Seen Toucher ---------------------------------------------------

func TouchLastSeen(toucher func(ctx context.Context, userID ids.ID, at time.Time) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		uid := UserID(c)
		if uid == ids.Nil {
			return
		}
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = toucher(ctx, uid, time.Now())
		}()
	}
}
