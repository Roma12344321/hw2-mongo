package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"

	"hw2-mongo/internal/domain"
)

type ConnManager interface {
	GetDB() *mongo.Database
}

func majorityJournal() *writeconcern.WriteConcern {
	journal := true

	return &writeconcern.WriteConcern{W: "majority", Journal: &journal}
}

type Storage struct {
	connManager  ConnManager
	coll         *mongo.Collection
	collMajority *mongo.Collection
}

func New(connManager ConnManager) *Storage {
	coll := connManager.GetDB().Collection("products")

	return &Storage{
		connManager:  connManager,
		coll:         coll,
		collMajority: coll.Clone(options.Collection().SetWriteConcern(majorityJournal())),
	}
}

func (s *Storage) InsertMany(ctx context.Context, products []domain.Product) error {
	if len(products) == 0 {
		return nil
	}

	if _, err := s.collMajority.InsertMany(ctx, products); err != nil {
		return fmt.Errorf("products.InsertMany: %w", err)
	}

	return nil
}

func (s *Storage) InsertRaw(ctx context.Context, doc bson.D) error {
	if _, err := s.coll.InsertOne(ctx, doc); err != nil {
		return err
	}

	return nil
}

func (s *Storage) DeleteByID(ctx context.Context, id string) error {
	if _, err := s.coll.DeleteOne(ctx, bson.M{"_id": id}); err != nil {
		return fmt.Errorf("products.DeleteOne %s: %w", id, err)
	}

	return nil
}

func (s *Storage) SearchByVariant(
	ctx context.Context,
	category, size string,
	qty int32,
) ([]domain.Product, error) {
	filter := bson.D{
		{Key: "category", Value: category},
		{Key: "variants", Value: bson.D{{Key: "$elemMatch", Value: bson.D{
			{Key: "size", Value: size},
			{Key: "stock", Value: bson.D{{Key: "$gte", Value: qty}}},
		}}}},
	}

	cursor, err := s.coll.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("products.Find: %w", err)
	}

	found := make([]domain.Product, 0)
	if err := cursor.All(ctx, &found); err != nil {
		return nil, fmt.Errorf("cursor.All: %w", err)
	}

	return found, nil
}

func (s *Storage) All(ctx context.Context) ([]domain.Product, error) {
	cursor, err := s.coll.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("products.Find all: %w", err)
	}

	products := make([]domain.Product, 0)
	if err := cursor.All(ctx, &products); err != nil {
		return nil, fmt.Errorf("cursor.All: %w", err)
	}

	return products, nil
}

func (s *Storage) FindByIDs(ctx context.Context, ids []string) ([]domain.Product, error) {
	filter := bson.M{"_id": bson.M{"$in": ids}}
	cursor, err := s.coll.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("products.Find by ids: %w", err)
	}

	found := make([]domain.Product, 0)
	if err := cursor.All(ctx, &found); err != nil {
		return nil, fmt.Errorf("cursor.All: %w", err)
	}

	return found, nil
}

func (s *Storage) FindByID(ctx context.Context, sku string) (domain.Product, error) {
	var product domain.Product
	if err := s.coll.FindOne(ctx, bson.M{"_id": sku}).Decode(&product); err != nil {
		return domain.Product{}, fmt.Errorf("products.FindOne %s: %w", sku, err)
	}

	return product, nil
}

func (s *Storage) SetVariantStock(ctx context.Context, sku, size string, stock int32) error {
	filter := bson.M{"_id": sku, "variants.size": size}
	update := bson.M{"$set": bson.M{"variants.$.stock": stock}}

	if _, err := s.coll.UpdateOne(ctx, filter, update); err != nil {
		return fmt.Errorf("products.UpdateOne %s/%s: %w", sku, size, err)
	}

	return nil
}

func (s *Storage) SetPrice(ctx context.Context, sku string, price bson.Decimal128) error {
	filter := bson.M{"_id": sku}
	update := bson.M{"$set": bson.M{"price": price}}

	if _, err := s.coll.UpdateOne(ctx, filter, update); err != nil {
		return fmt.Errorf("products.UpdateOne price %s: %w", sku, err)
	}

	return nil
}

func (s *Storage) ApplyValidator(ctx context.Context, validator any) error {
	command := bson.D{
		{Key: "collMod", Value: "products"},
		{Key: "validator", Value: validator},
		{Key: "validationLevel", Value: "strict"},
		{Key: "validationAction", Value: "error"},
	}

	if err := s.coll.Database().RunCommand(ctx, command).Err(); err != nil {
		return fmt.Errorf("collMod products: %w", err)
	}

	return nil
}

func (s *Storage) CollectionOptions(ctx context.Context) (bson.M, error) {
	cursor, err := s.coll.Database().ListCollections(ctx, bson.M{"name": "products"})
	if err != nil {
		return nil, fmt.Errorf("listCollections: %w", err)
	}

	var infos []struct {
		Name    string `bson:"name"`
		Options bson.M `bson:"options"`
	}
	if err := cursor.All(ctx, &infos); err != nil {
		return nil, fmt.Errorf("cursor.All: %w", err)
	}

	if len(infos) == 0 {
		return nil, fmt.Errorf("collection products not found")
	}

	return infos[0].Options, nil
}

func (s *Storage) ReserveStock(
	ctx context.Context,
	sku, size string,
	qty int32,
) (matched, modified int64, err error) {
	filter := bson.D{
		{Key: "_id", Value: sku},
		{Key: "variants", Value: bson.D{{Key: "$elemMatch", Value: bson.D{
			{Key: "size", Value: size},
			{Key: "stock", Value: bson.D{{Key: "$gte", Value: qty}}},
		}}}},
	}
	update := bson.D{{Key: "$inc", Value: bson.D{{Key: "variants.$.stock", Value: -qty}}}}

	result, err := s.coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return 0, 0, fmt.Errorf("products.UpdateOne reserve %s/%s: %w", sku, size, err)
	}

	return result.MatchedCount, result.ModifiedCount, nil
}
