package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response represents a standard JSON API envelope.
type Response struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Error   any    `json:"error,omitempty"`
}

// Success sends a success JSON response.
func Success(c *gin.Context, code int, message string, data any) {
	c.JSON(code, Response{
		Success: true,
		Code:    code,
		Message: message,
		Data:    data,
	})
}

// Error sends an error JSON response.
func Error(c *gin.Context, code int, message string, err any) {
	c.JSON(code, Response{
		Success: false,
		Code:    code,
		Message: message,
		Error:   err,
	})
}

// Unauthorized sends 401 Unauthorized response.
func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, message, nil)
}

// Forbidden sends 403 Forbidden response.
func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, message, nil)
}

// BadRequest sends 400 Bad Request response.
func BadRequest(c *gin.Context, message string, err any) {
	Error(c, http.StatusBadRequest, message, err)
}

// InternalServerError sends 500 Internal Server Error response.
func InternalServerError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, message, nil)
}
