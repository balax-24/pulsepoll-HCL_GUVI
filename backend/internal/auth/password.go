package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a plaintext password using bcrypt with standard cost (10).
func HashPassword(password string) (string, error) {
	if len(password) == 0 {
		return "", fmt.Errorf("password cannot be empty")
	}
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(bytes), nil
}

// CheckPassword compares a plaintext password with a bcrypt hashed password.
// Returns true if the password matches, false otherwise.
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
