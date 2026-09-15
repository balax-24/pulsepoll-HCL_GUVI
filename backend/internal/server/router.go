package server

import (
	"net/http"

	"pulsepoll/backend/internal/config"
	"pulsepoll/backend/internal/database/mongodb"
	"pulsepoll/backend/internal/database/redis"
	"pulsepoll/backend/internal/health"
	"pulsepoll/backend/internal/middleware"
	"pulsepoll/backend/internal/response"

	"github.com/gin-gonic/gin"
)

// SetupRouter initializes and configures the Gin HTTP engine with middleware and core routes.
func SetupRouter(cfg *config.Config, mongoClient *mongodb.Client, redisClient *redis.Client) *gin.Engine {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.New()

	// Global middleware
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS(cfg.AllowedOrigins))

	// Custom 404 & 405 handlers conforming to standard error JSON envelope
	r.NoRoute(func(c *gin.Context) {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "The requested endpoint was not found")
	})
	r.NoMethod(func(c *gin.Context) {
		response.Error(c, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "HTTP method not allowed on this endpoint")
	})

	// Health check handler
	healthHandler := health.NewHandler(mongoClient, redisClient)
	r.GET("/health", healthHandler.Check)

	// API group
	api := r.Group("/api")
	{
		api.GET("/health", healthHandler.Check)
	}

	return r
}
