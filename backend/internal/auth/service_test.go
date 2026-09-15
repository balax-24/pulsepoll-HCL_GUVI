package auth

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// mockUserRepository provides an in-memory test double for UserRepository.
type mockUserRepository struct {
	users map[string]*User // key: normalized email
	byID  map[string]*User // key: hex ID
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users: make(map[string]*User),
		byID:  make(map[string]*User),
	}
}

func (m *mockUserRepository) Create(ctx context.Context, user *User) error {
	if user.ID.IsZero() {
		user.ID = bson.NewObjectID()
	}
	normEmail := strings.ToLower(user.Email)
	if _, exists := m.users[normEmail]; exists {
		return ErrDuplicateEmail
	}
	m.users[normEmail] = user
	m.byID[user.ID.Hex()] = user
	return nil
}

func (m *mockUserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	norm := strings.ToLower(email)
	if u, ok := m.users[norm]; ok {
		return u, nil
	}
	return nil, ErrUserNotFound
}

func (m *mockUserRepository) FindByID(ctx context.Context, id string) (*User, error) {
	if u, ok := m.byID[id]; ok {
		return u, nil
	}
	return nil, ErrUserNotFound
}

func (m *mockUserRepository) EnsureIndexes(ctx context.Context) error {
	return nil
}

func setupTestService() (*Service, *mockUserRepository, *JWTManager) {
	repo := newMockUserRepository()
	jwtMgr := NewJWTManager("super-secret-key-12345678901234", 24)
	svc := NewService(repo, jwtMgr)
	return svc, repo, jwtMgr
}

func TestRegister_Success(t *testing.T) {
	svc, repo, _ := setupTestService()
	ctx := context.Background()

	req := RegisterRequest{
		Name:     "Alice Engineer",
		Email:    "Alice@Example.Com",
		Password: "SuperSecurePassword123!",
	}

	userResp, err := svc.Register(ctx, req)
	if err != nil {
		t.Fatalf("expected successful registration, got error: %v", err)
	}

	if userResp.Name != "Alice Engineer" {
		t.Errorf("expected name 'Alice Engineer', got %s", userResp.Name)
	}
	if userResp.Email != "alice@example.com" {
		t.Errorf("expected normalized lowercase email, got %s", userResp.Email)
	}
	if userResp.ID == "" {
		t.Errorf("expected non-empty user ID")
	}

	// Verify database record
	storedUser, err := repo.FindByEmail(ctx, "alice@example.com")
	if err != nil {
		t.Fatalf("failed to find stored user: %v", err)
	}

	// Verify password is NOT stored as plaintext
	if storedUser.PasswordHash == req.Password {
		t.Fatalf("CRITICAL: password was stored in plaintext!")
	}
	if !CheckPassword(req.Password, storedUser.PasswordHash) {
		t.Errorf("stored hash does not match original password")
	}
}

func TestRegister_ValidationFailures(t *testing.T) {
	svc, _, _ := setupTestService()
	ctx := context.Background()

	testCases := []struct {
		name string
		req  RegisterRequest
	}{
		{
			name: "empty name",
			req:  RegisterRequest{Name: "", Email: "test@example.com", Password: "password123"},
		},
		{
			name: "single char name",
			req:  RegisterRequest{Name: "A", Email: "test@example.com", Password: "password123"},
		},
		{
			name: "invalid email format",
			req:  RegisterRequest{Name: "Valid Name", Email: "not-an-email", Password: "password123"},
		},
		{
			name: "email missing domain dot",
			req:  RegisterRequest{Name: "Valid Name", Email: "test@localhost", Password: "password123"},
		},
		{
			name: "short password (<8 chars)",
			req:  RegisterRequest{Name: "Valid Name", Email: "test@example.com", Password: "short"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Register(ctx, tc.req)
			if err == nil {
				t.Fatalf("expected validation error, got nil")
			}
			if !errors.Is(err, ErrInvalidInput) {
				t.Errorf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc, _, _ := setupTestService()
	ctx := context.Background()

	req := RegisterRequest{
		Name:     "User One",
		Email:    "unique@example.com",
		Password: "password12345",
	}

	_, err := svc.Register(ctx, req)
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	// Attempt duplicate registration with different casing
	dupReq := RegisterRequest{
		Name:     "User Two",
		Email:    "UNIQUE@example.com",
		Password: "differentPassword123",
	}

	_, err = svc.Register(ctx, dupReq)
	if err == nil {
		t.Fatalf("expected duplicate email error, got nil")
	}
	if !errors.Is(err, ErrDuplicateEmail) {
		t.Errorf("expected ErrDuplicateEmail, got %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	svc, _, _ := setupTestService()
	ctx := context.Background()

	regReq := RegisterRequest{
		Name:     "Bob Engineer",
		Email:    "bob@example.com",
		Password: "CorrectPassword123!",
	}
	_, err := svc.Register(ctx, regReq)
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	loginResp, err := svc.Login(ctx, LoginRequest{
		Email:    "BOB@example.com", // case-insensitive login
		Password: "CorrectPassword123!",
	})
	if err != nil {
		t.Fatalf("expected login success, got: %v", err)
	}

	if loginResp.Token == "" {
		t.Errorf("expected non-empty JWT token")
	}
	if loginResp.User.Email != "bob@example.com" {
		t.Errorf("expected user email bob@example.com, got %s", loginResp.User.Email)
	}
}

func TestLogin_Failures(t *testing.T) {
	svc, _, _ := setupTestService()
	ctx := context.Background()

	regReq := RegisterRequest{
		Name:     "Bob Engineer",
		Email:    "bob@example.com",
		Password: "CorrectPassword123!",
	}
	_, err := svc.Register(ctx, regReq)
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	t.Run("wrong password returns generic error", func(t *testing.T) {
		_, err := svc.Login(ctx, LoginRequest{
			Email:    "bob@example.com",
			Password: "WrongPassword999",
		})
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("expected generic ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("nonexistent email returns generic error", func(t *testing.T) {
		_, err := svc.Login(ctx, LoginRequest{
			Email:    "nonexistent@example.com",
			Password: "SomePassword123",
		})
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("expected generic ErrInvalidCredentials, got %v", err)
		}
	})
}

func TestGetProfile(t *testing.T) {
	svc, _, _ := setupTestService()
	ctx := context.Background()

	regReq := RegisterRequest{
		Name:     "Charlie",
		Email:    "charlie@example.com",
		Password: "ValidPassword123",
	}
	userResp, err := svc.Register(ctx, regReq)
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	t.Run("existing user", func(t *testing.T) {
		p, err := svc.GetProfile(ctx, userResp.ID)
		if err != nil {
			t.Fatalf("failed to get profile: %v", err)
		}
		if p.Name != "Charlie" || p.Email != "charlie@example.com" {
			t.Errorf("unexpected profile: %+v", p)
		}
	})

	t.Run("nonexistent user", func(t *testing.T) {
		_, err := svc.GetProfile(ctx, "nonexistent-id")
		if !errors.Is(err, ErrUserNotFound) {
			t.Errorf("expected ErrUserNotFound, got %v", err)
		}
	})
}
