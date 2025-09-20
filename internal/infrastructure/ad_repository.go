package infrastructure

import (
	"context"
	"etlgo/internal/domain"
	"etlgo/pkg/logger"
	"sync"
	"time"
)

type AdRepository struct {
	data   map[string][]domain.ProcessedAdData
	mutex  sync.RWMutex
	logger *logger.Logger
}

func NewAdRepository(logger *logger.Logger) *AdRepository {
	return &AdRepository{
		data:   make(map[string][]domain.ProcessedAdData),
		logger: logger,
	}
}

func (r *AdRepository) Store(ctx context.Context, ads []domain.ProcessedAdData) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for _, ad := range ads {
		dateKey := ad.Date.Format("2006-01-02")
		r.data[dateKey] = append(r.data[dateKey], ad)
	}

	r.logger.WithContext(ctx).WithField("count", len(ads)).Info("Stored ads data in memory")
	return nil
}

func (r *AdRepository) GetByDateRange(ctx context.Context, from, to time.Time) ([]domain.ProcessedAdData, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var result []domain.ProcessedAdData

	for date := from; !date.After(to); date = date.AddDate(0, 0, 1) {
		dateKey := date.Format("2006-01-02")
		if ads, exists := r.data[dateKey]; exists {
			result = append(result, ads...)
		}
	}

	return result, nil
}

func (r *AdRepository) GetByUTM(ctx context.Context, utm domain.UTMKey, from, to time.Time) ([]domain.ProcessedAdData, error) {
	ads, err := r.GetByDateRange(ctx, from, to)
	if err != nil {
		return nil, err
	}

	var result []domain.ProcessedAdData
	for _, ad := range ads {
		if ad.UTMCampaign == utm.Campaign && ad.UTMSource == utm.Source && ad.UTMMedium == utm.Medium {
			result = append(result, ad)
		}
	}

	return result, nil
}

func (r *AdRepository) GetByCampaign(ctx context.Context, campaignID string, from, to time.Time) ([]domain.ProcessedAdData, error) {
	ads, err := r.GetByDateRange(ctx, from, to)
	if err != nil {
		return nil, err
	}

	var result []domain.ProcessedAdData
	for _, ad := range ads {
		if ad.CampaignID == campaignID {
			result = append(result, ad)
		}
	}

	return result, nil
}

func (r *AdRepository) GetByChannel(ctx context.Context, channel string, from, to time.Time) ([]domain.ProcessedAdData, error) {
	ads, err := r.GetByDateRange(ctx, from, to)
	if err != nil {
		return nil, err
	}

	var result []domain.ProcessedAdData
	for _, ad := range ads {
		if ad.Channel == channel {
			result = append(result, ad)
		}
	}

	return result, nil
}
