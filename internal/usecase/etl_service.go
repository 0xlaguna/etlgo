package usecase

import (
	"context"
	"fmt"
	"sync"
	"time"

	"etlgo/internal/domain"
	"etlgo/pkg/logger"
	"etlgo/pkg/metrics"
)

type ETLService struct {
	adRepo      domain.AdRepository
	crmRepo     domain.CRMRepository
	metricsRepo domain.MetricsRepository
	apiClient   domain.ExternalAPIClient
	logger      *logger.Logger
	metrics     *metrics.Metrics
	workerPool  int
	batchSize   int
}

func NewETLService(
	adRepo domain.AdRepository,
	crmRepo domain.CRMRepository,
	metricsRepo domain.MetricsRepository,
	apiClient domain.ExternalAPIClient,
	logger *logger.Logger,
	metrics *metrics.Metrics,
	workerPool, batchSize int,
) *ETLService {
	return &ETLService{
		adRepo:      adRepo,
		crmRepo:     crmRepo,
		metricsRepo: metricsRepo,
		apiClient:   apiClient,
		logger:      logger,
		metrics:     metrics,
		workerPool:  workerPool,
		batchSize:   batchSize,
	}
}

// Executes the complete ETL pipeline
func (s *ETLService) RunETL(ctx context.Context, since *time.Time) error {
	start := time.Now()
	s.metrics.IncETLJobsInProgress()
	defer s.metrics.DecETLJobsInProgress()

	log := s.logger.WithContext(ctx)
	log.Info("Starting ETL pipeline")

	// Extract data from external APIs
	adsData, crmData, err := s.extractData(ctx)
	if err != nil {
		s.metrics.RecordETLJob("failed", "extract", time.Since(start))
		return fmt.Errorf("failed to extract data: %w", err)
	}

	// Transform data
	processedAds, processedCRM, err := s.transformData(ctx, adsData, crmData, since)
	if err != nil {
		s.metrics.RecordETLJob("failed", "transform", time.Since(start))
		return fmt.Errorf("failed to transform data: %w", err)
	}

	// Load data into repositories
	if err := s.loadData(ctx, processedAds, processedCRM); err != nil {
		s.metrics.RecordETLJob("failed", "load", time.Since(start))
		return fmt.Errorf("failed to load data: %w", err)
	}

	// Calculate and store business metrics
	if err := s.calculateMetrics(ctx, since); err != nil {
		s.metrics.RecordETLJob("failed", "metrics", time.Since(start))
		return fmt.Errorf("failed to calculate metrics: %w", err)
	}

	duration := time.Since(start)
	s.metrics.RecordETLJob("success", "complete", duration)

	log.WithFields(map[string]any{
		"duration":     duration,
		"ads_records":  len(processedAds),
		"crm_records":  len(processedCRM),
		"since_filter": since != nil,
	}).Info("ETL pipeline completed successfully")

	return nil
}

// extractData fetches data from external APIs concurrently
func (s *ETLService) extractData(ctx context.Context) (*domain.AdData, *domain.CRMData, error) {
	log := s.logger.WithContext(ctx)
	log.Info("Extracting data from external APIs")

	var adsData *domain.AdData
	var crmData *domain.CRMData
	var adsErr, crmErr error

	// fetch data concurrently
	var wg sync.WaitGroup
	wg.Add(2)

	// Fetch ads data
	go func() {
		defer wg.Done()
		adsData, adsErr = s.apiClient.FetchAdsData(ctx)
		if adsErr != nil {
			log.WithError(adsErr).Error("Failed to fetch ads data")
		}
	}()

	// Fetch CRM data
	go func() {
		defer wg.Done()
		crmData, crmErr = s.apiClient.FetchCRMData(ctx)
		if crmErr != nil {
			log.WithError(crmErr).Error("Failed to fetch CRM data")
		}
	}()

	wg.Wait()

	if adsErr != nil {
		return nil, nil, fmt.Errorf("ads data extraction failed: %w", adsErr)
	}
	if crmErr != nil {
		return nil, nil, fmt.Errorf("CRM data extraction failed: %w", crmErr)
	}

	log.WithFields(map[string]any{
		"ads_records": len(adsData.External.Ads.Performance),
		"crm_records": len(crmData.External.CRM.Opportunities),
	}).Info("Data extraction completed")

	return adsData, crmData, nil
}

// processes and normalizes the raw data
func (s *ETLService) transformData(ctx context.Context, adsData *domain.AdData, crmData *domain.CRMData, since *time.Time) ([]domain.ProcessedAdData, []domain.ProcessedOpportunity, error) {
	log := s.logger.WithContext(ctx)
	log.Info("Transforming data")

	// Process ads data
	processedAds := s.processAdsData(adsData.External.Ads.Performance, since)

	// Process CRM data
	processedCRM := s.processCRMData(crmData.External.CRM.Opportunities, since)

	// Record processing metrics
	s.metrics.RecordETLRecords("ads", "success", len(processedAds))
	s.metrics.RecordETLRecords("crm", "success", len(processedCRM))

	log.WithFields(map[string]any{
		"processed_ads": len(processedAds),
		"processed_crm": len(processedCRM),
	}).Info("Data transformation completed")

	return processedAds, processedCRM, nil
}

// processes and normalizes ads data
func (s *ETLService) processAdsData(ads []domain.AdPerformance, since *time.Time) []domain.ProcessedAdData {
	var processed []domain.ProcessedAdData

	for _, ad := range ads {
		// Parse date - try multiple formats
		dateFormats := []string{
			"2006-01-02", // YYYY-MM-DD
			"2006/01/02", // YYYY/MM/DD
			"01/02/2006", // MM/DD/YYYY
			"02/01/2006", // DD/MM/YYYY
			time.RFC3339, // 2006-01-02T15:04:05Z07:00
		}

		var date time.Time
		var err error
		for _, format := range dateFormats {
			date, err = time.Parse(format, ad.Date)
			if err == nil {
				break
			}
		}

		if err != nil {
			s.logger.WithError(err).WithField("date", ad.Date).Warn("Failed to parse ad date, skipping")
			s.metrics.RecordETLRecordFailure("ads", "date_parse")
			continue
		}

		// Apply date filter if specified
		if since != nil && date.Before(*since) {
			continue
		}

		// Normalize UTM fields (handle empty values)
		utmCampaign := ad.UTMCampaign
		if utmCampaign == "" {
			utmCampaign = "unknown"
		}

		utmSource := ad.UTMSource
		if utmSource == "" {
			utmSource = "unknown"
		}

		utmMedium := ad.UTMMedium
		if utmMedium == "" {
			utmMedium = "unknown"
		}

		processed = append(processed, domain.ProcessedAdData{
			Date:        date,
			CampaignID:  ad.CampaignID,
			Channel:     ad.Channel,
			Clicks:      ad.Clicks,
			Impressions: ad.Impressions,
			Cost:        ad.Cost,
			UTMCampaign: utmCampaign,
			UTMSource:   utmSource,
			UTMMedium:   utmMedium,
			ProcessedAt: time.Now(),
		})
	}

	return processed
}

// processes and normalizes CRM data
func (s *ETLService) processCRMData(opportunities []domain.Opportunity, since *time.Time) []domain.ProcessedOpportunity {
	var processed []domain.ProcessedOpportunity

	for _, opp := range opportunities {
		// Parse date - try multiple formats
		dateFormats := []string{
			time.RFC3339,          // 2006-01-02T15:04:05Z07:00
			"2006-01-02 15:04:05", // YYYY-MM-DD HH:MM:SS
			"2006-01-02",          // YYYY-MM-DD
			"2006/01/02 15:04:05", // YYYY/MM/DD HH:MM:SS
			"2006/01/02",          // YYYY/MM/DD
		}

		var createdAt time.Time
		var err error
		for _, format := range dateFormats {
			createdAt, err = time.Parse(format, opp.CreatedAt)
			if err == nil {
				break
			}
		}

		if err != nil {
			s.logger.WithError(err).WithField("created_at", opp.CreatedAt).Warn("Failed to parse opportunity date, skipping")
			s.metrics.RecordETLRecordFailure("crm", "date_parse")
			continue
		}

		// Apply date filter if specified
		if since != nil && createdAt.Before(*since) {
			continue
		}

		// Normalize UTM fields (handle empty values)
		utmCampaign := opp.UTMCampaign
		if utmCampaign == "" {
			utmCampaign = "unknown"
		}

		utmSource := opp.UTMSource
		if utmSource == "" {
			utmSource = "unknown"
		}

		utmMedium := opp.UTMMedium
		if utmMedium == "" {
			utmMedium = "unknown"
		}

		processed = append(processed, domain.ProcessedOpportunity{
			OpportunityID: opp.OpportunityID,
			ContactEmail:  opp.ContactEmail,
			Stage:         opp.Stage,
			Amount:        opp.Amount,
			CreatedAt:     createdAt,
			UTMCampaign:   utmCampaign,
			UTMSource:     utmSource,
			UTMMedium:     utmMedium,
			ProcessedAt:   time.Now(),
		})
	}

	return processed
}

// stores the processed data in repositories
func (s *ETLService) loadData(ctx context.Context, ads []domain.ProcessedAdData, opportunities []domain.ProcessedOpportunity) error {
	log := s.logger.WithContext(ctx)
	log.Info("Loading data into repositories")

	// load data concurrently
	var wg sync.WaitGroup
	var adsErr, crmErr error

	wg.Add(2)

	// Load ads data
	go func() {
		defer wg.Done()
		adsErr = s.adRepo.Store(ctx, ads)
		if adsErr != nil {
			log.WithError(adsErr).Error("Failed to store ads data")
		}
	}()

	// Load CRM data
	go func() {
		defer wg.Done()
		crmErr = s.crmRepo.Store(ctx, opportunities)
		if crmErr != nil {
			log.WithError(crmErr).Error("Failed to store CRM data")
		}
	}()

	wg.Wait()

	if adsErr != nil {
		return fmt.Errorf("failed to store ads data: %w", adsErr)
	}
	if crmErr != nil {
		return fmt.Errorf("failed to store CRM data: %w", crmErr)
	}

	log.Info("Data loading completed")
	return nil
}

// calculates and stores business metrics
func (s *ETLService) calculateMetrics(ctx context.Context, since *time.Time) error {
	log := s.logger.WithContext(ctx)
	log.Info("Calculating business metrics")

	// Determine date range for metrics calculation
	from := time.Now().AddDate(0, 0, -365)
	to := time.Now().AddDate(0, 0, 30)

	if since != nil {
		from = *since
	}

	// Get processed data
	ads, err := s.adRepo.GetByDateRange(ctx, from, to)
	if err != nil {
		return fmt.Errorf("failed to get ads data for metrics: %w", err)
	}

	opportunities, err := s.crmRepo.GetByDateRange(ctx, from, to)
	if err != nil {
		return fmt.Errorf("failed to get CRM data for metrics: %w", err)
	}

	// Calculate metrics using worker pool
	metrics := s.calculateMetricsWithWorkerPool(ctx, ads, opportunities)

	// Store metrics
	if err := s.metricsRepo.Store(ctx, metrics); err != nil {
		return fmt.Errorf("failed to store metrics: %w", err)
	}

	log.WithField("metrics_count", len(metrics)).Info("Business metrics calculation completed")
	return nil
}

// calculates metrics using concurrent processing
func (s *ETLService) calculateMetricsWithWorkerPool(ctx context.Context, ads []domain.ProcessedAdData, opportunities []domain.ProcessedOpportunity) []domain.BusinessMetrics {
	// Group data by UTM for correlation
	adsByUTM := make(map[domain.UTMKey][]domain.ProcessedAdData)
	oppsByUTM := make(map[domain.UTMKey][]domain.ProcessedOpportunity)

	// Group ads by UTM
	for _, ad := range ads {
		utm := domain.UTMKey{
			Campaign: ad.UTMCampaign,
			Source:   ad.UTMSource,
			Medium:   ad.UTMMedium,
		}
		adsByUTM[utm] = append(adsByUTM[utm], ad)
	}

	// Group opportunities by UTM
	for _, opp := range opportunities {
		utm := domain.UTMKey{
			Campaign: opp.UTMCampaign,
			Source:   opp.UTMSource,
			Medium:   opp.UTMMedium,
		}
		oppsByUTM[utm] = append(oppsByUTM[utm], opp)
	}

	// Create jobs for worker pool
	jobs := make(chan domain.UTMKey, len(adsByUTM))
	results := make(chan domain.BusinessMetrics, len(adsByUTM))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < s.workerPool; i++ {
		wg.Go(func() {
			for utm := range jobs {
				metric := s.calculateMetricForUTM(adsByUTM[utm], oppsByUTM[utm], utm)
				if metric != nil {
					results <- *metric
				}
			}
		})
	}

	// Send jobs
	go func() {
		defer close(jobs)
		for utm := range adsByUTM {
			jobs <- utm
		}
	}()

	// Collect results
	go func() {
		wg.Wait()
		close(results)
	}()

	var metrics []domain.BusinessMetrics
	for metric := range results {
		metrics = append(metrics, metric)
		s.metrics.RecordBusinessMetric("calculated")
	}

	return metrics
}

// calculates business metrics for a specific UTM combination
func (s *ETLService) calculateMetricForUTM(ads []domain.ProcessedAdData, opportunities []domain.ProcessedOpportunity, utm domain.UTMKey) *domain.BusinessMetrics {
	if len(ads) == 0 {
		return nil
	}

	// Aggregate ads data
	var totalClicks, totalImpressions int
	var totalCost float64
	var latestDate time.Time
	var channel, campaignID string

	for _, ad := range ads {
		totalClicks += ad.Clicks
		totalImpressions += ad.Impressions
		totalCost += ad.Cost
		if ad.Date.After(latestDate) {
			latestDate = ad.Date
			channel = ad.Channel
			campaignID = ad.CampaignID
		}
	}

	// Count opportunities by stage
	var leads, opps, closedWon int
	var revenue float64

	for _, opp := range opportunities {
		switch opp.Stage {
		case domain.StageLead:
			leads++
		case domain.StageOpportunity:
			opps++
		case domain.StageClosedWon:
			closedWon++
			revenue += opp.Amount
		}
	}

	// Calculate metrics
	metric := &domain.BusinessMetrics{
		Date:        latestDate,
		Channel:     channel,
		CampaignID:  campaignID,
		UTMCampaign: utm.Campaign,
		UTMSource:   utm.Source,
		UTMMedium:   utm.Medium,

		Clicks:        totalClicks,
		Impressions:   totalImpressions,
		Cost:          totalCost,
		Leads:         leads,
		Opportunities: opps,
		ClosedWon:     closedWon,
		Revenue:       revenue,

		CalculatedAt: time.Now(),
	}

	// Calculate derived metrics with division by zero protection
	if totalClicks > 0 {
		metric.CPC = totalCost / float64(totalClicks)
	}

	if leads > 0 {
		metric.CPA = totalCost / float64(leads)
	}

	if leads > 0 {
		metric.CVRLeadToOpp = float64(opps) / float64(leads)
	}

	if opps > 0 {
		metric.CVROppToWon = float64(closedWon) / float64(opps)
	}

	if totalCost > 0 {
		metric.ROAS = revenue / totalCost
	}

	return metric
}
