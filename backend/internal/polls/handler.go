package polls

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"pulsepoll/backend/internal/middleware"
	"pulsepoll/backend/internal/response"

	"github.com/gin-gonic/gin"
)

// Handler handles incoming HTTP requests for poll management.
type Handler struct {
	service *Service
}

// NewHandler constructs a new polls Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// Create handles POST /api/polls.
func (h *Handler) Create(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok || strings.TrimSpace(userID) == "" {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	var req CreatePollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST_PAYLOAD", "Request body must include 'question' and 'options'")
		return
	}

	poll, err := h.service.CreatePoll(c.Request.Context(), userID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(c, http.StatusBadRequest, "VALIDATION_FAILED", err.Error())
		case errors.Is(err, ErrForbidden):
			response.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "POLL_CREATE_FAILED", "Failed to create poll")
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"poll": poll,
	})
}

// GetPublic handles GET /api/polls/:id (public audience endpoint).
func (h *Handler) GetPublic(c *gin.Context) {
	pollID := c.Param("id")
	if strings.TrimSpace(pollID) == "" {
		response.Error(c, http.StatusBadRequest, "INVALID_POLL_ID", "Poll ID cannot be empty")
		return
	}

	poll, err := h.service.GetPublicPoll(c.Request.Context(), pollID)
	if err != nil {
		if errors.Is(err, ErrPollNotFound) {
			response.Error(c, http.StatusNotFound, "POLL_NOT_FOUND", "Poll not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "POLL_QUERY_FAILED", "Failed to retrieve poll")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"poll": poll,
	})
}

// GetMyPolls handles GET /api/my/polls (creator polls listing).
func (h *Handler) GetMyPolls(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok || strings.TrimSpace(userID) == "" {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	page := int64(1)
	if pStr := c.Query("page"); pStr != "" {
		if p, err := strconv.ParseInt(pStr, 10, 64); err == nil && p > 0 {
			page = p
		}
	}

	limit := int64(20)
	if lStr := c.Query("limit"); lStr != "" {
		if l, err := strconv.ParseInt(lStr, 10, 64); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	result, err := h.service.GetCreatorPolls(c.Request.Context(), userID, page, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "CREATOR_POLLS_FAILED", "Failed to retrieve your polls")
		return
	}

	c.JSON(http.StatusOK, result)
}

// Update handles PATCH /api/polls/:id.
func (h *Handler) Update(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok || strings.TrimSpace(userID) == "" {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	pollID := c.Param("id")
	if strings.TrimSpace(pollID) == "" {
		response.Error(c, http.StatusBadRequest, "INVALID_POLL_ID", "Poll ID cannot be empty")
		return
	}

	var req UpdatePollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST_PAYLOAD", "Invalid JSON payload for poll update")
		return
	}

	poll, err := h.service.UpdatePoll(c.Request.Context(), userID, pollID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrPollNotFound):
			response.Error(c, http.StatusNotFound, "POLL_NOT_FOUND", "Poll not found")
		case errors.Is(err, ErrForbidden):
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "You do not have permission to modify this poll")
		case errors.Is(err, ErrPollClosed):
			response.Error(c, http.StatusConflict, "POLL_CLOSED", "Cannot update a closed poll")
		case errors.Is(err, ErrInvalidInput):
			response.Error(c, http.StatusBadRequest, "VALIDATION_FAILED", err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "POLL_UPDATE_FAILED", "Failed to update poll")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"poll": poll,
	})
}

// UpdateStatus handles PATCH /api/polls/:id/status.
func (h *Handler) UpdateStatus(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok || strings.TrimSpace(userID) == "" {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	pollID := c.Param("id")
	if strings.TrimSpace(pollID) == "" {
		response.Error(c, http.StatusBadRequest, "INVALID_POLL_ID", "Poll ID cannot be empty")
		return
	}

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST_PAYLOAD", "Field 'status' must be specified ('active' or 'closed')")
		return
	}

	poll, err := h.service.UpdateStatus(c.Request.Context(), userID, pollID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrPollNotFound):
			response.Error(c, http.StatusNotFound, "POLL_NOT_FOUND", "Poll not found")
		case errors.Is(err, ErrForbidden):
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "You do not have permission to modify this poll")
		case errors.Is(err, ErrInvalidInput):
			response.Error(c, http.StatusBadRequest, "VALIDATION_FAILED", err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "POLL_STATUS_UPDATE_FAILED", "Failed to update poll status")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"poll": poll,
	})
}

// Close handles POST /api/polls/:id/close.
func (h *Handler) Close(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok || strings.TrimSpace(userID) == "" {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	pollID := c.Param("id")
	if strings.TrimSpace(pollID) == "" {
		response.Error(c, http.StatusBadRequest, "INVALID_POLL_ID", "Poll ID cannot be empty")
		return
	}

	poll, err := h.service.ClosePoll(c.Request.Context(), userID, pollID)
	if err != nil {
		switch {
		case errors.Is(err, ErrPollNotFound):
			response.Error(c, http.StatusNotFound, "POLL_NOT_FOUND", "Poll not found")
		case errors.Is(err, ErrForbidden):
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "You do not have permission to modify this poll")
		default:
			response.Error(c, http.StatusInternalServerError, "POLL_CLOSE_FAILED", "Failed to close poll")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"poll": poll,
	})
}

// Delete handles DELETE /api/polls/:id.
func (h *Handler) Delete(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok || strings.TrimSpace(userID) == "" {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	pollID := c.Param("id")
	if strings.TrimSpace(pollID) == "" {
		response.Error(c, http.StatusBadRequest, "INVALID_POLL_ID", "Poll ID cannot be empty")
		return
	}

	err := h.service.DeletePoll(c.Request.Context(), userID, pollID)
	if err != nil {
		switch {
		case errors.Is(err, ErrPollNotFound):
			response.Error(c, http.StatusNotFound, "POLL_NOT_FOUND", "Poll not found")
		case errors.Is(err, ErrForbidden):
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "You do not have permission to delete this poll")
		default:
			response.Error(c, http.StatusInternalServerError, "POLL_DELETE_FAILED", "Failed to delete poll")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Poll deleted successfully",
	})
}
