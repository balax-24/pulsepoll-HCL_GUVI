package server

import (
	"net/http"

	"pulsepoll/backend/internal/auth"
	"pulsepoll/backend/internal/config"
	"pulsepoll/backend/internal/database/mongodb"
	"pulsepoll/backend/internal/database/redis"
	"pulsepoll/backend/internal/health"
	"pulsepoll/backend/internal/middleware"
	"pulsepoll/backend/internal/polls"
	"pulsepoll/backend/internal/response"
	"pulsepoll/backend/internal/votes"
	"pulsepoll/backend/internal/websocket"

	"github.com/gin-gonic/gin"
)

// SetupRouter initializes and configures the Gin HTTP engine with middleware, health checks, and API routes.
func SetupRouter(
	cfg *config.Config,
	mongoClient *mongodb.Client,
	redisClient *redis.Client,
	authHandler *auth.Handler,
	pollHandler *polls.Handler,
	voteHandler *votes.Handler,
	wsHandler *websocket.Handler,
	jwtMgr *auth.JWTManager,
) *gin.Engine {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.New()
	// Configure trusted reverse proxies from configuration.
	// In production, trusts loopback plus RFC1918 private subnets used by container platforms (Render).
	// In development, defaults strictly to local loopback (127.0.0.1, ::1).
	_ = r.SetTrustedProxies(cfg.TrustedProxies)

	// Global middleware
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.RequestBodyLimit(2 << 20)) // 2 MB body limit defends against memory exhaustion
	r.Use(middleware.CORS(cfg.AllowedOrigins))

	// Custom 404 & 405 handlers conforming to standard error JSON envelope
	r.NoRoute(func(c *gin.Context) {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "The requested endpoint was not found")
	})
	r.NoMethod(func(c *gin.Context) {
		response.Error(c, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "HTTP method not allowed on this endpoint")
	})

	// Health check handlers
	healthHandler := health.NewHandler(mongoClient, redisClient)
	r.GET("/health", healthHandler.Check)

	// API root group
	api := r.Group("/api")
	{
		api.GET("/health", healthHandler.Check)

		// Auth group — rate limited to prevent credential brute-force and registration spam.
		// 5 requests per minute per IP with burst of 5 (covers legitimate login retries).
		authGroup := api.Group("/auth")
		authGroup.Use(middleware.RateLimit(5.0/60.0, 5))
		{
			authGroup.POST("/register", authHandler.Register)
			authGroup.POST("/login", authHandler.Login)
			authGroup.GET("/me", middleware.Authenticate(jwtMgr), authHandler.Me)
		}

		// Creator's polls endpoint: GET /api/my/polls
		api.GET("/my/polls", middleware.Authenticate(jwtMgr), pollHandler.GetMyPolls)

		// Polls group
		pollsGroup := api.Group("/polls")
		{
			// Public audience endpoints: GET poll, GET results, POST vote, GET ws
			pollsGroup.GET("/:id", pollHandler.GetPublic)
			pollsGroup.GET("/:id/results", voteHandler.GetResults)
			// Vote endpoint — rate limited per IP + Poll ID to mitigate automated ballot stuffing
			// while accommodating legitimate shared NAT environments (e.g. college Wi-Fi, offices, live audiences).
			// 60 requests per minute with burst capacity of 30 per IP+poll.
			voteRateLimiter := middleware.RateLimitWithKey(func(c *gin.Context) string {
				return c.ClientIP() + ":" + c.Param("id")
			}, 60.0/60.0, 30)
			pollsGroup.POST("/:id/vote", voteRateLimiter, voteHandler.CastVote)
			pollsGroup.GET("/:id/ws", wsHandler.ServeWS)

			// Protected creator management endpoints
			protected := pollsGroup.Group("")
			protected.Use(middleware.Authenticate(jwtMgr))
			{
				protected.POST("", pollHandler.Create)
				protected.PATCH("/:id", pollHandler.Update)
				protected.PATCH("/:id/status", pollHandler.UpdateStatus)
				protected.POST("/:id/close", pollHandler.Close)
				protected.DELETE("/:id", pollHandler.Delete)
			}
		}
	}

	return r
}
