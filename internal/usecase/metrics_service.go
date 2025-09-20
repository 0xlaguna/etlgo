package usecase

import (
	"context"
	"fmt"
	"time"

	"etlgo/internal/domain"
	"etlgo/pkg/logger"
	"etlgo/pkg/metrics"
)

// MetricsService handles business metrics operations
type MetricsService struct {
	metricsRepo  domain.MetricsRepository
	exportClient domain.ExportClient
	logger       *logger.Logger
	metrics      *metrics.Metrics
}

// NewMetricsService creates a new metrics service
func NewMetricsService(
	metricsRepo domain.MetricsRepository,
	exportClient domain.ExportClient,
	logger *logger.Logger,
	metrics *metrics.Metrics,
) *MetricsService {
	return &MetricsService{
		metricsRepo:  metricsRepo,
		exportClient: exportClient,
		logger:       logger,
		metrics:      metrics,
	}
}

// GetMetricsByChannel retrieves metrics filtered by channel
func (s *MetricsService) GetMetricsByChannel(ctx context.Context, channel string, from, to time.Time, limit, offset int) (*domain.MetricsResponse, error) {
	log := s.logger.WithContext(ctx)
	log.WithFields(map[string]interface{}{
		"channel": channel,
		"from":    from.Format("2006-01-02"),
		"to":      to.Format("2006-01-02"),
		"limit":   limit,
		"offset":  offset,
	}).Info("Getting metrics by channel")

	filter := domain.MetricsFilter{
		From:    &from,
		To:      &to,
		Channel: channel,
		Limit:   limit,
		Offset:  offset,
	}

	response, err := s.metricsRepo.GetByFilter(ctx, filter)
	if err != nil {
		log.WithError(err).Error("Failed to get metrics by channel")
		return nil, fmt.Errorf("failed to get metrics by channel: %w", err)
	}

	s.metrics.RecordBusinessMetric("channel_query")

	log.WithField("count", len(response.Data)).Info("Retrieved metrics by channel")
	return response, nil
}

// GetMetricsByFunnel retrieves metrics filtered by UTM campaign (funnel analysis)
func (s *MetricsService) GetMetricsByFunnel(ctx context.Context, utmCampaign string, from, to time.Time, limit, offset int) (*domain.MetricsResponse, error) {
	log := s.logger.WithContext(ctx)
	log.WithFields(map[string]interface{}{
		"utm_campaign": utmCampaign,
		"from":         from.Format("2006-01-02"),
		"to":           to.Format("2006-01-02"),
		"limit":        limit,
		"offset":       offset,
	}).Info("Getting metrics by funnel")

	filter := domain.MetricsFilter{
		From:        &from,
		To:          &to,
		UTMCampaign: utmCampaign,
		Limit:       limit,
		Offset:      offset,
	}

	response, err := s.metricsRepo.GetByFilter(ctx, filter)
	if err != nil {
		log.WithError(err).Error("Failed to get metrics by funnel")
		return nil, fmt.Errorf("failed to get metrics by funnel: %w", err)
	}

	s.metrics.RecordBusinessMetric("funnel_query")

	log.WithField("count", len(response.Data)).Info("Retrieved metrics by funnel")
	return response, nil
}

// GetMetricsByFilter retrieves metrics with custom filters
func (s *MetricsService) GetMetricsByFilter(ctx context.Context, filter domain.MetricsFilter) (*domain.MetricsResponse, error) {
	log := s.logger.WithContext(ctx)
	log.WithFields(map[string]interface{}{
		"from":         filter.From,
		"to":           filter.To,
		"channel":      filter.Channel,
		"campaign_id":  filter.CampaignID,
		"utm_campaign": filter.UTMCampaign,
		"utm_source":   filter.UTMSource,
		"utm_medium":   filter.UTMMedium,
		"limit":        filter.Limit,
		"offset":       filter.Offset,
	}).Info("Getting metrics by filter")

	response, err := s.metricsRepo.GetByFilter(ctx, filter)
	if err != nil {
		log.WithError(err).Error("Failed to get metrics by filter")
		return nil, fmt.Errorf("failed to get metrics by filter: %w", err)
	}

	s.metrics.RecordBusinessMetric("filter_query")

	log.WithField("count", len(response.Data)).Info("Retrieved metrics by filter")
	return response, nil
}

// ExportMetrics exports metrics for a specific date
func (s *MetricsService) ExportMetrics(ctx context.Context, date time.Time) error {
	log := s.logger.WithContext(ctx)
	log.WithField("date", date.Format("2006-01-02")).Info("Starting metrics export")

	// Get metrics for the specified date
	metrics, err := s.metricsRepo.GetByDate(ctx, date)
	if err != nil {
		log.WithError(err).Error("Failed to get metrics for export")
		return fmt.Errorf("failed to get metrics for export: %w", err)
	}

	if len(metrics) == 0 {
		log.Warn("No metrics found for export date")
		return fmt.Errorf("no metrics found for date %s", date.Format("2006-01-02"))
	}

	// Convert to export format
	exportData := make([]domain.ExportData, len(metrics))
	for i, metric := range metrics {
		exportData[i] = domain.ExportData{
			Date:          metric.Date.Format("2006-01-02"),
			Channel:       metric.Channel,
			CampaignID:    metric.CampaignID,
			Clicks:        metric.Clicks,
			Impressions:   metric.Impressions,
			Cost:          metric.Cost,
			Leads:         metric.Leads,
			Opportunities: metric.Opportunities,
			ClosedWon:     metric.ClosedWon,
			Revenue:       metric.Revenue,
			CPC:           metric.CPC,
			CPA:           metric.CPA,
			CVRLeadToOpp:  metric.CVRLeadToOpp,
			CVROppToWon:   metric.CVROppToWon,
			ROAS:          metric.ROAS,
		}
	}

	// Export data
	if err := s.exportClient.Export(ctx, exportData, date); err != nil {
		log.WithError(err).Error("Failed to export metrics")
		return fmt.Errorf("failed to export metrics: %w", err)
	}

	s.metrics.RecordBusinessMetric("export")

	log.WithField("records", len(exportData)).Info("Metrics export completed successfully")
	return nil
}

// GetMetricsSummary returns a summary of available metrics
func (s *MetricsService) GetMetricsSummary(ctx context.Context) (map[string]interface{}, error) {
	log := s.logger.WithContext(ctx)
	log.Info("Getting metrics summary")

	// Get metrics for the last 30 days
	from := time.Now().AddDate(0, 0, -60)
	to := time.Now()

	filter := domain.MetricsFilter{
		From: &from,
		To:   &to,
	}

	response, err := s.metricsRepo.GetByFilter(ctx, filter)
	if err != nil {
		log.WithError(err).Error("Failed to get metrics summary")
		return nil, fmt.Errorf("failed to get metrics summary: %w", err)
	}

	// Calculate summary statistics
	var totalClicks, totalImpressions, totalLeads, totalOpportunities, totalClosedWon int
	var totalCost, totalRevenue float64
	channels := make(map[string]bool)
	campaigns := make(map[string]bool)

	for _, metric := range response.Data {
		totalClicks += metric.Clicks
		totalImpressions += metric.Impressions
		totalCost += metric.Cost
		totalLeads += metric.Leads
		totalOpportunities += metric.Opportunities
		totalClosedWon += metric.ClosedWon
		totalRevenue += metric.Revenue

		channels[metric.Channel] = true
		campaigns[metric.CampaignID] = true
	}

	// Calculate aggregate metrics
	var avgCPC, avgCPA, avgCVRLeadToOpp, avgCVROppToWon, avgROAS float64

	if totalClicks > 0 {
		avgCPC = totalCost / float64(totalClicks)
	}

	if totalLeads > 0 {
		avgCPA = totalCost / float64(totalLeads)
	}

	if totalLeads > 0 {
		avgCVRLeadToOpp = float64(totalOpportunities) / float64(totalLeads)
	}

	if totalOpportunities > 0 {
		avgCVROppToWon = float64(totalClosedWon) / float64(totalOpportunities)
	}

	if totalCost > 0 {
		avgROAS = totalRevenue / totalCost
	}

	summary := map[string]interface{}{
		"period": map[string]interface{}{
			"from": from.Format("2006-01-02"),
			"to":   to.Format("2006-01-02"),
		},
		"totals": map[string]interface{}{
			"clicks":        totalClicks,
			"impressions":   totalImpressions,
			"cost":          totalCost,
			"leads":         totalLeads,
			"opportunities": totalOpportunities,
			"closed_won":    totalClosedWon,
			"revenue":       totalRevenue,
		},
		"averages": map[string]interface{}{
			"cpc":             avgCPC,
			"cpa":             avgCPA,
			"cvr_lead_to_opp": avgCVRLeadToOpp,
			"cvr_opp_to_won":  avgCVROppToWon,
			"roas":            avgROAS,
		},
		"counts": map[string]interface{}{
			"unique_channels":  len(channels),
			"unique_campaigns": len(campaigns),
			"metric_records":   len(response.Data),
		},
	}

	s.metrics.RecordBusinessMetric("summary")

	log.WithField("records", len(response.Data)).Info("Metrics summary generated")
	return summary, nil
}
