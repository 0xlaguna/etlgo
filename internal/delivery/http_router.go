package delivery

import (
	"time"

	"etlgo/internal/delivery/middleware"
	"etlgo/pkg/logger"
	"etlgo/pkg/metrics"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type HTTPRouter struct {
	handlers *HTTPHandlers
	logger   *logger.Logger
	metrics  *metrics.Metrics
}

func NewHTTPRouter(handlers *HTTPHandlers, logger *logger.Logger, metrics *metrics.Metrics) *HTTPRouter {
	return &HTTPRouter{
		handlers: handlers,
		logger:   logger,
		metrics:  metrics,
	}
}

func (r *HTTPRouter) SetupRoutes() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	router.Use(middleware.RequestID())
	router.Use(middleware.Logger(r.logger))
	router.Use(middleware.Recovery(r.logger))
	router.Use(middleware.Metrics(r.metrics))
	router.Use(middleware.Timeout(30 * time.Second))

	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Content-Type", "X-Request-ID"}
	config.ExposeHeaders = []string{"X-Request-ID"}

	router.Use(cors.New(config))

	// Health endpoint
	router.GET("/health", r.handlers.HealthCheck)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		v1.GET("/", r.handlers.GetAPIInfo)
		v1.GET("", r.handlers.GetAPIInfo)

		// ETL endpoints
		etl := v1.Group("/ingest")
		{
			etl.POST("/run", r.handlers.IngestRun)
		}

		// Metrics endpoints
		metricsGroup := v1.Group("/metrics")
		{
			metricsGroup.GET("/channel", r.handlers.GetMetricsByChannel)
			metricsGroup.GET("/funnel", r.handlers.GetMetricsByFunnel)
			metricsGroup.GET("/summary", r.handlers.GetMetricsSummary)
		}

		// Export endpoints
		export := v1.Group("/export")
		{
			export.POST("/run", r.handlers.ExportRun)
		}
	}

	// Prometheus metrics endpoint
	router.GET("/metrics", middleware.PrometheusHandler())

	return router
}
