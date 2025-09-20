package delivery

import (
	"net/http"
	"time"

	"etlgo/pkg/logger"
	"etlgo/pkg/metrics"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type HTTPHandlers struct {
	logger  *logger.Logger
	metrics *metrics.Metrics
}

func NewHTTPHandlers(
	logger *logger.Logger,
	metrics *metrics.Metrics,
) *HTTPHandlers {
	return &HTTPHandlers{
		logger:  logger,
		metrics: metrics,
	}
}

// Health check
func (h *HTTPHandlers) HealthCheck(c *gin.Context) {
	start := time.Now()
	h.metrics.IncHTTPRequestsInFlight()
	defer h.metrics.DecHTTPRequestsInFlight()

	requestID := uuid.New().String()

	health := gin.H{
		"status":     "healthy",
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"service":    "etl-go",
		"version":    "1.0.0",
		"request_id": requestID,
	}

	h.metrics.RecordHTTPRequest("GET", "/health", "200", time.Since(start))
	c.JSON(http.StatusOK, health)
}
