package middleware

import (
	"net/http"

	"pulsepoll/backend/internal/response"

	"github.com/gin-gonic/gin"
)

// RequestBodyLimit returns Gin middleware that restricts incoming HTTP request bodies
// to at most maxBytes using an early Content-Length check and http.MaxBytesReader.
// This protects against denial-of-service attacks attempting to stream arbitrarily large
// payloads into server memory.
func RequestBodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			response.Error(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "Request payload exceeds maximum allowed size (2 MB)")
			c.Abort()
			return
		}
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
