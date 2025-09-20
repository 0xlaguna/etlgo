package infrastructure

import (
	"context"
	"sync"
	"time"

	"etlgo/internal/domain"
	"etlgo/pkg/logger"
)

// implements domain.CRMRepository interface
type CRMRepository struct {
	data   map[string][]domain.ProcessedOpportunity
	mutex  sync.RWMutex
	logger *logger.Logger
}

// creates a new CRM repository
func NewCRMRepository(logger *logger.Logger) *CRMRepository {
	return &CRMRepository{
		data:   make(map[string][]domain.ProcessedOpportunity),
		logger: logger,
	}
}

func (r *CRMRepository) Store(ctx context.Context, opportunities []domain.ProcessedOpportunity) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for _, opp := range opportunities {
		dateKey := opp.CreatedAt.Format("2006-01-02")
		r.data[dateKey] = append(r.data[dateKey], opp)
	}

	r.logger.WithContext(ctx).WithField("count", len(opportunities)).Info("Stored CRM data in memory")
	return nil
}

func (r *CRMRepository) GetByDateRange(ctx context.Context, from, to time.Time) ([]domain.ProcessedOpportunity, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var result []domain.ProcessedOpportunity

	for date := from; !date.After(to); date = date.AddDate(0, 0, 1) {
		dateKey := date.Format("2006-01-02")
		if opportunities, exists := r.data[dateKey]; exists {
			result = append(result, opportunities...)
		}
	}

	return result, nil
}

func (r *CRMRepository) GetByUTM(ctx context.Context, utm domain.UTMKey, from, to time.Time) ([]domain.ProcessedOpportunity, error) {
	opportunities, err := r.GetByDateRange(ctx, from, to)
	if err != nil {
		return nil, err
	}

	var result []domain.ProcessedOpportunity
	for _, opp := range opportunities {
		if opp.UTMCampaign == utm.Campaign && opp.UTMSource == utm.Source && opp.UTMMedium == utm.Medium {
			result = append(result, opp)
		}
	}

	return result, nil
}

func (r *CRMRepository) GetByStage(ctx context.Context, stage domain.OpportunityStage, from, to time.Time) ([]domain.ProcessedOpportunity, error) {
	opportunities, err := r.GetByDateRange(ctx, from, to)
	if err != nil {
		return nil, err
	}

	var result []domain.ProcessedOpportunity
	for _, opp := range opportunities {
		if opp.Stage == stage {
			result = append(result, opp)
		}
	}

	return result, nil
}
