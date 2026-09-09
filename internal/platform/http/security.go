package http

import (
	"github.com/gin-gonic/gin"
)

// ---- Security Headers -----------------------------------------------------
//
// Baseline hardening headers on every response. The SPA is served from the
// same origin in dev and via the reverse proxy in production, so a
// default-src 'self' policy covers both; 'unsafe-inline' styles are allowed
// because the frontend ships utility CSS injected at build time.

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Content-Security-Policy",
			"default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self'; frame-ancestors 'none'")
		h.Set("Permissions-Policy", "geolocation=(), camera=(), microphone=()")
		if c.Request.TLS != nil {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		c.Next()
	}
}
