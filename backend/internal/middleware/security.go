package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeaders applies baseline defensive HTTP security headers to all responses.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME-type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking / frame embedding
		c.Header("X-Frame-Options", "DENY")

		// Restrict referrer leakage to same-origin for cross-origin requests
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Disable unused browser permissions on this API surface
		c.Header("Permissions-Policy", "geolocation=(), camera=(), microphone=()")

		c.Next()
	}
}
