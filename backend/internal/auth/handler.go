package auth

import (
	"errors"
	"net/http"

	"pulsepoll/backend/internal/response"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for authentication and user accounts.
type Handler struct {
	service *Service
}

// NewHandler constructs a new auth Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// Register handles POST /api/auth/register.
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST_PAYLOAD", "Request body must contain name, email, and password")
		return
	}

	user, err := h.service.Register(c.Request.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(c, http.StatusBadRequest, "VALIDATION_FAILED", err.Error())
		case errors.Is(err, ErrDuplicateEmail):
			response.Error(c, http.StatusConflict, "EMAIL_EXISTS", "An account with this email address already exists")
		default:
			response.Error(c, http.StatusInternalServerError, "REGISTRATION_FAILED", "Failed to register user account")
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user": user,
	})
}

// Login handles POST /api/auth/login.
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST_PAYLOAD", "Request body must contain email and password")
		return
	}

	res, err := h.service.Login(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			response.Error(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
			return
		}
		response.Error(c, http.StatusInternalServerError, "LOGIN_FAILED", "Authentication failed due to an internal error")
		return
	}

	c.JSON(http.StatusOK, res)
}

// Me handles GET /api/auth/me (requires authentication).
func (h *Handler) Me(c *gin.Context) {
	// Retrieve authenticated user ID from context injected by middleware
	userIDVal, exists := c.Get("auth_user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid authentication state")
		return
	}

	user, err := h.service.GetProfile(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			response.Error(c, http.StatusNotFound, "USER_NOT_FOUND", "User profile not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "PROFILE_FAILED", "Failed to retrieve user profile")
		return
	}

	c.JSON(http.StatusOK, user)
}
