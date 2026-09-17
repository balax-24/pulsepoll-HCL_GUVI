package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTGenerateAndValidate(t *testing.T) {
	secret := "test-secret-key-1234567890123456"
	mgr := NewJWTManager(secret, 24)

	userID := "60c72b2f9b1d8b2bad000001"
	email := "user@example.com"

	token, err := mgr.Generate(userID, email)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	if token == "" {
		t.Fatalf("expected non-empty token")
	}

	claims, err := mgr.Validate(token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("expected userID %s, got %s", userID, claims.UserID)
	}
	if claims.Email != email {
		t.Errorf("expected email %s, got %s", email, claims.Email)
	}
	if claims.Issuer != "pulsepoll" {
		t.Errorf("expected issuer pulsepoll, got %s", claims.Issuer)
	}
}

func TestJWTInvalidSecret(t *testing.T) {
	mgr1 := NewJWTManager("secret-key-one-1234567890123456", 24)
	mgr2 := NewJWTManager("secret-key-two-1234567890123456", 24)

	token, err := mgr1.Generate("u1", "user@example.com")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = mgr2.Validate(token)
	if err == nil {
		t.Errorf("expected validation to fail with mismatched signing secret")
	}
}

func TestJWTExpired(t *testing.T) {
	secret := []byte("test-secret-key-1234567890123456")
	now := time.Now().Add(-2 * time.Hour)

	claims := CustomClaims{
		UserID: "u1",
		Email:  "user@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "u1",
			Issuer:    "pulsepoll",
			IssuedAt:  jwt.NewNumericDate(now.Add(-1 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(now), // Expired 2 hours ago
		},
	}

	expiredTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := expiredTokenObj.SignedString(secret)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	mgr := NewJWTManager(string(secret), 24)
	_, err = mgr.Validate(tokenStr)
	if err == nil {
		t.Errorf("expected expired token to fail validation")
	}
}

func TestJWTForgedTampered(t *testing.T) {
	mgr := NewJWTManager("secret-key-1234567890123456", 24)

	token, err := mgr.Generate("u1", "user@example.com")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Tamper with token payload
	tamperedToken := token[:len(token)-5] + "XXXXX"

	_, err = mgr.Validate(tamperedToken)
	if err == nil {
		t.Errorf("expected tampered token to fail validation")
	}
}

func TestJWTEmptyOrWhitespaceToken(t *testing.T) {
	mgr := NewJWTManager("secret-key-1234567890123456", 24)

	for _, tc := range []string{"", "   ", "\t\n"} {
		_, err := mgr.Validate(tc)
		if err == nil {
			t.Errorf("expected validation to fail for empty or whitespace token %q", tc)
		}
	}
}

func TestJWTInvalidIssuer(t *testing.T) {
	secret := []byte("secret-key-1234567890123456")
	claims := CustomClaims{
		UserID: "u1",
		Email:  "user@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "u1",
			Issuer:    "malicious-foreign-issuer",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	}

	tokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := tokenObj.SignedString(secret)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	mgr := NewJWTManager(string(secret), 24)
	_, err = mgr.Validate(tokenStr)
	if err == nil {
		t.Fatalf("expected validation to fail for non-pulsepoll issuer")
	}
}

func TestJWTInvalidSubject(t *testing.T) {
	secret := []byte("secret-key-1234567890123456")
	claims := CustomClaims{
		UserID: "u1",
		Email:  "user@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "u2", // Mismatched subject vs UserID
			Issuer:    "pulsepoll",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	}

	tokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := tokenObj.SignedString(secret)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	mgr := NewJWTManager(string(secret), 24)
	_, err = mgr.Validate(tokenStr)
	if err == nil {
		t.Fatalf("expected validation to fail for mismatched subject")
	}
}

func TestJWTWrongSigningAlgorithm(t *testing.T) {
	// Attempt signing with "none" method
	claims := CustomClaims{
		UserID: "u1",
		Email:  "user@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "u1",
			Issuer:    "pulsepoll",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	}

	tokenObj := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenStr, err := tokenObj.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("failed to sign none token: %v", err)
	}

	mgr := NewJWTManager("secret-key-1234567890123456", 24)
	_, err = mgr.Validate(tokenStr)
	if err == nil {
		t.Fatalf("expected validation to reject alg:none token")
	}
}
