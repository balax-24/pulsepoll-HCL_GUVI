package polls

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

var (
	// ErrPollNotFound is returned when a requested poll does not exist or has been deleted.
	ErrPollNotFound = errors.New("poll not found")

	// ErrForbidden is returned when a non-owner attempts to modify or delete a poll.
	ErrForbidden = errors.New("you do not have permission to modify this poll")

	// ErrInvalidInput indicates validation failure on client-provided parameters.
	ErrInvalidInput = errors.New("validation failed")

	// ErrPollClosed is returned when an operation is disallowed because the poll is closed.
	ErrPollClosed = errors.New("poll is closed")
)

// PollEventBroadcaster allows broadcasting poll lifecycle events (e.g. poll closed).
type PollEventBroadcaster interface {
	BroadcastPollClosed(ctx context.Context, pollID string) error
}

// Service encapsulates core business and authorization logic for polls.
type Service struct {
	repo        PollRepository
	broadcaster PollEventBroadcaster
}

// NewService creates a new Service backed by a PollRepository.
func NewService(repo PollRepository) *Service {
	return &Service{
		repo: repo,
	}
}

// SetBroadcaster registers an optional realtime event broadcaster.
func (s *Service) SetBroadcaster(b PollEventBroadcaster) {
	s.broadcaster = b
}

// CreatePoll validates inputs, assigns stable IDs to options, and creates an active poll document.
// The creator ID is always sourced from the authenticated token context.
func (s *Service) CreatePoll(ctx context.Context, creatorID string, req CreatePollRequest) (*PollResponse, error) {
	if strings.TrimSpace(creatorID) == "" {
		return nil, ErrForbidden
	}

	question, options, err := validateCreatePoll(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidInput, err.Error())
	}

	pollOptions := make([]PollOption, len(options))
	for i, text := range options {
		pollOptions[i] = PollOption{
			ID:   bson.NewObjectID().Hex(),
			Text: text,
		}
	}

	now := time.Now().UTC()
	poll := &Poll{
		ID:        bson.NewObjectID(),
		CreatorID: creatorID,
		Question:  question,
		Options:   pollOptions,
		Status:    PollStatusActive,
		IsDeleted: false,
		ExpiresAt: req.ExpiresAt,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Create(ctx, poll); err != nil {
		return nil, fmt.Errorf("failed to persist poll: %w", err)
	}

	return poll.ToResponse(), nil
}

// GetPublicPoll retrieves an active or closed poll for public audience consumption.
// It returns a sanitized view that does not leak private creator identifiers.
func (s *Service) GetPublicPoll(ctx context.Context, pollID string) (*PublicPollResponse, error) {
	pollID = strings.TrimSpace(pollID)
	if pollID == "" {
		return nil, ErrPollNotFound
	}

	poll, err := s.repo.FindByID(ctx, pollID)
	if err != nil {
		return nil, err
	}

	return poll.ToPublicResponse(), nil
}

// GetCreatorPolls retrieves a paginated list of polls owned strictly by the authenticated creator.
func (s *Service) GetCreatorPolls(ctx context.Context, creatorID string, page, limit int64) (*ListPollsResponse, error) {
	if strings.TrimSpace(creatorID) == "" {
		return nil, ErrForbidden
	}

	const maxPage = 1000
	if page < 1 {
		page = 1
	} else if page > maxPage {
		page = maxPage
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	polls, total, err := s.repo.FindByCreatorID(ctx, creatorID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve creator polls: %w", err)
	}

	pollResponses := make([]*PollResponse, len(polls))
	for i, p := range polls {
		pollResponses[i] = p.ToResponse()
	}

	return &ListPollsResponse{
		Polls:      pollResponses,
		TotalCount: total,
		Page:       page,
		Limit:      limit,
	}, nil
}

// UpdatePoll allows the poll creator to update question and option text.
// Updates are rejected if the caller is not the owner, or if the poll is closed.
// Stable option IDs are preserved.
func (s *Service) UpdatePoll(ctx context.Context, creatorID, pollID string, req UpdatePollRequest) (*PollResponse, error) {
	if strings.TrimSpace(creatorID) == "" {
		return nil, ErrForbidden
	}

	pollID = strings.TrimSpace(pollID)
	if pollID == "" {
		return nil, ErrPollNotFound
	}

	poll, err := s.repo.FindByID(ctx, pollID)
	if err != nil {
		return nil, err
	}

	if poll.CreatorID != creatorID {
		return nil, ErrForbidden
	}

	if poll.Status == PollStatusClosed {
		return nil, ErrPollClosed
	}

	updatedPoll, err := validateAndUpdatePoll(poll, req)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidInput, err.Error())
	}

	updatedPoll.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, updatedPoll); err != nil {
		return nil, fmt.Errorf("failed to update poll: %w", err)
	}

	return updatedPoll.ToResponse(), nil
}

// UpdateStatus modifies the lifecycle status of a poll (e.g. closing it).
// Only the creator is permitted to change the status.
func (s *Service) UpdateStatus(ctx context.Context, creatorID, pollID string, req UpdateStatusRequest) (*PollResponse, error) {
	if strings.TrimSpace(creatorID) == "" {
		return nil, ErrForbidden
	}

	pollID = strings.TrimSpace(pollID)
	if pollID == "" {
		return nil, ErrPollNotFound
	}

	if req.Status != PollStatusActive && req.Status != PollStatusClosed {
		return nil, fmt.Errorf("%w: status must be 'active' or 'closed'", ErrInvalidInput)
	}

	poll, err := s.repo.FindByID(ctx, pollID)
	if err != nil {
		return nil, err
	}

	if poll.CreatorID != creatorID {
		return nil, ErrForbidden
	}

	// Idempotent transition: if already at the requested status, return current state
	if poll.Status == req.Status {
		return poll.ToResponse(), nil
	}

	updated, err := s.repo.UpdateStatus(ctx, pollID, req.Status)
	if err != nil {
		return nil, fmt.Errorf("failed to update poll status: %w", err)
	}

	if req.Status == PollStatusClosed && s.broadcaster != nil {
		_ = s.broadcaster.BroadcastPollClosed(ctx, pollID)
	}

	return updated.ToResponse(), nil
}

// ClosePoll provides a clean dedicated method to close a poll.
func (s *Service) ClosePoll(ctx context.Context, creatorID, pollID string) (*PollResponse, error) {
	return s.UpdateStatus(ctx, creatorID, pollID, UpdateStatusRequest{Status: PollStatusClosed})
}

// DeletePoll soft-deletes a poll owned by the authenticated creator.
// Non-owners receive ErrForbidden. Non-existent polls return ErrPollNotFound.
func (s *Service) DeletePoll(ctx context.Context, creatorID, pollID string) error {
	if strings.TrimSpace(creatorID) == "" {
		return ErrForbidden
	}

	pollID = strings.TrimSpace(pollID)
	if pollID == "" {
		return ErrPollNotFound
	}

	poll, err := s.repo.FindByID(ctx, pollID)
	if err != nil {
		return err
	}

	if poll.CreatorID != creatorID {
		return ErrForbidden
	}

	if err := s.repo.SoftDelete(ctx, pollID); err != nil {
		return fmt.Errorf("failed to delete poll: %w", err)
	}

	return nil
}
