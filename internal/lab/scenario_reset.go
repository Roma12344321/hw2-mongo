package lab

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"hw2-mongo/internal/storage/mongo/fixtures"
)

const (
	ResetModeData  = "data"
	ResetModeIndex = "index"
)

func ResetScenario(ctx context.Context, r *Runner) error {
	mode := r.env.ResetMode
	if mode != ResetModeData && mode != ResetModeIndex {
		return fmt.Errorf("используйте reset или reset index, получен %q", mode)
	}

	if err := r.env.Dataset.Reset(ctx, mode == ResetModeData); err != nil {
		return fmt.Errorf("reset: %w", err)
	}

	if mode == ResetModeIndex {
		if err := r.env.Dataset.LoadLargeOrders(ctx); err != nil {
			return fmt.Errorf("load large orders: %w", err)
		}
	}

	products, err := r.env.Products.All(ctx)
	if err != nil {
		return err
	}

	ordersCount, err := r.env.OrdersSt.Count(ctx)
	if err != nil {
		return err
	}

	expectedOrders := int64(35)
	if mode == ResetModeIndex {
		expectedOrders = 10000
	}

	if err := r.check("fixtures loaded",
		int64(len(products)) == 6 && ordersCount == expectedOrders); err != nil {
		return err
	}

	artifact := bson.D{
		{Key: "datasetVersion", Value: fixtures.DatasetVersion},
		{Key: "mode", Value: mode},
		{Key: "products", Value: len(products)},
		{Key: "orders", Value: ordersCount},
	}

	if err := r.save("reset", artifact); err != nil {
		return err
	}

	Log("DATASET", artifact)

	return nil
}
