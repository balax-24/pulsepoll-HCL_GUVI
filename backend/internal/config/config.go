package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// Config holds all backend configuration parameters.
type Config struct {
	Port            string
	Env             string
	MongoURI        string
	MongoDBName     string
	RedisURL        string
	JWTSecret       string
	JWTExpiryHours  int
	AllowedOrigins  []string
	TrustedProxies  []string
	TrustedPlatform string
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

	trustedProxiesRaw := strings.TrimSpace(os.Getenv("TRUSTED_PROXIES"))
	var trustedProxies []string
	if trustedProxiesRaw != "" {
		trustedProxies = parseOrigins(trustedProxiesRaw)
	} else if strings.EqualFold(env, "production") {
		// In production (Render / container orchestration), the reverse proxy
		// connects over internal private subnets (RFC1918). Trusting loopback plus
		// RFC1918 allows Gin to parse X-Forwarded-For and extract real client IPs.
		trustedProxies = []string{"127.0.0.1", "::1", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}
	} else {
		// In local development, only trust loopback
		trustedProxies = []string{"127.0.0.1", "::1"}
	}

	trustedPlatform := strings.TrimSpace(os.Getenv("TRUSTED_PLATFORM"))
	if trustedPlatform == "" && strings.EqualFold(env, "production") {
		trustedPlatform = gin.PlatformCloudflare
	}

	cfg := &Config{
		Port:            port,
		Env:             env,
		MongoURI:        mongoURI,
		MongoDBName:     mongoDBName,
		RedisURL:        redisURL,
		JWTSecret:       jwtSecret,
		JWTExpiryHours:  jwtExpiryHours,
		AllowedOrigins:  origins,
		TrustedProxies:  trustedProxies,
		TrustedPlatform: trustedPlatform,
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

	if len(c.TrustedProxies) == 0 {
		c.TrustedProxies = []string{"127.0.0.1", "::1"}
	}

	// Enforce minimum JWT secret length to prevent brute-force attacks against HS256.
	// Production: hard fail. Development: warn but allow startup.
	const minJWTSecretLength = 32
	if len(c.JWTSecret) < minJWTSecretLength {
		if c.IsProduction() {
			return fmt.Errorf("JWT_SECRET must be at least %d characters in production (got %d)", minJWTSecretLength, len(c.JWTSecret))
		}
		slog.Warn("JWT_SECRET is shorter than recommended minimum; acceptable for local development only",
			slog.Int("length", len(c.JWTSecret)),
			slog.Int("recommended_min", minJWTSecretLength),
		)
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
