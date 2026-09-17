package auth

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// CustomClaims encapsulates user claims and standard JWT registered claims.
type CustomClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// JWTManager handles signing and validating JWT authentication tokens.
type JWTManager struct {
	secret []byte
	expiry time.Duration
}

// NewJWTManager creates a new JWTManager with the provided secret and expiry hours.
func NewJWTManager(secret string, expiryHours int) *JWTManager {
	return &JWTManager{
		secret: []byte(secret),
		expiry: time.Duration(expiryHours) * time.Hour,
	}
}

// Generate issues a new signed JWT string containing user credentials and expiry claims.
func (m *JWTManager) Generate(userID, email string) (string, error) {
	if len(m.secret) == 0 {
		return "", errors.New("jwt secret is not configured")
	}

	now := time.Now()
	claims := CustomClaims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    "pulsepoll",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.expiry)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}

// Validate verifies the token signature and claims, ensuring it has not expired or been forged.
func (m *JWTManager) Validate(tokenString string) (*CustomClaims, error) {
	tokenString = strings.TrimSpace(tokenString)
	if tokenString == "" {
		return nil, errors.New("token string cannot be empty")
	}

	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (any, error) {
		// Enforce HMAC algorithm to prevent alg:none and asymmetric confusion attacks
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid or expired token claims")
	}

	// Defense-in-depth: verify issuer matches pulsepoll
	if claims.Issuer != "pulsepoll" {
		return nil, errors.New("invalid token issuer")
	}

	// Defense-in-depth: verify subject claim exists and matches userID
	if claims.Subject == "" || claims.Subject != claims.UserID {
		return nil, errors.New("invalid token subject")
	}

	return claims, nil
}
