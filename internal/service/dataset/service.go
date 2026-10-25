package dataset

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"hw2-mongo/internal/storage/mongo/fixtures"
	ordersStorage "hw2-mongo/internal/storage/mongo/orders"
	productsStorage "hw2-mongo/internal/storage/mongo/products"
)

type Service struct {
	logger   zerolog.Logger
	client   *mongo.Client
	dbName   string
	products *productsStorage.Storage
	orders   *ordersStorage.Storage
}

func New(
	logger zerolog.Logger,
	client *mongo.Client,
	dbName string,
	products *productsStorage.Storage,
	orders *ordersStorage.Storage,
) *Service {
	return &Service{
		logger:   logger.With().Str("component", "dataset.Service").Logger(),
		client:   client,
		dbName:   dbName,
		products: products,
		orders:   orders,
	}
}

func (s *Service) Reset(ctx context.Context, withOrders bool) error {
	db := s.client.Database(s.dbName)

	if err := db.Drop(ctx); err != nil {
		return fmt.Errorf("drop %s: %w", s.dbName, err)
	}

	if err := db.CreateCollection(ctx, "products"); err != nil {
		return fmt.Errorf("createCollection products: %w", err)
	}

	if err := db.CreateCollection(ctx, "orders"); err != nil {
		return fmt.Errorf("createCollection orders: %w", err)
	}

	if err := s.products.InsertMany(ctx, fixtures.Products()); err != nil {
		return fmt.Errorf("load products: %w", err)
	}

	if withOrders {
		if err := s.orders.InsertMany(ctx, fixtures.Orders()); err != nil {
			return fmt.Errorf("load orders: %w", err)
		}
	}

	s.logger.Info().Bool("withOrders", withOrders).Msg("dataset reset")

	return nil
}

func (s *Service) LoadLargeOrders(ctx context.Context) error {
	if err := s.orders.InsertMany(ctx, fixtures.LargeOrders()); err != nil {
		return fmt.Errorf("load large orders: %w", err)
	}

	return nil
}
