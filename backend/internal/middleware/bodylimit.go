package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequestBodyLimit returns Gin middleware that restricts incoming HTTP request bodies
// to at most maxBytes using http.MaxBytesReader. This protects against denial-of-service
// attacks attempting to stream arbitrarily large payloads into server memory.
func RequestBodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
