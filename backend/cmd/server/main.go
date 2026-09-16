package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pulsepoll/backend/internal/auth"
	"pulsepoll/backend/internal/config"
	"pulsepoll/backend/internal/database/mongodb"
	"pulsepoll/backend/internal/database/redis"
	"pulsepoll/backend/internal/polls"
	"pulsepoll/backend/internal/server"
	"pulsepoll/backend/internal/votes"
)

func main() {
	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load and validate environment configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Configuration validation failed at startup", slog.String("error", err.Error()))
		os.Exit(1)
	}

	slog.Info("Configuration loaded successfully",
		slog.String("env", cfg.Env),
		slog.String("port", cfg.Port),
		slog.String("mongodb_database", cfg.MongoDBName),
	)

	// Connect to MongoDB
	mongoCtx, mongoCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer mongoCancel()

	mongoClient, err := mongodb.Connect(mongoCtx, cfg.MongoURI, cfg.MongoDBName)
	if err != nil {
		slog.Error("Failed to initialize MongoDB connection", slog.String("error", err.Error()))
		os.Exit(1)
	}
	slog.Info("MongoDB connection established and verified")

	// Connect to Redis
	redisCtx, redisCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer redisCancel()

	redisClient, err := redis.Connect(redisCtx, cfg.RedisURL)
	if err != nil {
		slog.Error("Failed to initialize Redis connection", slog.String("error", err.Error()))
		_ = mongoClient.Close(context.Background())
		os.Exit(1)
	}
	slog.Info("Redis connection established and verified")

	// Initialize User repository and ensure indexes
	userRepo := auth.NewMongoUserRepository(mongoClient.Database())
	if err := userRepo.EnsureIndexes(mongoCtx); err != nil {
		slog.Error("Failed to ensure user collection indexes", slog.String("error", err.Error()))
		_ = redisClient.Close()
		_ = mongoClient.Close(context.Background())
		os.Exit(1)
	}
	slog.Info("MongoDB user indexes verified")

	// Initialize Poll repository and ensure indexes
	pollRepo := polls.NewMongoPollRepository(mongoClient.Database())
	if err := pollRepo.EnsureIndexes(mongoCtx); err != nil {
		slog.Error("Failed to ensure poll collection indexes", slog.String("error", err.Error()))
		_ = redisClient.Close()
		_ = mongoClient.Close(context.Background())
		os.Exit(1)
	}
	slog.Info("MongoDB poll indexes verified")

	// Initialize Vote repository and ensure unique compound constraint
	voteRepo := votes.NewMongoVoteRepository(mongoClient.Database())
	if err := voteRepo.EnsureIndexes(mongoCtx); err != nil {
		slog.Error("Failed to ensure vote collection indexes", slog.String("error", err.Error()))
		_ = redisClient.Close()
		_ = mongoClient.Close(context.Background())
		os.Exit(1)
	}
	slog.Info("MongoDB vote indexes verified")

	// Initialize Redis atomic vote counter store
	redisVoteRepo := votes.NewRedisVoteRepository(redisClient.Raw())

	// Initialize JWT manager, auth service, and HTTP handler
	jwtMgr := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTExpiryHours)
	authService := auth.NewService(userRepo, jwtMgr)
	authHandler := auth.NewHandler(authService)

	// Initialize Polls service and HTTP handler
	pollService := polls.NewService(pollRepo)
	pollHandler := polls.NewHandler(pollService)

	// Initialize Votes service and HTTP handler
	voteService := votes.NewService(pollRepo, voteRepo, redisVoteRepo)
	voteHandler := votes.NewHandler(voteService)

	// Set up router and HTTP server
	router := server.SetupRouter(cfg, mongoClient, redisClient, authHandler, pollHandler, voteHandler, jwtMgr)
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP server in a separate goroutine
	go func() {
		slog.Info("HTTP server listening", slog.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// Graceful shutdown on SIGINT / SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	slog.Info("Shutdown signal received, shutting down gracefully...", slog.String("signal", sig.String()))

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// 1. Shutdown HTTP server (drains in-flight requests)
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("Error during HTTP server shutdown", slog.String("error", err.Error()))
	} else {
		slog.Info("HTTP server stopped gracefully")
	}

	// 2. Disconnect MongoDB
	if err := mongoClient.Close(shutdownCtx); err != nil {
		slog.Error("Error disconnecting MongoDB", slog.String("error", err.Error()))
	} else {
		slog.Info("MongoDB connection closed cleanly")
	}

	// 3. Disconnect Redis
	if err := redisClient.Close(); err != nil {
		slog.Error("Error closing Redis connection", slog.String("error", err.Error()))
	} else {
		slog.Info("Redis connection closed cleanly")
	}

	slog.Info("PulsePoll backend shutdown complete")
}
