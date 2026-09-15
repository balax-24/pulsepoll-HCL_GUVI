package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all backend configuration parameters.
type Config struct {
	Port           string
	Env            string
	MongoURI       string
	MongoDBName    string
	RedisURL       string
	JWTSecret      string
	JWTExpiryHours int
	AllowedOrigins []string
}

// Load reads configuration from environment variables and an optional .env file.
// It enforces required fields and returns descriptive errors on failure.
func Load() (*Config, error) {
	// Attempt to load from .env if present (non-fatal if absent)
	_ = godotenv.Load(".env", "backend/.env")

	port := getEnvOrDefault("PORT", "8080")
	env := getEnvOrDefault("ENV", "development")
	mongoURI := strings.TrimSpace(os.Getenv("MONGODB_URI"))
	mongoDBName := getEnvOrDefault("MONGODB_DATABASE", "pulsepoll")
	redisURL := strings.TrimSpace(os.Getenv("REDIS_URL"))
	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))

	jwtExpiryStr := getEnvOrDefault("JWT_EXPIRY_HOURS", "72")
	jwtExpiryHours, err := strconv.Atoi(jwtExpiryStr)
	if err != nil || jwtExpiryHours <= 0 {
		return nil, fmt.Errorf("invalid JWT_EXPIRY_HOURS: must be a positive integer, got %q", jwtExpiryStr)
	}

	originsRaw := getEnvOrDefault("ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173")
	origins := parseOrigins(originsRaw)

	cfg := &Config{
		Port:           port,
		Env:            env,
		MongoURI:       mongoURI,
		MongoDBName:    mongoDBName,
		RedisURL:       redisURL,
		JWTSecret:      jwtSecret,
		JWTExpiryHours: jwtExpiryHours,
		AllowedOrigins: origins,
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks whether all required configuration values are provided and valid.
func (c *Config) Validate() error {
	var missing []string

	if c.Port == "" {
		missing = append(missing, "PORT")
	}
	if c.MongoURI == "" {
		missing = append(missing, "MONGODB_URI")
	}
	if c.MongoDBName == "" {
		missing = append(missing, "MONGODB_DATABASE")
	}
	if c.RedisURL == "" {
		missing = append(missing, "REDIS_URL")
	}
	if c.JWTSecret == "" {
		missing = append(missing, "JWT_SECRET")
	}
	if len(c.AllowedOrigins) == 0 {
		missing = append(missing, "ALLOWED_ORIGINS")
	}

	if len(missing) > 0 {
		return errors.New("missing required configuration: " + strings.Join(missing, ", "))
	}

	if c.JWTExpiryHours <= 0 {
		return errors.New("JWT_EXPIRY_HOURS must be greater than 0")
	}

	return nil
}

// IsProduction returns true if running in production mode.
func (c *Config) IsProduction() bool {
	return strings.ToLower(c.Env) == "production"
}

func getEnvOrDefault(key, fallback string) string {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback
	}
	return val
}

func parseOrigins(raw string) []string {
	var origins []string
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			origins = append(origins, item)
		}
	}
	return origins
}
