package lab

import (
	"context"
	"fmt"
	"sort"

	"go.mongodb.org/mongo-driver/v2/bson"

	"hw2-mongo/internal/domain"
	"hw2-mongo/internal/storage/mongo/fixtures"
)

func IndexScenario(ctx context.Context, r *Runner) error {
	env := r.env

	if err := env.Dataset.Reset(ctx, false); err != nil {
		return fmt.Errorf("reset: %w", err)
	}

	if err := env.Dataset.LoadLargeOrders(ctx); err != nil {
		return fmt.Errorf("load large orders: %w", err)
	}

	count, err := env.Orders.Count(ctx)
	if err != nil {
		return err
	}

	if err := r.equal("10000 documents", count, 10000); err != nil {
		return err
	}

	indexNames, err := env.Orders.IndexNames(ctx)
	if err != nil {
		return err
	}

	if err := r.equal("only _id index initially", indexNames, []string{"_id_"}); err != nil {
		return err
	}

	historyInput := domain.HistoryInput{UserID: 2500, Status: domain.OrderStatusPaid}

	beforeDocuments, err := env.Orders.History(ctx, historyInput)
	if err != nil {
		return err
	}

	beforeIDs := OrderIDs(beforeDocuments)

	beforePlan, err := env.Orders.ExplainHistory(ctx, historyInput)
	if err != nil {
		return err
	}

	if err := env.Orders.CreateHistoryIndex(ctx); err != nil {
		return err
	}

	afterPlan, err := env.Orders.ExplainHistory(ctx, historyInput)
	if err != nil {
		return err
	}

	afterDocuments, err := env.Orders.History(ctx, historyInput)
	if err != nil {
		return err
	}

	afterIDs := OrderIDs(afterDocuments)

	if err := r.equal("same IDs and order", afterIDs, beforeIDs); err != nil {
		return err
	}

	if err := r.equal("matches independent fixture order", afterIDs, expectedLargeHistory()); err != nil {
		return err
	}

	beforeRow, err := planRow(beforePlan)
	if err != nil {
		return err
	}

	afterRow, err := planRow(afterPlan)
	if err != nil {
		return err
	}

	if err := r.check("before scans collection + SORT",
		hasStage(beforePlan["queryPlanner"], "COLLSCAN") &&
			hasStage(beforePlan["queryPlanner"], "SORT")); err != nil {
		return err
	}

	if err := r.check("after uses IXSCAN without SORT",
		hasStage(afterPlan["queryPlanner"], "IXSCAN") &&
			!hasStage(afterPlan["queryPlanner"], "SORT")); err != nil {
		return err
	}

	if err := r.check("examines fewer documents",
		rowField(afterRow, "totalDocsExamined") < rowField(beforeRow, "totalDocsExamined")); err != nil {
		return err
	}

	summary := bson.D{
		{Key: "filter", Value: bson.D{
			{Key: "userId", Value: int32(2500)},
			{Key: "status", Value: domain.OrderStatusPaid},
		}},
		{Key: "sort", Value: bson.D{
			{Key: "createdAt", Value: -1},
			{Key: "_id", Value: 1},
		}},
		{Key: "index", Value: env.OrdersSt.HistoryIndexSpec()},
		{Key: "hint", Value: false},
		{Key: "before", Value: beforeRow},
		{Key: "after", Value: afterRow},
		{Key: "beforeIds", Value: beforeIDs},
		{Key: "afterIds", Value: afterIDs},
		{Key: "beforeDocuments", Value: beforeDocuments},
		{Key: "afterDocuments", Value: afterDocuments},
	}

	for name, artifact := range map[string]any{
		"explain-before": beforePlan,
		"explain-after":  afterPlan,
		"index":          summary,
	} {
		if err := r.save(name, artifact); err != nil {
			return err
		}
	}

	Log("INDEX", bson.D{
		{Key: "filter", Value: summary[1].Value},
		{Key: "sort", Value: summary[2].Value},
		{Key: "index", Value: summary[3].Value},
		{Key: "hint", Value: false},
		{Key: "before", Value: beforeRow},
		{Key: "after", Value: afterRow},
	})

	return nil
}

func expectedLargeHistory() []string {
	all := fixtures.LargeOrders()

	matched := make([]domain.Order, 0, len(all)/3)
	for _, order := range all {
		if order.UserID == 2500 && order.Status == domain.OrderStatusPaid {
			matched = append(matched, order)
		}
	}

	sort.SliceStable(matched, func(i, j int) bool {
		if !matched[i].CreatedAt.Equal(matched[j].CreatedAt) {
			return matched[i].CreatedAt.After(matched[j].CreatedAt)
		}

		return matched[i].ID < matched[j].ID
	})

	if len(matched) > 20 {
		matched = matched[:20]
	}

	return OrderIDs(matched)
}

func rowField(row bson.D, field string) int64 {
	for _, element := range row {
		if element.Key == field {
			value, _ := element.Value.(int64)

			return value
		}
	}

	return 0
}
