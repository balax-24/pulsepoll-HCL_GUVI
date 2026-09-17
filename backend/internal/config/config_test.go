package config

import (
	"os"
	"testing"
)

func TestConfigValidation_Success(t *testing.T) {
	cfg := &Config{
		Port:           "8080",
		Env:            "development",
		MongoURI:       "mongodb://localhost:27017",
		MongoDBName:    "pulsepoll",
		RedisURL:       "redis://localhost:6380",
		JWTSecret:      "test-secret-must-be-at-least-32-chars-long",
		JWTExpiryHours: 24,
		AllowedOrigins: []string{"http://localhost:5173"},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid config, got error: %v", err)
	}

	if cfg.IsProduction() {
		t.Errorf("expected IsProduction() to be false for development")
	}
}

func TestConfigValidation_MissingRequired(t *testing.T) {
	testCases := []struct {
		name        string
		modify      func(c *Config)
		expectedErr string
	}{
		{
			name: "missing MongoURI",
			modify: func(c *Config) {
				c.MongoURI = ""
			},
			expectedErr: "MONGODB_URI",
		},
		{
			name: "missing RedisURL",
			modify: func(c *Config) {
				c.RedisURL = ""
			},
			expectedErr: "REDIS_URL",
		},
		{
			name: "missing JWTSecret",
			modify: func(c *Config) {
				c.JWTSecret = ""
			},
			expectedErr: "JWT_SECRET",
		},
		{
			name: "invalid JWTExpiryHours",
			modify: func(c *Config) {
				c.JWTExpiryHours = 0
			},
			expectedErr: "JWT_EXPIRY_HOURS must be greater than 0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{
				Port:           "8080",
				Env:            "development",
				MongoURI:       "mongodb://localhost:27017",
				MongoDBName:    "pulsepoll",
				RedisURL:       "redis://localhost:6380",
				JWTSecret:      "test-secret-must-be-at-least-32-chars-long",
				JWTExpiryHours: 24,
				AllowedOrigins: []string{"http://localhost:5173"},
			}
			tc.modify(cfg)
			err := cfg.Validate()
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.expectedErr)
			}
			if !containsSubstring(err.Error(), tc.expectedErr) {
				t.Errorf("expected error to contain %q, got %q", tc.expectedErr, err.Error())
			}
		})
	}
}

func TestConfigLoadFromEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("PORT", "9090")
	os.Setenv("ENV", "production")
	os.Setenv("MONGODB_URI", "mongodb://test:27017")
	os.Setenv("MONGODB_DATABASE", "testdb")
	os.Setenv("REDIS_URL", "redis://test:6380")
	os.Setenv("JWT_SECRET", "production-jwt-secret-at-least-32-chars")
	os.Setenv("JWT_EXPIRY_HOURS", "48")
	os.Setenv("ALLOWED_ORIGINS", "https://example.com, https://app.example.com")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("ENV")
		os.Unsetenv("MONGODB_URI")
		os.Unsetenv("MONGODB_DATABASE")
		os.Unsetenv("REDIS_URL")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("JWT_EXPIRY_HOURS")
		os.Unsetenv("ALLOWED_ORIGINS")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	if cfg.Port != "9090" {
		t.Errorf("expected port 9090, got %s", cfg.Port)
	}
	if !cfg.IsProduction() {
		t.Errorf("expected IsProduction() to be true")
	}
	if cfg.MongoDBName != "testdb" {
		t.Errorf("expected db testdb, got %s", cfg.MongoDBName)
	}
	if cfg.JWTExpiryHours != 48 {
		t.Errorf("expected expiry 48, got %d", cfg.JWTExpiryHours)
	}
	if len(cfg.AllowedOrigins) != 2 || cfg.AllowedOrigins[0] != "https://example.com" {
		t.Errorf("unexpected allowed origins: %v", cfg.AllowedOrigins)
	}
}

func TestConfigValidation_ShortJWTSecretProduction(t *testing.T) {
	cfg := &Config{
		Port:           "8080",
		Env:            "production",
		MongoURI:       "mongodb://localhost:27017",
		MongoDBName:    "pulsepoll",
		RedisURL:       "redis://localhost:6380",
		JWTSecret:      "short",
		JWTExpiryHours: 24,
		AllowedOrigins: []string{"https://example.com"},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for short JWT secret in production, got nil")
	}
	if !containsSubstring(err.Error(), "JWT_SECRET must be at least 32 characters") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestConfigValidation_ShortJWTSecretDevelopment(t *testing.T) {
	cfg := &Config{
		Port:           "8080",
		Env:            "development",
		MongoURI:       "mongodb://localhost:27017",
		MongoDBName:    "pulsepoll",
		RedisURL:       "redis://localhost:6380",
		JWTSecret:      "short",
		JWTExpiryHours: 24,
		AllowedOrigins: []string{"http://localhost:5173"},
	}

	// Development mode should succeed with a warning, not fail
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected short JWT secret to pass in development, got error: %v", err)
	}
}

func TestConfig_TrustedProxiesDefaultProduction(t *testing.T) {
	os.Clearenv()
	os.Setenv("PORT", "8080")
	os.Setenv("ENV", "production")
	os.Setenv("MONGODB_URI", "mongodb://localhost:27017")
	os.Setenv("REDIS_URL", "redis://localhost:6379")
	os.Setenv("JWT_SECRET", "production-jwt-secret-at-least-32-chars")
	os.Setenv("ALLOWED_ORIGINS", "https://app.example.com")
	defer os.Clearenv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// In production without TRUSTED_PROXIES set, should default to loopback + RFC1918
	foundRFC1918 := false
	for _, p := range cfg.TrustedProxies {
		if p == "10.0.0.0/8" || p == "172.16.0.0/12" {
			foundRFC1918 = true
			break
		}
	}
	if !foundRFC1918 {
		t.Errorf("expected production trusted proxies to include RFC1918 subnets, got %v", cfg.TrustedProxies)
	}
}

func TestConfig_TrustedProxiesDefaultDevelopment(t *testing.T) {
	os.Clearenv()
	os.Setenv("PORT", "8080")
	os.Setenv("ENV", "development")
	os.Setenv("MONGODB_URI", "mongodb://localhost:27017")
	os.Setenv("REDIS_URL", "redis://localhost:6379")
	os.Setenv("JWT_SECRET", "dev-secret-32-characters-or-more-here")
	os.Setenv("ALLOWED_ORIGINS", "http://localhost:5173")
	defer os.Clearenv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// In development without TRUSTED_PROXIES set, should default to loopback only
	if len(cfg.TrustedProxies) != 2 || cfg.TrustedProxies[0] != "127.0.0.1" {
		t.Errorf("expected development trusted proxies to be loopback only, got %v", cfg.TrustedProxies)
	}
}

func TestConfig_TrustedProxiesCustom(t *testing.T) {
	os.Clearenv()
	os.Setenv("PORT", "8080")
	os.Setenv("ENV", "production")
	os.Setenv("MONGODB_URI", "mongodb://localhost:27017")
	os.Setenv("REDIS_URL", "redis://localhost:6379")
	os.Setenv("JWT_SECRET", "production-jwt-secret-at-least-32-chars")
	os.Setenv("ALLOWED_ORIGINS", "https://app.example.com")
	os.Setenv("TRUSTED_PROXIES", "192.168.1.0/24, 10.0.1.5")
	defer os.Clearenv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(cfg.TrustedProxies) != 2 || cfg.TrustedProxies[0] != "192.168.1.0/24" || cfg.TrustedProxies[1] != "10.0.1.5" {
		t.Errorf("expected custom trusted proxies, got %v", cfg.TrustedProxies)
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && (s[:len(substr)] == substr || containsSubstring(s[1:], substr))))
}
