package domain

import (
	"context"
	"time"
)

// interface for ad data operations
type AdRepository interface {
	Store(ctx context.Context, ads []ProcessedAdData) error
	GetByDateRange(ctx context.Context, from, to time.Time) ([]ProcessedAdData, error)
	GetByUTM(ctx context.Context, utm UTMKey, from, to time.Time) ([]ProcessedAdData, error)
	GetByCampaign(ctx context.Context, campaignID string, from, to time.Time) ([]ProcessedAdData, error)
	GetByChannel(ctx context.Context, channel string, from, to time.Time) ([]ProcessedAdData, error)
}

// the interface for CRM data operations
type CRMRepository interface {
	Store(ctx context.Context, opportunities []ProcessedOpportunity) error
	GetByDateRange(ctx context.Context, from, to time.Time) ([]ProcessedOpportunity, error)
	GetByUTM(ctx context.Context, utm UTMKey, from, to time.Time) ([]ProcessedOpportunity, error)
	GetByStage(ctx context.Context, stage OpportunityStage, from, to time.Time) ([]ProcessedOpportunity, error)
}

// interface for metrics operations
type MetricsRepository interface {
	Store(ctx context.Context, metrics []BusinessMetrics) error
	GetByFilter(ctx context.Context, filter MetricsFilter) (*MetricsResponse, error)
	GetByDate(ctx context.Context, date time.Time) ([]BusinessMetrics, error)
}

// interface for external API calls
type ExternalAPIClient interface {
	FetchAdsData(ctx context.Context) (*AdData, error)
	FetchCRMData(ctx context.Context) (*CRMData, error)
}

// interface for data export
type ExportClient interface {
	Export(ctx context.Context, data []ExportData, date time.Time) error
}
