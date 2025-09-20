package infrastructure

import (
	"context"
	"sync"

	"etlgo/internal/domain"
	"etlgo/pkg/logger"
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
