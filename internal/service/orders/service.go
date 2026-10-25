package orders

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	"go.mongodb.org/mongo-driver/v2/bson"

	"hw2-mongo/internal/domain"
)

type Storage interface {
	FindHistory(ctx context.Context, in domain.HistoryInput) ([]domain.Order, error)
	ExplainHistory(ctx context.Context, in domain.HistoryInput) (bson.M, error)
	CreateHistoryIndex(ctx context.Context) error
	IndexNames(ctx context.Context) ([]string, error)
	Count(ctx context.Context) (int64, error)
	InsertMany(ctx context.Context, orders []domain.Order) error
}

type Service struct {
	logger  zerolog.Logger
	storage Storage
}

func New(logger zerolog.Logger, storage Storage) *Service {
	return &Service{
		logger:  logger.With().Str("component", "orders.Service").Logger(),
		storage: storage,
	}
}

func (s *Service) History(ctx context.Context, in domain.HistoryInput) ([]domain.Order, error) {
	history, err := s.storage.FindHistory(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("find history: %w", err)
	}

	return history, nil
}

func (s *Service) ExplainHistory(ctx context.Context, in domain.HistoryInput) (bson.M, error) {
	plan, err := s.storage.ExplainHistory(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("explain history: %w", err)
	}

	return plan, nil
}

func (s *Service) CreateHistoryIndex(ctx context.Context) error {
	if err := s.storage.CreateHistoryIndex(ctx); err != nil {
		return fmt.Errorf("create history index: %w", err)
	}

	return nil
}

func (s *Service) IndexNames(ctx context.Context) ([]string, error) {
	names, err := s.storage.IndexNames(ctx)
	if err != nil {
		return nil, fmt.Errorf("index names: %w", err)
	}

	return names, nil
}

func (s *Service) Count(ctx context.Context) (int64, error) {
	count, err := s.storage.Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("count: %w", err)
	}

	return count, nil
}

func (s *Service) LoadOrders(ctx context.Context, orders []domain.Order) error {
	if err := s.storage.InsertMany(ctx, orders); err != nil {
		return fmt.Errorf("load orders: %w", err)
	}

	return nil
}
