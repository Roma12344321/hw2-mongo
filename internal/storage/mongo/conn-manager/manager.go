package conn_manager

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"

	"hw2-mongo/internal/config"
)

const disconnectTimeout = 5 * time.Second

type ConnManager struct {
	client         *mongo.Client
	dbName         string
	failoverDBName string
}

func New(cfg *config.Config) (*ConnManager, error) {
	ctx := context.Background()

	client, err := mongo.Connect(options.Client().ApplyURI(cfg.Mongo.URI))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongo: %w", err)
	}

	if err = client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping mongo: %w", err)
	}

	return &ConnManager{
		client:         client,
		dbName:         cfg.Mongo.DBName,
		failoverDBName: cfg.Mongo.FailoverDBName,
	}, nil
}

func (m *ConnManager) GetClient() *mongo.Client {
	return m.client
}

func (m *ConnManager) GetDB() *mongo.Database {
	return m.client.Database(m.dbName)
}

func (m *ConnManager) GetFailoverDB() *mongo.Database {
	return m.client.Database(m.failoverDBName)
}

func (m *ConnManager) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), disconnectTimeout)
	defer cancel()

	if err := m.client.Disconnect(ctx); err != nil {
		return fmt.Errorf("client.Disconnect: %w", err)
	}

	return nil
}

func MajorityJournal() *writeconcern.WriteConcern {
	journal := true

	return &writeconcern.WriteConcern{W: "majority", Journal: &journal}
}
