// Package http owns the HTTP server bootstrap, middleware stack and uniform
// error handler for the hikingfo API.  It sits in the platform layer and
// composes routes from every bounded context's interfaces/ package.
package http

import (
	"github.com/gin-gonic/gin"

	"hikingfo/backend/internal/shared/ids"
)

type ctxKey string

const (
	ctxUserID   ctxKey = "user_id"
	ctxUserRole ctxKey = "user_role"
)

// SetUser writes the authenticated user's id and role into the Gin context so
// downstream handlers can read them without repeating the session lookup.
func SetUser(c *gin.Context, userID ids.ID, role string) {
	c.Set(string(ctxUserID), userID)
	c.Set(string(ctxUserRole), role)
}

// UserID reads the authenticated user's ID from the Gin context.  Returns
// ids.Nil when no session is active (public route).
func UserID(c *gin.Context) ids.ID {
	v, ok := c.Get(string(ctxUserID))
	if !ok {
		return ids.Nil
	}
	id, _ := v.(ids.ID)
	return id
}

// UserRole reads the authenticated user's role from the Gin context.
func UserRole(c *gin.Context) string {
	v, ok := c.Get(string(ctxUserRole))
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}
