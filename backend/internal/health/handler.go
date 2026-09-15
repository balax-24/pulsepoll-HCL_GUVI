package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Pinger defines the interface for verifying dependency liveness.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Handler contains dependencies required for health status evaluation.
type Handler struct {
	mongoPinger Pinger
	redisPinger Pinger
}

// NewHandler constructs a new health Check Handler.
func NewHandler(mongoPinger, redisPinger Pinger) *Handler {
	return &Handler{
		mongoPinger: mongoPinger,
		redisPinger: redisPinger,
	}
}

// ServicesStatus maps service names to their current health string ("ok" or "unavailable").
type ServicesStatus struct {
	MongoDB string `json:"mongodb"`
	Redis   string `json:"redis"`
}

// Response represents the payload returned by health endpoints.
type Response struct {
	Status   string         `json:"status"`
	Services ServicesStatus `json:"services"`
}

// Check handles GET /health requests.
func (h *Handler) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	mongoStatus := "ok"
	if h.mongoPinger == nil || h.mongoPinger.Ping(ctx) != nil {
		mongoStatus = "unavailable"
	}

	redisStatus := "ok"
	if h.redisPinger == nil || h.redisPinger.Ping(ctx) != nil {
		redisStatus = "unavailable"
	}

	overallStatus := "ok"
	httpCode := http.StatusOK

	if mongoStatus != "ok" || redisStatus != "ok" {
		overallStatus = "degraded"
		httpCode = http.StatusServiceUnavailable
	}

	c.JSON(httpCode, Response{
		Status: overallStatus,
		Services: ServicesStatus{
			MongoDB: mongoStatus,
			Redis:   redisStatus,
		},
	})
}
