package response

import (
	"errors"
	"net/http"

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

// BindJSON decodes the incoming JSON request body into target. If binding fails
// because the payload exceeds the maximum allowed size (e.g. MaxBytesReader triggering
// *http.MaxBytesError), it writes HTTP 413 PAYLOAD_TOO_LARGE.
// For any other binding failure (malformed syntax or missing required fields),
// it writes HTTP 400 INVALID_REQUEST_PAYLOAD with the provided fallback validation message.
// Returns true on successful binding, or false if an error response was sent.
func BindJSON(c *gin.Context, target any, fallbackValidationMessage string) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			Error(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "Request payload exceeds maximum allowed size (2 MB)")
			return false
		}
		Error(c, http.StatusBadRequest, "INVALID_REQUEST_PAYLOAD", fallbackValidationMessage)
		return false
	}
	return true
}
