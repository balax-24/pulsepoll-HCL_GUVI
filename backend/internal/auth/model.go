package auth

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// User represents the persistent user account document in MongoDB.
type User struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Name         string        `bson:"name" json:"name"`
	Email        string        `bson:"email" json:"email"`
	PasswordHash string        `bson:"password_hash" json:"-"`
	CreatedAt    time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time     `bson:"updated_at" json:"updated_at"`
}

// UserResponse represents safe, public-facing user profile information.
// Passwords, hashes, and secrets are strictly excluded.
type UserResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// ToResponse converts a User entity into a safe UserResponse DTO.
func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:        u.ID.Hex(),
		Name:      u.Name,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
	}
}

// RegisterRequest holds the payload for registering a new account.
type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginRequest holds credentials for user login.
type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse contains the issued JWT token and user profile.
type LoginResponse struct {
	Token string        `json:"token"`
	User  *UserResponse `json:"user"`
}
