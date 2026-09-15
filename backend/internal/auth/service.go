package auth

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
)

var (
	// ErrInvalidCredentials is a generic error returned for authentication failures
	// to prevent account enumeration / timing discrepancies.
	ErrInvalidCredentials = errors.New("invalid email or password")

	// ErrInvalidInput indicates validation failure on supplied fields.
	ErrInvalidInput = errors.New("validation failed")
)

// Service handles domain logic for authentication and user management.
type Service struct {
	repo   UserRepository
	jwtMgr *JWTManager
}

// NewService constructs a new authentication Service.
func NewService(repo UserRepository, jwtMgr *JWTManager) *Service {
	return &Service{
		repo:   repo,
		jwtMgr: jwtMgr,
	}
}

// Register validates registration parameters, normalizes the email, checks for uniqueness,
// securely hashes the password with bcrypt, and stores the user record.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (*UserResponse, error) {
	name := strings.TrimSpace(req.Name)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	password := req.Password

	if err := validateRegistration(name, email, password); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidInput, err.Error())
	}

	// Check if user already exists
	existing, err := s.repo.FindByEmail(ctx, email)
	if err == nil && existing != nil {
		return nil, ErrDuplicateEmail
	} else if err != nil && !errors.Is(err, ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check existing email: %w", err)
	}

	// Hash password
	passwordHash, err := HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &User{
		Name:         name,
		Email:        email,
		PasswordHash: passwordHash,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user.ToResponse(), nil
}

// Login validates credentials, verifies the bcrypt password hash, and issues a signed JWT.
func (s *Service) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	password := req.Password

	if email == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to lookup user: %w", err)
	}

	if !CheckPassword(password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	token, err := s.jwtMgr.Generate(user.ID.Hex(), user.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate authentication token: %w", err)
	}

	return &LoginResponse{
		Token: token,
		User:  user.ToResponse(),
	}, nil
}

// GetProfile retrieves safe profile details for an authenticated user ID.
func (s *Service) GetProfile(ctx context.Context, userID string) (*UserResponse, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, ErrUserNotFound
	}

	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return user.ToResponse(), nil
}

func validateRegistration(name, email, password string) error {
	if len(name) < 2 || len(name) > 100 {
		return errors.New("name must be between 2 and 100 characters")
	}

	if email == "" || len(email) > 255 {
		return errors.New("email is required and must not exceed 255 characters")
	}

	if _, err := mail.ParseAddress(email); err != nil || !strings.Contains(email, ".") {
		return errors.New("invalid email address format")
	}

	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}

	if len(password) > 72 {
		return errors.New("password exceeds maximum allowed length (72 characters)")
	}

	return nil
}
