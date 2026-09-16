package polls

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	MinQuestionLength = 3
	MaxQuestionLength = 255

	MinOptionsCount = 2
	MaxOptionsCount = 10

	MinOptionLength = 1
	MaxOptionLength = 150
)

// validateCreatePoll validates the payload for a new poll and returns sanitized values.
func validateCreatePoll(req CreatePollRequest) (string, []string, error) {
	question := strings.TrimSpace(req.Question)
	if len(question) < MinQuestionLength || len(question) > MaxQuestionLength {
		return "", nil, fmt.Errorf("question is required and must be between %d and %d characters", MinQuestionLength, MaxQuestionLength)
	}

	if len(req.Options) < MinOptionsCount || len(req.Options) > MaxOptionsCount {
		return "", nil, fmt.Errorf("poll must contain between %d and %d options", MinOptionsCount, MaxOptionsCount)
	}

	sanitizedOptions := make([]string, len(req.Options))
	seen := make(map[string]struct{}, len(req.Options))

	for i, opt := range req.Options {
		trimmed := strings.TrimSpace(opt)
		if len(trimmed) < MinOptionLength {
			return "", nil, errors.New("option text cannot be empty")
		}
		if len(trimmed) > MaxOptionLength {
			return "", nil, fmt.Errorf("option text cannot exceed %d characters", MaxOptionLength)
		}

		normalized := strings.ToLower(trimmed)
		if _, exists := seen[normalized]; exists {
			return "", nil, fmt.Errorf("duplicate option text detected: '%s'", trimmed)
		}
		seen[normalized] = struct{}{}
		sanitizedOptions[i] = trimmed
	}

	if req.ExpiresAt != nil && !req.ExpiresAt.After(time.Now().UTC()) {
		return "", nil, errors.New("poll expiration time must be in the future")
	}

	return question, sanitizedOptions, nil
}

// validateAndUpdatePoll validates update inputs against an existing active poll and updates fields safely.
func validateAndUpdatePoll(existing *Poll, req UpdatePollRequest) (*Poll, error) {
	if req.Question == nil && len(req.Options) == 0 {
		return nil, errors.New("at least one field (question or options) must be provided for update")
	}

	// Update question if provided
	if req.Question != nil {
		q := strings.TrimSpace(*req.Question)
		if len(q) < MinQuestionLength || len(q) > MaxQuestionLength {
			return nil, fmt.Errorf("question must be between %d and %d characters", MinQuestionLength, MaxQuestionLength)
		}
		existing.Question = q
	}

	// Update options if provided
	if len(req.Options) > 0 {
		if len(req.Options) != len(existing.Options) {
			return nil, fmt.Errorf("option count cannot change; all %d options must be provided to preserve stable identifiers", len(existing.Options))
		}

		// Check if IDs are specified or if positional updates are used
		hasIDs := false
		for _, opt := range req.Options {
			if strings.TrimSpace(opt.ID) != "" {
				hasIDs = true
				break
			}
		}

		updatedOptions := make([]PollOption, len(existing.Options))
		seen := make(map[string]struct{}, len(existing.Options))

		if hasIDs {
			// Map existing options by ID
			existingByID := make(map[string]PollOption, len(existing.Options))
			for _, o := range existing.Options {
				existingByID[o.ID] = o
			}

			for i, updateOpt := range req.Options {
				trimmedID := strings.TrimSpace(updateOpt.ID)
				if _, ok := existingByID[trimmedID]; !ok {
					return nil, fmt.Errorf("unknown option ID '%s'", trimmedID)
				}

				trimmedText := strings.TrimSpace(updateOpt.Text)
				if len(trimmedText) < MinOptionLength {
					return nil, errors.New("option text cannot be empty")
				}
				if len(trimmedText) > MaxOptionLength {
					return nil, fmt.Errorf("option text cannot exceed %d characters", MaxOptionLength)
				}

				normalized := strings.ToLower(trimmedText)
				if _, exists := seen[normalized]; exists {
					return nil, fmt.Errorf("duplicate option text detected: '%s'", trimmedText)
				}
				seen[normalized] = struct{}{}

				updatedOptions[i] = PollOption{
					ID:   trimmedID,
					Text: trimmedText,
				}
			}
		} else {
			// Positional update preserving existing stable IDs
			for i, updateOpt := range req.Options {
				trimmedText := strings.TrimSpace(updateOpt.Text)
				if len(trimmedText) < MinOptionLength {
					return nil, errors.New("option text cannot be empty")
				}
				if len(trimmedText) > MaxOptionLength {
					return nil, fmt.Errorf("option text cannot exceed %d characters", MaxOptionLength)
				}

				normalized := strings.ToLower(trimmedText)
				if _, exists := seen[normalized]; exists {
					return nil, fmt.Errorf("duplicate option text detected: '%s'", trimmedText)
				}
				seen[normalized] = struct{}{}

				updatedOptions[i] = PollOption{
					ID:   existing.Options[i].ID, // Maintain original stable ID
					Text: trimmedText,
				}
			}
		}

		existing.Options = updatedOptions
	}

	return existing, nil
}
