package catalog

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	"go.mongodb.org/mongo-driver/v2/bson"

	"hw2-mongo/internal/domain"
)

type Storage interface {
	ApplyValidator(ctx context.Context, validator any) error
	CollectionOptions(ctx context.Context) (bson.M, error)
	SearchByVariant(ctx context.Context, category, size string, qty int32) ([]domain.Product, error)
}

type Service struct {
	logger  zerolog.Logger
	storage Storage
}

func New(logger zerolog.Logger, storage Storage) *Service {
	return &Service{
		logger:  logger.With().Str("component", "catalog.Service").Logger(),
		storage: storage,
	}
}

func (s *Service) InstallValidator(ctx context.Context) error {
	if err := s.storage.ApplyValidator(ctx, validatorSchema()); err != nil {
		return fmt.Errorf("apply validator: %w", err)
	}

	return nil
}

func (s *Service) ValidatorOptions(ctx context.Context) (bson.M, error) {
	options, err := s.storage.CollectionOptions(ctx)
	if err != nil {
		return nil, fmt.Errorf("collection options: %w", err)
	}

	return options, nil
}

func (s *Service) SearchProducts(ctx context.Context, in domain.SearchInput) ([]domain.Product, error) {
	found, err := s.storage.SearchByVariant(ctx, in.Category, in.Size, in.Qty)
	if err != nil {
		return nil, fmt.Errorf("search by variant: %w", err)
	}

	return found, nil
}

func validatorSchema() bson.D {
	return bson.D{
		{Key: "$jsonSchema", Value: bson.D{
			{Key: "bsonType", Value: "object"},
			{Key: "required", Value: bson.A{"_id", "name", "category", "price", "variants"}},
			{Key: "properties", Value: bson.D{
				{Key: "_id", Value: bson.D{
					{Key: "bsonType", Value: "string"},
				}},
				{Key: "name", Value: bson.D{
					{Key: "bsonType", Value: "string"},
				}},
				{Key: "category", Value: bson.D{
					{Key: "bsonType", Value: "string"},
				}},
				{Key: "price", Value: bson.D{
					{Key: "bsonType", Value: "decimal"},
					{Key: "minimum", Value: 0},
				}},
				{Key: "variants", Value: bson.D{
					{Key: "bsonType", Value: "array"},
					{Key: "items", Value: bson.D{
						{Key: "bsonType", Value: "object"},
						{Key: "required", Value: bson.A{"size", "stock"}},
						{Key: "properties", Value: bson.D{
							{Key: "size", Value: bson.D{
								{Key: "bsonType", Value: "string"},
							}},
							{Key: "stock", Value: bson.D{
								{Key: "bsonType", Value: bson.A{"int", "long"}},
								{Key: "minimum", Value: 0},
							}},
						}},
					}},
				}},
			}},
		}},
	}
}
