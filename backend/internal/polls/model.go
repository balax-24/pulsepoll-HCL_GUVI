package polls

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// PollStatus represents the lifecycle state of a poll.
type PollStatus string

const (
	// PollStatusActive indicates the poll is active and ready for interactions.
	PollStatusActive PollStatus = "active"

	// PollStatusClosed indicates the poll has closed and is in a read-only state.
	PollStatusClosed PollStatus = "closed"
)

// PollOption represents an individual choice inside a poll with a unique stable identifier.
type PollOption struct {
	ID   string `bson:"id" json:"id"`
	Text string `bson:"text" json:"text"`
}

// Poll represents the MongoDB document persistence structure for a poll.
type Poll struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	CreatorID string        `bson:"creator_id" json:"creator_id"`
	Question  string        `bson:"question" json:"question"`
	Options   []PollOption  `bson:"options" json:"options"`
	Status    PollStatus    `bson:"status" json:"status"`
	IsDeleted bool          `bson:"is_deleted" json:"-"`
	DeletedAt *time.Time    `bson:"deleted_at,omitempty" json:"-"`
	ExpiresAt *time.Time    `bson:"expires_at,omitempty" json:"expires_at,omitempty"`
	CreatedAt time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time     `bson:"updated_at" json:"updated_at"`
}

// PollResponse represents the complete poll profile returned to creators.
type PollResponse struct {
	ID        string       `json:"id"`
	CreatorID string       `json:"creator_id"`
	Question  string       `json:"question"`
	Options   []PollOption `json:"options"`
	Status    PollStatus   `json:"status"`
	ExpiresAt *time.Time   `json:"expires_at,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// PublicPollResponse represents the sanitized public audience view of a poll.
// Private creator references, internal soft-delete flags, and system timestamps are excluded.
type PublicPollResponse struct {
	ID        string       `json:"id"`
	Question  string       `json:"question"`
	Options   []PollOption `json:"options"`
	Status    PollStatus   `json:"status"`
	ExpiresAt *time.Time   `json:"expires_at,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
}

// ToResponse maps a Poll entity into a full creator-facing PollResponse DTO.
func (p *Poll) ToResponse() *PollResponse {
	return &PollResponse{
		ID:        p.ID.Hex(),
		CreatorID: p.CreatorID,
		Question:  p.Question,
		Options:   p.Options,
		Status:    p.Status,
		ExpiresAt: p.ExpiresAt,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

// ToPublicResponse maps a Poll entity into a sanitized audience-facing PublicPollResponse DTO.
func (p *Poll) ToPublicResponse() *PublicPollResponse {
	return &PublicPollResponse{
		ID:        p.ID.Hex(),
		Question:  p.Question,
		Options:   p.Options,
		Status:    p.Status,
		ExpiresAt: p.ExpiresAt,
		CreatedAt: p.CreatedAt,
	}
}

// CreatePollRequest specifies payload requirements for creating a new poll.
type CreatePollRequest struct {
	Question  string     `json:"question" binding:"required"`
	Options   []string   `json:"options" binding:"required"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// UpdateOptionRequest specifies option text updates while preserving stable option IDs.
type UpdateOptionRequest struct {
	ID   string `json:"id,omitempty"`
	Text string `json:"text" binding:"required"`
}

// UpdatePollRequest defines fields eligible for modification on active polls.
type UpdatePollRequest struct {
	Question *string               `json:"question,omitempty"`
	Options  []UpdateOptionRequest `json:"options,omitempty"`
}

// UpdateStatusRequest defines the target status for a poll transition.
type UpdateStatusRequest struct {
	Status PollStatus `json:"status" binding:"required"`
}

// ListPollsResponse encapsulates creator poll lists with pagination metrics.
type ListPollsResponse struct {
	Polls      []*PollResponse `json:"polls"`
	TotalCount int64           `json:"total_count"`
	Page       int64           `json:"page"`
	Limit      int64           `json:"limit"`
}
