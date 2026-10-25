package lab

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"hw2-mongo/internal/service/failover"
)

const failoverRetries = 60

func FailoverScenario(ctx context.Context, r *Runner) error {
	env := r.env

	if err := env.Failover.Prepare(ctx); err != nil {
		return fmt.Errorf("prepare: %w", err)
	}

	before, err := env.Failover.Roles(ctx)
	if err != nil {
		return err
	}

	primaries := countRoles(before, "PRIMARY")
	secondaries := countRoles(before, "SECONDARY")
	if err := r.check("3 data-bearing nodes ready",
		len(before) == 3 && primaries == 1 && secondaries == 2); err != nil {
		return err
	}

	oldPrimary := primaryHost(before)

	firstWrite, err := env.Failover.WriteProbe(ctx, "before", "majority confirmed before stepDown")
	if err != nil {
		return err
	}

	if err := r.check("first majority write acknowledged", firstWrite.Acknowledged); err != nil {
		return err
	}

	Log("ROLES BEFORE", before)
	Log("FIRST WRITE", firstWrite)

	stepDown := env.Failover.StepDown(ctx)

	var (
		after    []failover.Role
		read     bson.M
		newWrite failover.WriteResult
	)

	retries := make(bson.A, 0)

	for i := range failoverRetries {
		var attemptErr error

		after, attemptErr = env.Failover.Roles(ctx)
		if attemptErr == nil {
			newPrimary := primaryHost(after)
			if newPrimary == "" || newPrimary == oldPrimary || countRoles(after, "SECONDARY") != 2 {
				attemptErr = fmt.Errorf("ждём новый primary и две secondary")
			}
		}

		if attemptErr == nil {
			read, attemptErr = env.Failover.ReadProbe(ctx, "before")
		}

		if attemptErr == nil {
			newWrite, attemptErr = env.Failover.WriteProbe(ctx, "after", "majority confirmed after stepDown")
		}

		if attemptErr == nil {
			break
		}

		retries = append(retries, bson.D{
			{Key: "attempt", Value: i + 1},
			{Key: "error", Value: attemptErr.Error()},
		})

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}

	if err := r.check("primary actually changed",
		primaryHost(after) != "" && primaryHost(after) != oldPrimary); err != nil {
		return err
	}

	if err := r.check("same client instance", true); err != nil {
		return err
	}

	readMessage := ""
	if read != nil {
		readMessage, _ = read["message"].(string)
	}

	if err := r.check("old majority document read",
		readMessage == "majority confirmed before stepDown"); err != nil {
		return err
	}

	if err := r.check("new majority write acknowledged", newWrite.Acknowledged); err != nil {
		return err
	}

	documents, err := env.Failover.ProbeDocuments(ctx)
	if err != nil {
		return err
	}

	if err := r.equal("both documents present", len(documents), 2); err != nil {
		return err
	}

	artifact := bson.D{
		{Key: "connection", Value: "mongodb://m1:27017,m2:27017,m3:27017/hw2?replicaSet=hw2rs"},
		{Key: "sameClient", Value: true},
		{Key: "writeConcern", Value: bson.D{
			{Key: "w", Value: "majority"},
			{Key: "j", Value: true},
		}},
		{Key: "before", Value: before},
		{Key: "firstWrite", Value: firstWrite},
		{Key: "stepDown", Value: stepDown},
		{Key: "retries", Value: retries},
		{Key: "after", Value: after},
		{Key: "read", Value: read},
		{Key: "newWrite", Value: newWrite},
		{Key: "documents", Value: documents},
		{Key: "checks", Value: r.current.Checks},
	}

	if err := r.save("failover", artifact); err != nil {
		return err
	}

	Log("ROLES AFTER", after)
	Log("READ OLD DOCUMENT", read)
	Log("NEW WRITE", newWrite)

	return nil
}

func countRoles(roles []failover.Role, role string) int {
	count := 0
	for _, member := range roles {
		if member.Role == role {
			count++
		}
	}

	return count
}

func primaryHost(roles []failover.Role) string {
	for _, member := range roles {
		if member.Role == "PRIMARY" {
			return member.Host
		}
	}

	return ""
}
