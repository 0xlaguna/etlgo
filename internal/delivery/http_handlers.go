package delivery

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"etlgo/internal/usecase"
	"etlgo/pkg/logger"
	"etlgo/pkg/metrics"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// handles HTTP requests
type HTTPHandlers struct {
	etlService     *usecase.ETLService
	metricsService *usecase.MetricsService
	logger         *logger.Logger
	metrics        *metrics.Metrics
}

// creates new HTTP handlers
func NewHTTPHandlers(
	etlService *usecase.ETLService,
	metricsService *usecase.MetricsService,
	logger *logger.Logger,
	metrics *metrics.Metrics,
) *HTTPHandlers {
	return &HTTPHandlers{
		etlService:     etlService,
		metricsService: metricsService,
		logger:         logger,
		metrics:        metrics,
	}
}

// triggers the ETL pipeline
func (h *HTTPHandlers) IngestRun(c *gin.Context) {
	start := time.Now()
	h.metrics.IncHTTPRequestsInFlight()
	defer h.metrics.DecHTTPRequestsInFlight()

	// Generate request ID for tracing
	requestID := uuid.New().String()
	ctx := context.WithValue(c.Request.Context(), logger.RequestIDKey, requestID)

	log := h.logger.WithContext(ctx)
	log.Info("Starting ETL ingestion")

	// Parse since parameter
	var since *time.Time
	if sinceStr := c.Query("since"); sinceStr != "" {
		if parsedSince, err := time.Parse("2006-01-02", sinceStr); err != nil {
			h.metrics.RecordHTTPRequest("POST", "/ingest/run", "400", time.Since(start))
			c.JSON(http.StatusBadRequest, gin.H{
				"error":      "Invalid date format",
				"message":    "Date must be in YYYY-MM-DD format",
				"request_id": requestID,
			})
			return
		} else {
			since = &parsedSince
		}
	}

	// Run ETL pipeline
	if err := h.etlService.RunETL(ctx, since); err != nil {
		h.metrics.RecordHTTPRequest("POST", "/ingest/run", "500", time.Since(start))
		log.WithError(err).Error("ETL ingestion failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      "ETL ingestion failed",
			"message":    err.Error(),
			"request_id": requestID,
		})
		return
	}

	h.metrics.RecordHTTPRequest("POST", "/ingest/run", "200", time.Since(start))

	response := gin.H{
		"message":    "ETL ingestion completed successfully",
		"request_id": requestID,
	}

	if since != nil {
		response["since"] = since.Format("2006-01-02")
	}

	c.JSON(http.StatusOK, response)
}

// GetAPIInfo returns API v1 information and available endpoints
func (h *HTTPHandlers) GetAPIInfo(c *gin.Context) {
	start := time.Now()
	h.metrics.IncHTTPRequestsInFlight()
	defer h.metrics.DecHTTPRequestsInFlight()

	requestID := uuid.New().String()

	apiInfo := gin.H{
		"api_version": "v1",
		"service":     "ETL Service",
		"version":     "1.0.0",
		"description": "ETL service for processing Ads and CRM data into business metrics",
		"endpoints": gin.H{
			"ingest": gin.H{
				"description": "Trigger ETL pipeline to process data",
				"methods":     []string{"POST"},
				"endpoints": gin.H{
					"run": gin.H{
						"path":        "/api/v1/ingest/run",
						"description": "Run ETL pipeline with optional date filter",
						"parameters": gin.H{
							"since": "Optional date filter (YYYY-MM-DD format)",
						},
						"example": "/api/v1/ingest/run?since=2025-01-01",
					},
				},
			},
			"metrics": gin.H{
				"description": "Query business metrics with various filters",
				"methods":     []string{"GET"},
				"endpoints": gin.H{
					"channel": gin.H{
						"path":        "/api/v1/metrics/channel",
						"description": "Get metrics filtered by channel",
						"parameters": gin.H{
							"channel": "Required: Channel name (e.g., google_ads)",
							"from":    "Optional: Start date (YYYY-MM-DD)",
							"to":      "Optional: End date (YYYY-MM-DD)",
							"limit":   "Optional: Number of results (default: 100)",
							"offset":  "Optional: Pagination offset (default: 0)",
						},
						"example": "/api/v1/metrics/channel?channel=google_ads&from=2025-01-01&to=2025-01-31",
					},
					"funnel": gin.H{
						"path":        "/api/v1/metrics/funnel",
						"description": "Get metrics filtered by UTM campaign (funnel analysis)",
						"parameters": gin.H{
							"utm_campaign": "Required: UTM campaign name",
							"from":         "Optional: Start date (YYYY-MM-DD)",
							"to":           "Optional: End date (YYYY-MM-DD)",
							"limit":        "Optional: Number of results (default: 100)",
							"offset":       "Optional: Pagination offset (default: 0)",
						},
						"example": "/api/v1/metrics/funnel?utm_campaign=back_to_school&from=2025-01-01&to=2025-01-31",
					},
					"summary": gin.H{
						"path":        "/api/v1/metrics/summary",
						"description": "Get aggregated metrics summary for the last 30 days",
						"parameters":  gin.H{},
						"example":     "/api/v1/metrics/summary",
					},
				},
			},
			"export": gin.H{
				"description": "Export processed data to external systems",
				"methods":     []string{"POST"},
				"endpoints": gin.H{
					"run": gin.H{
						"path":        "/api/v1/export/run",
						"description": "Export metrics for a specific date",
						"parameters": gin.H{
							"date": "Required: Date to export (YYYY-MM-DD format)",
						},
						"example": "/api/v1/export/run?date=2025-01-01",
					},
				},
			},
		},
		"business_metrics": gin.H{
			"cpc":             "Cost Per Click (cost / clicks)",
			"cpa":             "Cost Per Acquisition (cost / leads)",
			"cvr_lead_to_opp": "Conversion Rate Lead to Opportunity (opportunities / leads)",
			"cvr_opp_to_won":  "Conversion Rate Opportunity to Won (closed_won / opportunities)",
			"roas":            "Return on Ad Spend (revenue / cost)",
		},
		"request_id": requestID,
	}

	h.metrics.RecordHTTPRequest("GET", "/api/v1", "200", time.Since(start))
	c.JSON(http.StatusOK, apiInfo)
}

// GetMetricsByChannel retrieves metrics filtered by channel
func (h *HTTPHandlers) GetMetricsByChannel(c *gin.Context) {
	start := time.Now()
	h.metrics.IncHTTPRequestsInFlight()
	defer h.metrics.DecHTTPRequestsInFlight()

	requestID := uuid.New().String()
	ctx := context.WithValue(c.Request.Context(), logger.RequestIDKey, requestID)

	// Parse query parameters
	channel := c.Query("channel")
	if channel == "" {
		h.metrics.RecordHTTPRequest("GET", "/metrics/channel", "400", time.Since(start))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "Missing required parameter",
			"message":    "channel parameter is required",
			"request_id": requestID,
		})
		return
	}

	from, to, limit, offset, err := h.parseMetricsParams(c)
	if err != nil {
		h.metrics.RecordHTTPRequest("GET", "/metrics/channel", "400", time.Since(start))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "Invalid parameters",
			"message":    err.Error(),
			"request_id": requestID,
		})
		return
	}

	// Get metrics
	response, err := h.metricsService.GetMetricsByChannel(ctx, channel, from, to, limit, offset)
	if err != nil {
		h.metrics.RecordHTTPRequest("GET", "/metrics/channel", "500", time.Since(start))
		h.logger.WithContext(ctx).WithError(err).Error("Failed to get metrics by channel")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      "Failed to retrieve metrics",
			"message":    err.Error(),
			"request_id": requestID,
		})
		return
	}

	h.metrics.RecordHTTPRequest("GET", "/metrics/channel", "200", time.Since(start))

	responseData := gin.H{
		"data":       response.Data,
		"total":      response.Total,
		"limit":      response.Limit,
		"offset":     response.Offset,
		"has_more":   response.HasMore,
		"request_id": requestID,
	}

	c.JSON(http.StatusOK, responseData)
}

// GetMetricsByFunnel retrieves metrics filtered by UTM campaign
func (h *HTTPHandlers) GetMetricsByFunnel(c *gin.Context) {
	start := time.Now()
	h.metrics.IncHTTPRequestsInFlight()
	defer h.metrics.DecHTTPRequestsInFlight()

	requestID := uuid.New().String()
	ctx := context.WithValue(c.Request.Context(), logger.RequestIDKey, requestID)

	// Parse query parameters
	utmCampaign := c.Query("utm_campaign")
	if utmCampaign == "" {
		h.metrics.RecordHTTPRequest("GET", "/metrics/funnel", "400", time.Since(start))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "Missing required parameter",
			"message":    "utm_campaign parameter is required",
			"request_id": requestID,
		})
		return
	}

	from, to, limit, offset, err := h.parseMetricsParams(c)
	if err != nil {
		h.metrics.RecordHTTPRequest("GET", "/metrics/funnel", "400", time.Since(start))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "Invalid parameters",
			"message":    err.Error(),
			"request_id": requestID,
		})
		return
	}

	// Get metrics
	response, err := h.metricsService.GetMetricsByFunnel(ctx, utmCampaign, from, to, limit, offset)
	if err != nil {
		h.metrics.RecordHTTPRequest("GET", "/metrics/funnel", "500", time.Since(start))
		h.logger.WithContext(ctx).WithError(err).Error("Failed to get metrics by funnel")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      "Failed to retrieve metrics",
			"message":    err.Error(),
			"request_id": requestID,
		})
		return
	}

	h.metrics.RecordHTTPRequest("GET", "/metrics/funnel", "200", time.Since(start))

	responseData := gin.H{
		"data":       response.Data,
		"total":      response.Total,
		"limit":      response.Limit,
		"offset":     response.Offset,
		"has_more":   response.HasMore,
		"request_id": requestID,
	}

	c.JSON(http.StatusOK, responseData)
}

// ExportRun exports metrics for a specific date
func (h *HTTPHandlers) ExportRun(c *gin.Context) {
	start := time.Now()
	h.metrics.IncHTTPRequestsInFlight()
	defer h.metrics.DecHTTPRequestsInFlight()

	requestID := uuid.New().String()
	ctx := context.WithValue(c.Request.Context(), logger.RequestIDKey, requestID)

	// Parse date parameter
	dateStr := c.Query("date")
	if dateStr == "" {
		h.metrics.RecordHTTPRequest("POST", "/export/run", "400", time.Since(start))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "Missing required parameter",
			"message":    "date parameter is required",
			"request_id": requestID,
		})
		return
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		h.metrics.RecordHTTPRequest("POST", "/export/run", "400", time.Since(start))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "Invalid date format",
			"message":    "Date must be in YYYY-MM-DD format",
			"request_id": requestID,
		})
		return
	}

	// Export metrics
	if err := h.metricsService.ExportMetrics(ctx, date); err != nil {
		h.metrics.RecordHTTPRequest("POST", "/export/run", "500", time.Since(start))
		h.logger.WithContext(ctx).WithError(err).Error("Failed to export metrics")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      "Export failed",
			"message":    err.Error(),
			"request_id": requestID,
		})
		return
	}

	h.metrics.RecordHTTPRequest("POST", "/export/run", "200", time.Since(start))

	c.JSON(http.StatusOK, gin.H{
		"message":    "Export completed successfully",
		"date":       date.Format("2006-01-02"),
		"request_id": requestID,
	})
}

// GetMetricsSummary returns a summary of available metrics
func (h *HTTPHandlers) GetMetricsSummary(c *gin.Context) {
	start := time.Now()
	h.metrics.IncHTTPRequestsInFlight()
	defer h.metrics.DecHTTPRequestsInFlight()

	requestID := uuid.New().String()
	ctx := context.WithValue(c.Request.Context(), logger.RequestIDKey, requestID)

	// Get summary
	summary, err := h.metricsService.GetMetricsSummary(ctx)
	if err != nil {
		h.metrics.RecordHTTPRequest("GET", "/metrics/summary", "500", time.Since(start))
		h.logger.WithContext(ctx).WithError(err).Error("Failed to get metrics summary")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      "Failed to retrieve summary",
			"message":    err.Error(),
			"request_id": requestID,
		})
		return
	}

	h.metrics.RecordHTTPRequest("GET", "/metrics/summary", "200", time.Since(start))

	summary["request_id"] = requestID
	c.JSON(http.StatusOK, summary)
}

// HealthCheck returns the health status of the service
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

// parseMetricsParams parses common query parameters for metrics endpoints
func (h *HTTPHandlers) parseMetricsParams(c *gin.Context) (from, to time.Time, limit, offset int, err error) {
	// Parse from parameter
	fromStr := c.Query("from")
	if fromStr == "" {
		from = time.Now().AddDate(0, 0, -365) // Default to last 365 days
	} else {
		from, err = time.Parse("2006-01-02", fromStr)
		if err != nil {
			return time.Time{}, time.Time{}, 0, 0, err
		}
	}

	// Parse to parameter
	toStr := c.Query("to")
	if toStr == "" {
		to = time.Now() // Default to now
	} else {
		to, err = time.Parse("2006-01-02", toStr)
		if err != nil {
			return time.Time{}, time.Time{}, 0, 0, err
		}
	}

	// Parse limit parameter
	limitStr := c.Query("limit")
	if limitStr == "" {
		limit = 100 // Default limit
	} else {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			return time.Time{}, time.Time{}, 0, 0, err
		}
	}

	// Parse offset parameter
	offsetStr := c.Query("offset")
	if offsetStr == "" {
		offset = 0 // Default offset
	} else {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil {
			return time.Time{}, time.Time{}, 0, 0, err
		}
	}

	return from, to, limit, offset, nil
}
