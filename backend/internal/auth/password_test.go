package auth

import (
	"testing"
)

func TestPasswordHashing(t *testing.T) {
	raw := "SecureP@ssw0rd123"

	hash, err := HashPassword(raw)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if hash == raw {
		t.Fatalf("hash must not equal raw password")
	}

	if !CheckPassword(raw, hash) {
		t.Errorf("expected CheckPassword to return true for matching password")
	}

	if CheckPassword("WrongPassword!", hash) {
		t.Errorf("expected CheckPassword to return false for mismatched password")
	}
}

func TestPasswordEmpty(t *testing.T) {
	_, err := HashPassword("")
	if err == nil {
		t.Errorf("expected error when hashing empty password")
	}
}
