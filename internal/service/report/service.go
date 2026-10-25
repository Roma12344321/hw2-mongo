package report

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"hw2-mongo/internal/domain"
)

type Storage interface {
	AggregateReport(ctx context.Context, from, to time.Time) ([]domain.ReportRow, error)
}

type Service struct {
	logger  zerolog.Logger
	storage Storage
}

func New(logger zerolog.Logger, storage Storage) *Service {
	return &Service{
		logger:  logger.With().Str("component", "report.Service").Logger(),
		storage: storage,
	}
}

func (s *Service) Sales(ctx context.Context, from, to time.Time) ([]domain.ReportRow, error) {
	rows, err := s.storage.AggregateReport(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("aggregate report: %w", err)
	}

	return rows, nil
}
