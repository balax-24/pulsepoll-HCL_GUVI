package response

import (
	"github.com/gin-gonic/gin"
)

// ErrorDetail represents the structured error details sent to clients.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse represents the standardized JSON error envelope.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// Error aborts the request context and writes a standardized JSON error response.
func Error(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

// JSON sends a JSON response with the given status code and payload.
func JSON(c *gin.Context, status int, data any) {
	c.JSON(status, data)
}
