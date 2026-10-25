package txmanager

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"

	connManager "hw2-mongo/internal/storage/mongo/conn-manager"
)

type ConnManager interface {
	GetClient() *mongo.Client
}

type Manager struct {
	connManager ConnManager
}

func New(connManager ConnManager) *Manager {
	return &Manager{connManager: connManager}
}

func (m *Manager) Do(ctx context.Context, process func(ctx context.Context) error) error {
	session, err := m.connManager.GetClient().StartSession()
	if err != nil {
		return fmt.Errorf("startSession: %w", err)
	}
	defer session.EndSession(context.WithoutCancel(ctx))

	txOpts := options.Transaction().
		SetReadConcern(readconcern.Snapshot()).
		SetWriteConcern(connManager.MajorityJournal()).
		SetReadPreference(readpref.Primary())

	if err := session.StartTransaction(txOpts); err != nil {
		return fmt.Errorf("startTransaction: %w", err)
	}

	if err := process(mongo.NewSessionContext(ctx, session)); err != nil {

		if abortErr := session.AbortTransaction(context.WithoutCancel(ctx)); abortErr != nil {
			return fmt.Errorf("abort после ошибки %v: %w", err, abortErr)
		}

		return err
	}

	if err := session.CommitTransaction(ctx); err != nil {
		return fmt.Errorf("commitTransaction: %w", err)
	}

	return nil
}
