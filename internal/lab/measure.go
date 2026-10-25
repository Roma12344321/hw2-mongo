package lab

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type txSnapshot struct {
	Host         string    `bson:"host"`
	PID          int64     `bson:"pid"`
	Uptime       float64   `bson:"uptime"`
	LocalTime    time.Time `bson:"localTime"`
	Transactions bson.M    `bson:"transactions"`
}

type TxEvidence struct {
	Scenario    string     `bson:"scenario"`
	Before      txSnapshot `bson:"before"`
	After       txSnapshot `bson:"after"`
	Delta       bson.D     `bson:"delta"`
	SameProcess bool       `bson:"sameProcess"`
}

var txCounterNames = [3]string{"totalStarted", "totalCommitted", "totalAborted"}

func snapshotTransactions(ctx context.Context, client *mongo.Client) (txSnapshot, error) {
	var status struct {
		Host         string    `bson:"host"`
		PID          int64     `bson:"pid"`
		Uptime       float64   `bson:"uptime"`
		LocalTime    time.Time `bson:"localTime"`
		OK           float64   `bson:"ok"`
		Transactions bson.M    `bson:"transactions"`
	}

	err := client.Database("admin").
		RunCommand(ctx, bson.D{{Key: "serverStatus", Value: 1}, {Key: "transactions", Value: 1}}).
		Decode(&status)
	if err != nil {
		return txSnapshot{}, fmt.Errorf("serverStatus: %w", err)
	}

	if status.OK != 1 || status.Transactions == nil {
		return txSnapshot{}, fmt.Errorf("serverStatus.transactions недоступен")
	}

	return txSnapshot{
		Host: status.Host, PID: status.PID, Uptime: status.Uptime,
		LocalTime: status.LocalTime, Transactions: status.Transactions,
	}, nil
}

func counter(snapshot txSnapshot, name string) int64 {
	value, _ := toInt64(snapshot.Transactions[name])

	return value
}

func MeasureTransactions(
	ctx context.Context,
	client *mongo.Client,
	scenario string,
	operation func() (any, error),
) (result any, opErr error, evidence TxEvidence, err error) {
	before, err := snapshotTransactions(ctx, client)
	if err != nil {
		return nil, nil, TxEvidence{}, fmt.Errorf("снимок до: %w", err)
	}

	result, opErr = operation()

	after, err := snapshotTransactions(ctx, client)
	if err != nil {
		return nil, nil, TxEvidence{}, fmt.Errorf("снимок после: %w", err)
	}

	evidence = TxEvidence{
		Scenario: scenario, Before: before, After: after,
		SameProcess: before.Host == after.Host && before.PID == after.PID && after.Uptime >= before.Uptime,
	}
	evidence.Delta = bson.D{
		{Key: "totalStarted", Value: counter(after, "totalStarted") - counter(before, "totalStarted")},
		{Key: "totalCommitted", Value: counter(after, "totalCommitted") - counter(before, "totalCommitted")},
		{Key: "totalAborted", Value: counter(after, "totalAborted") - counter(before, "totalAborted")},
	}

	if !evidence.SameProcess {
		return result, opErr, evidence, fmt.Errorf("повторите опыт: primary или процесс изменился между снимками")
	}

	return result, opErr, evidence, nil
}

func DeltaCounter(evidence TxEvidence, name string) int64 {
	for _, element := range evidence.Delta {
		if element.Key == name {
			value, _ := element.Value.(int64)

			return value
		}
	}

	return 0
}
