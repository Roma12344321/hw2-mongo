package failover

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

const probeCollection = "probe"

type ConnManager interface {
	GetClient() *mongo.Client
	GetFailoverDB() *mongo.Database
}

type Service struct {
	logger zerolog.Logger
	client *mongo.Client
	db     *mongo.Database

	probe *mongo.Collection
}

func New(logger zerolog.Logger, connManager ConnManager) *Service {
	journal := true
	writeConcern := &writeconcern.WriteConcern{W: "majority", Journal: &journal}
	db := connManager.GetFailoverDB()

	return &Service{
		logger: logger.With().Str("component", "failover.Service").Logger(),
		client: connManager.GetClient(),
		db:     db,
		probe:  db.Collection(probeCollection, options.Collection().SetWriteConcern(writeConcern)),
	}
}

func (s *Service) Prepare(ctx context.Context) error {
	if err := s.db.Drop(ctx); err != nil {
		return fmt.Errorf("drop %s: %w", s.db.Name(), err)
	}

	if err := s.db.CreateCollection(ctx, probeCollection); err != nil {
		return fmt.Errorf("createCollection %s: %w", probeCollection, err)
	}

	return nil
}

type Role struct {
	Host string `bson:"host"`
	Role string `bson:"role"`
}

func (s *Service) Roles(ctx context.Context) ([]Role, error) {
	var status struct {
		Members []struct {
			Name     string `bson:"name"`
			StateStr string `bson:"stateStr"`
		} `bson:"members"`
	}

	err := s.client.Database("admin").
		RunCommand(ctx, bson.D{{Key: "replSetGetStatus", Value: 1}}).Decode(&status)
	if err != nil {
		return nil, fmt.Errorf("replSetGetStatus: %w", err)
	}

	roles := make([]Role, 0, len(status.Members))
	for _, member := range status.Members {
		roles = append(roles, Role{Host: member.Name, Role: member.StateStr})
	}

	return roles, nil
}

type WriteResult struct {
	Acknowledged  bool  `bson:"acknowledged"`
	MatchedCount  int64 `bson:"matchedCount"`
	ModifiedCount int64 `bson:"modifiedCount"`
	UpsertedCount int64 `bson:"upsertedCount"`
}

func (s *Service) WriteProbe(ctx context.Context, id, message string) (WriteResult, error) {
	filter := bson.M{"_id": id}
	update := bson.M{"$set": bson.M{"message": message}}

	result, err := s.probe.UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true))
	if err != nil {
		return WriteResult{}, fmt.Errorf("probe.UpdateOne %s: %w", id, err)
	}

	return WriteResult{
		Acknowledged:  result.Acknowledged,
		MatchedCount:  result.MatchedCount,
		ModifiedCount: result.ModifiedCount,
		UpsertedCount: result.UpsertedCount,
	}, nil
}

func (s *Service) ReadProbe(ctx context.Context, id string) (bson.M, error) {
	var doc bson.M
	err := s.probe.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("probe.FindOne %s: %w", id, err)
	}

	return doc, nil
}

func (s *Service) ProbeDocuments(ctx context.Context) ([]bson.M, error) {
	cursor, err := s.probe.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("probe.Find: %w", err)
	}

	docs := make([]bson.M, 0)
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("cursor.All: %w", err)
	}

	return docs, nil
}

func (s *Service) ProbeCount(ctx context.Context) (int64, error) {
	count, err := s.probe.CountDocuments(ctx, bson.M{})
	if err != nil {
		return 0, fmt.Errorf("probe.CountDocuments: %w", err)
	}

	return count, nil
}

type StepDownResult struct {
	Message string `bson:"message"`
	Code    any    `bson:"code"`
	OK      any    `bson:"ok,omitempty"`
}

func (s *Service) StepDown(ctx context.Context) StepDownResult {
	command := bson.D{
		{Key: "replSetStepDown", Value: 60},
		{Key: "secondaryCatchUpPeriodSecs", Value: 10},
	}

	var reply bson.M
	err := s.client.Database("admin").RunCommand(ctx, command).Decode(&reply)
	if err != nil {
		return StepDownResult{Message: err.Error()}
	}

	return StepDownResult{OK: reply["ok"]}
}
