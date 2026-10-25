package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"

	"hw2-mongo/internal/domain"
)

var HistoryIndexKeys = bson.D{
	{Key: "userId", Value: 1},
	{Key: "status", Value: 1},
	{Key: "createdAt", Value: -1},
	{Key: "_id", Value: 1},
}

const (
	historyIndexName = "history_user_status_date_id"
	historyLimit     = int64(20)
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
	coll := connManager.GetDB().Collection("orders")

	return &Storage{
		connManager:  connManager,
		coll:         coll,
		collMajority: coll.Clone(options.Collection().SetWriteConcern(majorityJournal())),
	}
}

func (s *Storage) InsertMany(ctx context.Context, orders []domain.Order) error {
	if len(orders) == 0 {
		return nil
	}

	if _, err := s.collMajority.InsertMany(ctx, orders); err != nil {
		return fmt.Errorf("orders.InsertMany: %w", err)
	}

	return nil
}

func (s *Storage) InsertOne(ctx context.Context, order domain.Order) error {
	if _, err := s.coll.InsertOne(ctx, order); err != nil {
		return fmt.Errorf("orders.InsertOne: %w", err)
	}

	return nil
}

func (s *Storage) All(ctx context.Context) ([]domain.Order, error) {
	cursor, err := s.coll.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("orders.Find all: %w", err)
	}

	orders := make([]domain.Order, 0)
	if err := cursor.All(ctx, &orders); err != nil {
		return nil, fmt.Errorf("cursor.All: %w", err)
	}

	return orders, nil
}

func (s *Storage) Count(ctx context.Context) (int64, error) {
	count, err := s.coll.CountDocuments(ctx, bson.M{})
	if err != nil {
		return 0, fmt.Errorf("orders.CountDocuments: %w", err)
	}

	return count, nil
}

func (s *Storage) FindByID(ctx context.Context, id string) (domain.Order, error) {
	var order domain.Order
	if err := s.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&order); err != nil {
		return domain.Order{}, fmt.Errorf("orders.FindOne %s: %w", id, err)
	}

	return order, nil
}

func HistorySpec(in domain.HistoryInput) (filter, sort bson.D) {
	filter = bson.D{
		{Key: "userId", Value: in.UserID},
		{Key: "status", Value: in.Status},
	}
	sort = bson.D{
		{Key: "createdAt", Value: -1},
		{Key: "_id", Value: 1},
	}

	return filter, sort
}

func (s *Storage) FindHistory(ctx context.Context, in domain.HistoryInput) ([]domain.Order, error) {
	filter, sort := HistorySpec(in)

	cursor, err := s.coll.Find(ctx, filter,
		options.Find().SetSort(sort).SetLimit(historyLimit))
	if err != nil {
		return nil, fmt.Errorf("orders.Find history: %w", err)
	}

	history := make([]domain.Order, 0)
	if err := cursor.All(ctx, &history); err != nil {
		return nil, fmt.Errorf("cursor.All: %w", err)
	}

	return history, nil
}

func (s *Storage) ExplainHistory(ctx context.Context, in domain.HistoryInput) (bson.M, error) {
	filter, sort := HistorySpec(in)

	command := bson.D{
		{Key: "explain", Value: bson.D{
			{Key: "find", Value: "orders"},
			{Key: "filter", Value: filter},
			{Key: "sort", Value: sort},
			{Key: "limit", Value: historyLimit},
		}},
		{Key: "verbosity", Value: "executionStats"},
	}

	plan := bson.M{}
	if err := s.coll.Database().RunCommand(ctx, command).Decode(&plan); err != nil {
		return nil, fmt.Errorf("explain: %w", err)
	}

	return plan, nil
}

func (s *Storage) CreateHistoryIndex(ctx context.Context) error {
	model := mongo.IndexModel{
		Keys:    HistoryIndexKeys,
		Options: options.Index().SetName(historyIndexName),
	}

	if _, err := s.coll.Indexes().CreateOne(ctx, model); err != nil {
		return fmt.Errorf("createIndex %s: %w", historyIndexName, err)
	}

	return nil
}

func (s *Storage) HistoryIndexSpec() bson.D {
	return HistoryIndexKeys
}

func (s *Storage) IndexNames(ctx context.Context) ([]string, error) {
	cursor, err := s.coll.Indexes().List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listIndexes: %w", err)
	}

	var indexes []bson.M
	if err := cursor.All(ctx, &indexes); err != nil {
		return nil, fmt.Errorf("cursor.All: %w", err)
	}

	names := make([]string, 0, len(indexes))
	for _, index := range indexes {
		names = append(names, fmt.Sprintf("%v", index["name"]))
	}

	return names, nil
}

func (s *Storage) AggregateReport(ctx context.Context, from, to time.Time) ([]domain.ReportRow, error) {
	pipeline := bson.A{
		bson.D{{Key: "$match", Value: bson.D{
			{Key: "status", Value: domain.OrderStatusPaid},
			{Key: "createdAt", Value: bson.D{
				{Key: "$gte", Value: from},
				{Key: "$lt", Value: to},
			}},
		}}},
		bson.D{{Key: "$unwind", Value: "$items"}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$items.sku"},
			{Key: "qty", Value: bson.D{{Key: "$sum", Value: "$items.qty"}}},
			{Key: "revenue", Value: bson.D{{Key: "$sum", Value: bson.D{
				{Key: "$multiply", Value: bson.A{"$items.qty", "$items.price"}},
			}}}},
		}}},
		bson.D{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "products"},
			{Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "product"},
		}}},
		bson.D{{Key: "$unwind", Value: "$product"}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "sku", Value: "$_id"},
			{Key: "name", Value: "$product.name"},
		}}},
		bson.D{{Key: "$sort", Value: bson.D{
			{Key: "revenue", Value: -1},
			{Key: "sku", Value: 1},
		}}},
		bson.D{{Key: "$limit", Value: 5}},
		bson.D{{Key: "$project", Value: bson.D{
			{Key: "_id", Value: 0},
			{Key: "sku", Value: 1},
			{Key: "name", Value: 1},
			{Key: "qty", Value: 1},
			{Key: "revenue", Value: 1},
		}}},
	}

	cursor, err := s.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("orders.Aggregate: %w", err)
	}

	rows := make([]domain.ReportRow, 0)
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, fmt.Errorf("cursor.All: %w", err)
	}

	return rows, nil
}

func (s *Storage) StoredTypes(ctx context.Context, id string) ([]bson.M, error) {
	pipeline := bson.A{
		bson.D{{Key: "$match", Value: bson.M{"_id": id}}},
		bson.D{{Key: "$project", Value: bson.D{
			{Key: "userId", Value: bson.D{{Key: "$type", Value: "$userId"}}},
			{Key: "createdAt", Value: bson.D{{Key: "$type", Value: "$createdAt"}}},
			{Key: "items", Value: bson.D{{Key: "$map", Value: bson.D{
				{Key: "input", Value: "$items"},
				{Key: "as", Value: "item"},
				{Key: "in", Value: bson.D{
					{Key: "qty", Value: bson.D{{Key: "$type", Value: "$$item.qty"}}},
					{Key: "price", Value: bson.D{{Key: "$type", Value: "$$item.price"}}},
				}},
			}}}},
		}}},
	}

	cursor, err := s.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("orders.Aggregate types: %w", err)
	}

	rows := make([]bson.M, 0)
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, fmt.Errorf("cursor.All: %w", err)
	}

	return rows, nil
}
