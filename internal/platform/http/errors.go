package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"hikingfo/backend/internal/shared/kerr"
)

// WriteError maps a *kerr.Error (or any error) to the uniform JSON shape
// defined in contracts/api.md → Conventions → Error shape.
func WriteError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	var ke *kerr.Error
	if !errors.As(err, &ke) {
		ke = kerr.WrapInternal("internal error", err)
	}
	c.JSON(kerrToStatus(ke), gin.H{
		"error": gin.H{
			"code":    string(ke.Code),
			"message": ke.Message,
			"fields":  ke.Fields,
		},
	})
}

// kerrToStatus maps a kerr.Code to the HTTP status code used in the API.
func kerrToStatus(ke *kerr.Error) int {
	switch ke.Code {
	case kerr.CodeValidation:
		return http.StatusBadRequest
	case kerr.CodeUnauthorized:
		return http.StatusUnauthorized
	case kerr.CodeForbidden:
		return http.StatusForbidden
	case kerr.CodeNotFound:
		return http.StatusNotFound
	case kerr.CodeConflict:
		return http.StatusConflict
	case kerr.CodeRateLimited:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}
