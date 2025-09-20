package infrastructure

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"etlgo/internal/domain"
	"etlgo/pkg/logger"
	"etlgo/pkg/metrics"

	"golang.org/x/time/rate"
)

// implements ExternalAPIClient interface
type HTTPClient struct {
	client      *http.Client
	adsURL      string
	crmURL      string
	sinkURL     string
	sinkSecret  string
	logger      *logger.Logger
	metrics     *metrics.Metrics
	rateLimiter rate.Limiter
}

// creates a new HTTP client
func NewHTTPClient(adsURL, crmURL, sinkURL, sinkSecret string, timeout time.Duration, logger *logger.Logger, metrics *metrics.Metrics) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		adsURL:      adsURL,
		crmURL:      crmURL,
		sinkURL:     sinkURL,
		sinkSecret:  sinkSecret,
		logger:      logger,
		metrics:     metrics,
		rateLimiter: *rate.NewLimiter(rate.Limit(100), 10),
	}
}

// fetches ads data from external API
func (c *HTTPClient) FetchAdsData(ctx context.Context) (*domain.AdData, error) {
	start := time.Now()

	// Apply rate limiting
	if err := c.rateLimiter.Wait(ctx); err != nil {
		c.metrics.RecordExternalAPIFailure("ads", "rate_limit")
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.adsURL, nil)
	if err != nil {
		c.metrics.RecordExternalAPIFailure("ads", "request_creation")
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		c.metrics.RecordExternalAPIFailure("ads", "network_error")
		return nil, fmt.Errorf("failed to fetch ads data: %w", err)
	}
	defer resp.Body.Close()

	duration := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		c.metrics.RecordExternalAPICall("ads", fmt.Sprintf("error_%d", resp.StatusCode), duration)
		return nil, fmt.Errorf("ads API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.metrics.RecordExternalAPIFailure("ads", "read_body")
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var adData domain.AdData
	if err := json.Unmarshal(body, &adData); err != nil {
		c.metrics.RecordExternalAPIFailure("ads", "json_parse")
		return nil, fmt.Errorf("failed to parse ads data: %w", err)
	}

	c.metrics.RecordExternalAPICall("ads", "success", duration)

	c.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"url":      c.adsURL,
		"duration": duration,
		"records":  len(adData.External.Ads.Performance),
	}).Info("Successfully fetched ads data")

	return &adData, nil
}

// fetches CRM data from external API
func (c *HTTPClient) FetchCRMData(ctx context.Context) (*domain.CRMData, error) {
	start := time.Now()

	// Apply rate limiting
	if err := c.rateLimiter.Wait(ctx); err != nil {
		c.metrics.RecordExternalAPIFailure("crm", "rate_limit")
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.crmURL, nil)
	if err != nil {
		c.metrics.RecordExternalAPIFailure("crm", "request_creation")
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		c.metrics.RecordExternalAPIFailure("crm", "network_error")
		return nil, fmt.Errorf("failed to fetch CRM data: %w", err)
	}
	defer resp.Body.Close()

	duration := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		c.metrics.RecordExternalAPICall("crm", fmt.Sprintf("error_%d", resp.StatusCode), duration)
		return nil, fmt.Errorf("CRM API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.metrics.RecordExternalAPIFailure("crm", "read_body")
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var crmData domain.CRMData
	if err := json.Unmarshal(body, &crmData); err != nil {
		c.metrics.RecordExternalAPIFailure("crm", "json_parse")
		return nil, fmt.Errorf("failed to parse CRM data: %w", err)
	}

	c.metrics.RecordExternalAPICall("crm", "success", duration)

	c.logger.WithContext(ctx).WithFields(map[string]any{
		"url":      c.crmURL,
		"duration": duration,
		"records":  len(crmData.External.CRM.Opportunities),
	}).Info("Successfully fetched CRM data")

	return &crmData, nil
}

// implements ExportClient interface
func (c *HTTPClient) Export(ctx context.Context, data []domain.ExportData, date time.Time) error {
	if c.sinkURL == "" {
		return fmt.Errorf("sink URL not configured")
	}

	start := time.Now()

	// Apply rate limiting
	if err := c.rateLimiter.Wait(ctx); err != nil {
		c.metrics.RecordExternalAPIFailure("sink", "rate_limit")
		return fmt.Errorf("rate limit exceeded: %w", err)
	}

	payload, err := json.Marshal(data)
	if err != nil {
		c.metrics.RecordExternalAPIFailure("sink", "json_marshal")
		return fmt.Errorf("failed to marshal export data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.sinkURL, bytes.NewReader(payload))
	if err != nil {
		c.metrics.RecordExternalAPIFailure("sink", "request_creation")
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add HMAC signature if secret is provided
	if c.sinkSecret != "" {
		signature := c.generateHMACSignature(payload)
		req.Header.Set("X-Signature", signature)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		c.metrics.RecordExternalAPIFailure("sink", "network_error")
		return fmt.Errorf("failed to export data: %w", err)
	}
	defer resp.Body.Close()

	duration := time.Since(start)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.metrics.RecordExternalAPICall("sink", fmt.Sprintf("error_%d", resp.StatusCode), duration)
		return fmt.Errorf("sink API returned status %d", resp.StatusCode)
	}

	c.metrics.RecordExternalAPICall("sink", "success", duration)

	c.logger.WithContext(ctx).WithFields(map[string]any{
		"url":      c.sinkURL,
		"duration": duration,
		"records":  len(data),
		"date":     date.Format("2006-01-02"),
	}).Info("Successfully exported data")

	return nil
}

// generates HMAC-SHA256 signature for the payload
func (c *HTTPClient) generateHMACSignature(payload []byte) string {
	h := hmac.New(sha256.New, []byte(c.sinkSecret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}
