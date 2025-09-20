package infrastructure

import (
	"context"
	"sort"
	"sync"
	"time"

	"etlgo/internal/domain"
	"etlgo/pkg/logger"
)

// implements domain.MetricsRepository interface
type MetricsRepository struct {
	data   map[string][]domain.BusinessMetrics
	mutex  sync.RWMutex
	logger *logger.Logger
}

// creates a new metrics repository
func NewMetricsRepository(logger *logger.Logger) *MetricsRepository {
	return &MetricsRepository{
		data:   make(map[string][]domain.BusinessMetrics),
		logger: logger,
	}
}

func (r *MetricsRepository) Store(ctx context.Context, metrics []domain.BusinessMetrics) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	log := r.logger.WithContext(ctx)

	for _, metric := range metrics {
		dateKey := metric.Date.Format("2006-01-02")
		r.data[dateKey] = append(r.data[dateKey], metric)

		log.WithFields(map[string]any{
			"date":         dateKey,
			"utm_campaign": metric.UTMCampaign,
			"utm_source":   metric.UTMSource,
			"utm_medium":   metric.UTMMedium,
			"channel":      metric.Channel,
		}).Debug("Stored individual metric")
	}

	log.WithField("count", len(metrics)).Info("Stored business metrics in memory")
	return nil
}

func (r *MetricsRepository) GetByFilter(ctx context.Context, filter domain.MetricsFilter) (*domain.MetricsResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	log := r.logger.WithContext(ctx)
	log.WithFields(map[string]any{
		"filter":             filter,
		"total_stored_dates": len(r.data),
	}).Info("GetByFilter called")

	var allMetrics []domain.BusinessMetrics

	// Get date range
	from := time.Now().AddDate(0, 0, -365)
	to := time.Now()

	if filter.From != nil {
		from = *filter.From
	}
	if filter.To != nil {
		to = *filter.To
	}

	log.WithFields(map[string]any{
		"from": from.Format("2006-01-02"),
		"to":   to.Format("2006-01-02"),
	}).Info("Date range for metrics collection")

	// Collect metrics from date range
	for date := from; !date.After(to); date = date.AddDate(0, 0, 1) {
		dateKey := date.Format("2006-01-02")
		if metrics, exists := r.data[dateKey]; exists {
			log.WithFields(map[string]any{
				"date":  dateKey,
				"count": len(metrics),
			}).Info("Found metrics for date")
			allMetrics = append(allMetrics, metrics...)
		}
	}

	log.WithField("total_collected", len(allMetrics)).Info("Collected metrics from date range")

	// Apply filters
	var filteredMetrics []domain.BusinessMetrics
	for _, metric := range allMetrics {
		if r.matchesFilter(metric, filter) {
			filteredMetrics = append(filteredMetrics, metric)
		}
	}

	log.WithFields(map[string]any{
		"before_filter":       len(allMetrics),
		"after_filter":        len(filteredMetrics),
		"utm_campaign_filter": filter.UTMCampaign,
	}).Info("Applied filters to metrics")

	// Sort by date
	sort.Slice(filteredMetrics, func(i, j int) bool {
		return filteredMetrics[i].Date.Before(filteredMetrics[j].Date)
	})

	// Apply pagination
	limit := 100 // Default limit
	offset := 0  // Default offset

	if filter.Limit > 0 {
		limit = filter.Limit
	}
	if filter.Offset > 0 {
		offset = filter.Offset
	}

	total := len(filteredMetrics)
	start := offset
	end := offset + limit

	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	var paginatedMetrics []domain.BusinessMetrics
	if start < end {
		paginatedMetrics = filteredMetrics[start:end]
	}

	hasMore := end < total

	log.WithFields(map[string]any{
		"final_count": len(paginatedMetrics),
		"total":       total,
		"limit":       limit,
		"offset":      offset,
		"has_more":    hasMore,
	}).Info("Returning metrics response")

	return &domain.MetricsResponse{
		Data:    paginatedMetrics,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: hasMore,
	}, nil
}

func (r *MetricsRepository) GetByDate(ctx context.Context, date time.Time) ([]domain.BusinessMetrics, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	dateKey := date.Format("2006-01-02")
	if metrics, exists := r.data[dateKey]; exists {
		return metrics, nil
	}

	return []domain.BusinessMetrics{}, nil
}

// matchesFilter checks if a metric matches the given filter
func (r *MetricsRepository) matchesFilter(metric domain.BusinessMetrics, filter domain.MetricsFilter) bool {
	if filter.Channel != "" && metric.Channel != filter.Channel {
		return false
	}
	if filter.CampaignID != "" && metric.CampaignID != filter.CampaignID {
		return false
	}
	if filter.UTMCampaign != "" && metric.UTMCampaign != filter.UTMCampaign {
		return false
	}
	if filter.UTMSource != "" && metric.UTMSource != filter.UTMSource {
		return false
	}
	if filter.UTMMedium != "" && metric.UTMMedium != filter.UTMMedium {
		return false
	}

	return true
}
