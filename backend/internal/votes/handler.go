package votes

import (
	"errors"
	"net/http"
	"strings"

	"pulsepoll/backend/internal/response"

	"github.com/gin-gonic/gin"
)

// Handler processes audience voting and results requests.
type Handler struct {
	service *Service
}

// NewHandler constructs a votes Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// CastVote handles POST /api/polls/:id/vote (public audience endpoint).
func (h *Handler) CastVote(c *gin.Context) {
	pollID := strings.TrimSpace(c.Param("id"))
	if pollID == "" {
		response.Error(c, http.StatusBadRequest, "INVALID_POLL_ID", "Poll ID is required in route")
		return
	}

	var req CastVoteRequest
	if !response.BindJSON(c, &req, "Request body must include 'option_id'") {
		return
	}

	if strings.TrimSpace(req.OptionID) == "" {
		response.Error(c, http.StatusBadRequest, "INVALID_OPTION", "Field 'option_id' cannot be empty")
		return
	}

	// Resolve or initialize anonymous voter identity via header/cookie
	voterID := GetOrCreateVoterID(c)

	resp, err := h.service.CastVote(c.Request.Context(), pollID, req.OptionID, voterID)
	if err != nil {
		switch {
		case errors.Is(err, ErrPollNotFound):
			response.Error(c, http.StatusNotFound, "POLL_NOT_FOUND", "Poll not found")
		case errors.Is(err, ErrPollClosed):
			response.Error(c, http.StatusConflict, "POLL_CLOSED", "This poll is closed and no longer accepting votes.")
		case errors.Is(err, ErrPollExpired):
			response.Error(c, http.StatusConflict, "POLL_EXPIRED", "This poll has expired and is no longer accepting votes.")
		case errors.Is(err, ErrInvalidOption):
			response.Error(c, http.StatusBadRequest, "INVALID_OPTION", "Option does not belong to this poll")
		case errors.Is(err, ErrDuplicateVote):
			response.Error(c, http.StatusConflict, "DUPLICATE_VOTE", "You have already voted in this poll")
		case errors.Is(err, ErrInvalidInput):
			response.Error(c, http.StatusBadRequest, "VALIDATION_FAILED", err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "VOTE_FAILED", "Failed to record vote")
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetResults handles GET /api/polls/:id/results (public audience endpoint).
func (h *Handler) GetResults(c *gin.Context) {
	pollID := strings.TrimSpace(c.Param("id"))
	if pollID == "" {
		response.Error(c, http.StatusBadRequest, "INVALID_POLL_ID", "Poll ID is required in route")
		return
	}

	results, err := h.service.GetResults(c.Request.Context(), pollID)
	if err != nil {
		if errors.Is(err, ErrPollNotFound) {
			response.Error(c, http.StatusNotFound, "POLL_NOT_FOUND", "Poll not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "RESULTS_QUERY_FAILED", "Failed to retrieve poll results")
		return
	}

	c.JSON(http.StatusOK, results)
}
