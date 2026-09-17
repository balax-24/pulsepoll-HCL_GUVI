package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"pulsepoll/backend/internal/response"

	"github.com/gin-gonic/gin"
)

// Recovery returns a Gin middleware that gracefully recovers from panics,
// logs the stack/error context safely, and returns a standardized 500 error response.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				errStr := fmt.Sprintf("%v", r)
				stackTrace := string(debug.Stack())
				slog.Error("Unhandled panic recovered in HTTP handler",
					slog.String("error", errStr),
					slog.String("path", c.Request.URL.Path),
					slog.String("method", c.Request.Method),
					slog.String("stack", stackTrace),
				)

				response.Error(
					c,
					http.StatusInternalServerError,
					"INTERNAL_SERVER_ERROR",
					"An unexpected error occurred while processing your request",
				)
			}
		}()
		c.Next()
	}
}
