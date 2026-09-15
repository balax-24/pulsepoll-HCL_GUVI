package middleware

import (
	"net/http"
	"strings"

	"pulsepoll/backend/internal/auth"
	"pulsepoll/backend/internal/response"

	"github.com/gin-gonic/gin"
)

const (
	// ContextUserIDKey is the context key for storing the authenticated user's ID.
	ContextUserIDKey = "auth_user_id"

	// ContextEmailKey is the context key for storing the authenticated user's email.
	ContextEmailKey = "auth_user_email"
)

// Authenticate verifies the presence and validity of a JWT Bearer token in the Authorization header.
func Authenticate(jwtMgr *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if authHeader == "" {
			response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authorization header is required")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authorization format must be 'Bearer <token>'")
			return
		}

		tokenString := strings.TrimSpace(parts[1])
		if tokenString == "" {
			response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Bearer token is empty")
			return
		}

		claims, err := jwtMgr.Validate(tokenString)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired authorization token")
			return
		}

		// Inject authenticated identity into Gin context for downstream handlers
		c.Set(ContextUserIDKey, claims.UserID)
		c.Set(ContextEmailKey, claims.Email)

		c.Next()
	}
}

// GetUserID retrieves the authenticated user's ID from the Gin request context.
func GetUserID(c *gin.Context) (string, bool) {
	val, exists := c.Get(ContextUserIDKey)
	if !exists {
		return "", false
	}
	userID, ok := val.(string)
	return userID, ok && userID != ""
}

// GetUserEmail retrieves the authenticated user's email from the Gin request context.
func GetUserEmail(c *gin.Context) (string, bool) {
	val, exists := c.Get(ContextEmailKey)
	if !exists {
		return "", false
	}
	email, ok := val.(string)
	return email, ok && email != ""
}
