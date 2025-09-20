package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	// HTTP metrics
	HTTPRequestsTotal    *prometheus.CounterVec
	HTTPRequestDuration  *prometheus.HistogramVec
	HTTPRequestsInFlight prometheus.Gauge

	// ETL metrics
	ETLJobsTotal        *prometheus.CounterVec
	ETLJobDuration      *prometheus.HistogramVec
	ETLJobsInProgress   prometheus.Gauge
	ETLRecordsProcessed *prometheus.CounterVec
	ETLRecordsFailed    *prometheus.CounterVec

	// External API metrics
	ExternalAPICalls    *prometheus.CounterVec
	ExternalAPIDuration *prometheus.HistogramVec
	ExternalAPIFailures *prometheus.CounterVec

	// Business metrics
	BusinessMetricsCalculated *prometheus.CounterVec
}

func New() *Metrics {
	return &Metrics{
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status_code"},
		),

		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "endpoint"},
		),

		HTTPRequestsInFlight: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "http_requests_in_flight",
				Help: "Number of HTTP requests currently being processed",
			},
		),

		ETLJobsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "etl_jobs_total",
				Help: "Total number of ETL jobs",
			},
			[]string{"status", "source"},
		),

		ETLJobDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "etl_job_duration_seconds",
				Help:    "ETL job duration in seconds",
				Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600},
			},
			[]string{"source"},
		),

		ETLJobsInProgress: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "etl_jobs_in_progress",
				Help: "Number of ETL jobs currently in progress",
			},
		),

		ETLRecordsProcessed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "etl_records_processed_total",
				Help: "Total number of records processed by ETL",
			},
			[]string{"source", "status"},
		),

		ETLRecordsFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "etl_records_failed_total",
				Help: "Total number of records that failed processing",
			},
			[]string{"source", "error_type"},
		),

		ExternalAPICalls: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "external_api_calls_total",
				Help: "Total number of external API calls",
			},
			[]string{"api", "status"},
		),

		ExternalAPIDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "external_api_duration_seconds",
				Help:    "External API call duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"api"},
		),

		ExternalAPIFailures: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "external_api_failures_total",
				Help: "Total number of external API failures",
			},
			[]string{"api", "error_type"},
		),

		BusinessMetricsCalculated: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "business_metrics_calculated_total",
				Help: "Total number of business metrics calculated",
			},
			[]string{"metric_type"},
		),
	}
}

// HTTP request metrics
func (m *Metrics) RecordHTTPRequest(method, endpoint, statusCode string, duration time.Duration) {
	m.HTTPRequestsTotal.WithLabelValues(method, endpoint, statusCode).Inc()
	m.HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

// ETL job metrics
func (m *Metrics) RecordETLJob(status, source string, duration time.Duration) {
	m.ETLJobsTotal.WithLabelValues(status, source).Inc()
	m.ETLJobDuration.WithLabelValues(source).Observe(duration.Seconds())
}

// ETL record processing metrics
func (m *Metrics) RecordETLRecords(source, status string, count int) {
	m.ETLRecordsProcessed.WithLabelValues(source, status).Add(float64(count))
}

// ETL record failure metrics
func (m *Metrics) RecordETLRecordFailure(source, errorType string) {
	m.ETLRecordsFailed.WithLabelValues(source, errorType).Inc()
}

// External API call metrics
func (m *Metrics) RecordExternalAPICall(api, status string, duration time.Duration) {
	m.ExternalAPICalls.WithLabelValues(api, status).Inc()
	m.ExternalAPIDuration.WithLabelValues(api).Observe(duration.Seconds())
}

// External API failure metrics
func (m *Metrics) RecordExternalAPIFailure(api, errorType string) {
	m.ExternalAPIFailures.WithLabelValues(api, errorType).Inc()
}

// Business metric calculation
func (m *Metrics) RecordBusinessMetric(metricType string) {
	m.BusinessMetricsCalculated.WithLabelValues(metricType).Inc()
}

// ETL jobs in progress counter
func (m *Metrics) IncETLJobsInProgress() {
	m.ETLJobsInProgress.Inc()
}

// ETL jobs in progress counter
func (m *Metrics) DecETLJobsInProgress() {
	m.ETLJobsInProgress.Dec()
}

// HTTP requests in flight counter
func (m *Metrics) IncHTTPRequestsInFlight() {
	m.HTTPRequestsInFlight.Inc()
}

// HTTP requests in flight counter
func (m *Metrics) DecHTTPRequestsInFlight() {
	m.HTTPRequestsInFlight.Dec()
}
