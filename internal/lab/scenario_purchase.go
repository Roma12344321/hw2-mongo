package lab

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"hw2-mongo/internal/domain"
	"hw2-mongo/internal/hwerror"
	"hw2-mongo/internal/service/purchase"
	"hw2-mongo/internal/storage/mongo/fixtures"
)

type dbState struct {
	Products []domain.Product `bson:"products"`
	Orders   []domain.Order   `bson:"orders"`
}

func state(ctx context.Context, r *Runner) (dbState, error) {
	products, err := r.env.Products.All(ctx)
	if err != nil {
		return dbState{}, err
	}

	orders, err := r.env.OrdersSt.All(ctx)
	if err != nil {
		return dbState{}, err
	}

	return dbState{Products: products, Orders: orders}, nil
}

func compactState(s dbState) bson.D {
	stocks := make(bson.A, 0, len(s.Products))
	for _, product := range s.Products {
		variants := make(bson.A, 0, len(product.Variants))
		for _, variant := range product.Variants {
			variants = append(variants, bson.D{
				{Key: "size", Value: variant.Size},
				{Key: "stock", Value: variant.Stock},
			})
		}
		stocks = append(stocks, bson.D{
			{Key: "sku", Value: product.ID},
			{Key: "variants", Value: variants},
		})
	}

	return bson.D{
		{Key: "stocks", Value: stocks},
		{Key: "orderCount", Value: len(s.Orders)},
		{Key: "orderIds", Value: OrderIDs(s.Orders)},
	}
}

func PurchaseScenario(ctx context.Context, r *Runner) error {
	env := r.env

	if err := env.Dataset.Reset(ctx, false); err != nil {
		return fmt.Errorf("reset: %w", err)
	}

	if err := env.Catalog.InstallValidator(ctx); err != nil {
		return fmt.Errorf("installValidator: %w", err)
	}

	request := domain.PurchaseRequest{
		OperationID: "purchase-001",
		UserID:      2500,
		Items: []domain.PurchaseItem{
			{SKU: "A", Size: "M", Qty: 2},
			{SKU: "B", Size: "L", Qty: 1},
		},
	}

	var trace []purchase.UpdateTrace
	emit := func(event purchase.UpdateTrace) {
		trace = append(trace, event)
		Log("TX", event)
	}

	initial, err := state(ctx, r)
	if err != nil {
		return err
	}

	rollbackRequest := domain.PurchaseRequest{
		OperationID: "rollback-001",
		UserID:      request.UserID,
		Items: []domain.PurchaseItem{
			request.Items[0],
			{SKU: "B", Size: "L", Qty: 6},
		},
	}

	rollbackResult, rollbackErr, rollbackEvidence, err := MeasureTransactions(ctx, env.Client,
		"rollback", func() (any, error) {
			return env.Purchase.Purchase(ctx, rollbackRequest, emit)
		})
	if err != nil {
		return err
	}

	if err := r.save("transactions-rollback", rollbackEvidence); err != nil {
		return err
	}

	if err := r.check("second item insufficient",
		hwerror.Code(rollbackErr) == hwerror.CodeItemUnavailable); err != nil {
		return err
	}

	if err := r.check("first update modified before abort",
		len(trace) > 0 && trace[0].SKU == "A" && trace[0].Modified == 1); err != nil {
		return err
	}

	if err := r.check("server observed started and aborted transaction",
		DeltaCounter(rollbackEvidence, "totalStarted") >= 1 &&
			DeltaCounter(rollbackEvidence, "totalAborted") >= 1); err != nil {
		return err
	}

	rolledBack, err := state(ctx, r)
	if err != nil {
		return err
	}

	if err := r.equal("abort restores both full documents and leaves no order",
		rolledBack, initial); err != nil {
		return err
	}

	Log("AFTER ABORT", compactState(rolledBack))

	rollbackArtifact := bson.D{
		{Key: "request", Value: purchase.RequestDoc(rollbackRequest)},
		{Key: "trace", Value: append([]purchase.UpdateTrace{}, trace...)},
		{Key: "error", Value: hwerror.Code(rollbackErr)},
		{Key: "before", Value: initial},
		{Key: "after", Value: rolledBack},
	}

	_ = rollbackResult

	trace = nil

	successResult, successErr, successEvidence, err := MeasureTransactions(ctx, env.Client,
		"success", func() (any, error) {
			return env.Purchase.Purchase(ctx, request, emit)
		})
	if err != nil {
		return err
	}

	if err := r.save("transactions-success", successEvidence); err != nil {
		return err
	}

	if successErr != nil {
		return fmt.Errorf("успешная покупка: %w", successErr)
	}

	if err := r.check("server observed started and committed transaction",
		DeltaCounter(successEvidence, "totalStarted") >= 1 &&
			DeltaCounter(successEvidence, "totalCommitted") >= 1); err != nil {
		return err
	}

	order, ok := successResult.(domain.Order)
	if !ok {
		return fmt.Errorf("успешная покупка вернула %T вместо Order", successResult)
	}

	successState, err := state(ctx, r)
	if err != nil {
		return err
	}

	expectedProducts := fixtures.Products()
	expectedProducts[0].Variants[1].Stock = 8
	expectedProducts[1].Variants[0].Stock = 4

	if err := r.equal("success changes only selected stocks",
		ProductFieldsList(successState.Products), ProductFieldsList(expectedProducts)); err != nil {
		return err
	}

	if err := r.check("exactly one created order",
		len(successState.Orders) == 1 &&
			order.ID == request.OperationID &&
			order.Status == domain.OrderStatusCreated); err != nil {
		return err
	}

	if err := r.check("userId/date types",
		order.UserID == request.UserID && !order.CreatedAt.IsZero()); err != nil {
		return err
	}

	if err := r.equal("snapshot item prices", itemPrices(order.Items),
		bson.A{"100.00", "80.00"}); err != nil {
		return err
	}

	if err := r.equal("saved items match request in original order",
		itemsTriple(order.Items), requestItemsTriple(request)); err != nil {
		return err
	}

	if err := r.equal("returned order is saved order", order, successState.Orders[0]); err != nil {
		return err
	}

	storedTypes, err := env.OrdersSt.StoredTypes(ctx, request.OperationID)
	if err != nil {
		return err
	}

	if err := r.check("stored BSON types: integer user/qty, decimal prices, date",
		storedTypesCorrect(storedTypes)); err != nil {
		return err
	}

	Log("SUCCESS", compactState(successState))
	Log("SAVED ORDER", order)

	if err := env.Products.SetPrice(ctx, "A", fixtures.Decimal("777.00")); err != nil {
		return err
	}

	beforeReplay, err := state(ctx, r)
	if err != nil {
		return err
	}

	replayRequest := domain.PurchaseRequest{
		UserID: 2500,
		Items: []domain.PurchaseItem{
			{Qty: 2, Size: "M", SKU: "A"},
			{Qty: 1, Size: "L", SKU: "B"},
		},
		OperationID: request.OperationID,
	}

	replayed, err := env.Purchase.Purchase(ctx, replayRequest, emit)
	if err != nil {
		return fmt.Errorf("повтор покупки: %w", err)
	}

	if err := r.equal("replay returns original order", replayed, order); err != nil {
		return err
	}

	afterReplay, err := state(ctx, r)
	if err != nil {
		return err
	}

	if err := r.equal("replay no writes despite changed catalog price/key order",
		afterReplay, beforeReplay); err != nil {
		return err
	}

	Log("REPLAY: same order, stocks unchanged", compactState(afterReplay))

	conflictRequest := domain.PurchaseRequest{
		OperationID: request.OperationID,
		UserID:      request.UserID,
		Items: []domain.PurchaseItem{
			{SKU: "A", Size: "M", Qty: 3},
			request.Items[1],
		},
	}

	_, conflictErr := env.Purchase.Purchase(ctx, conflictRequest, emit)
	if conflictErr != nil && hwerror.Code(conflictErr) == "" {
		return conflictErr
	}

	if err := r.check("changed qty rejected as request conflict",
		hwerror.Code(conflictErr) == hwerror.CodeRequestConflict); err != nil {
		return err
	}

	afterConflict, err := state(ctx, r)
	if err != nil {
		return err
	}

	if err := r.equal("conflict changes nothing", afterConflict, afterReplay); err != nil {
		return err
	}

	missingRequest := domain.PurchaseRequest{
		OperationID: "missing-001",
		UserID:      request.UserID,
		Items: []domain.PurchaseItem{
			request.Items[0],
			{SKU: "B", Size: "XXL", Qty: 1},
		},
	}

	_, missingErr := env.Purchase.Purchase(ctx, missingRequest, emit)
	if missingErr != nil && hwerror.Code(missingErr) == "" {
		return missingErr
	}

	if err := r.check("missing variant rejected",
		hwerror.Code(missingErr) == hwerror.CodeItemUnavailable); err != nil {
		return err
	}

	afterMissing, err := state(ctx, r)
	if err != nil {
		return err
	}

	if err := r.equal("missing variant rollback", afterMissing, afterReplay); err != nil {
		return err
	}

	unknownRequest := domain.PurchaseRequest{
		OperationID: "unknown-001",
		UserID:      request.UserID,
		Items: []domain.PurchaseItem{
			request.Items[0],
			{SKU: "NO_SUCH_SKU", Size: "L", Qty: 1},
		},
	}

	_, unknownErr := env.Purchase.Purchase(ctx, unknownRequest, emit)
	if unknownErr != nil && hwerror.Code(unknownErr) == "" {
		return unknownErr
	}

	if err := r.check("unknown SKU rejected",
		hwerror.Code(unknownErr) == hwerror.CodeItemUnavailable); err != nil {
		return err
	}

	afterUnknown, err := state(ctx, r)
	if err != nil {
		return err
	}

	if err := r.equal("unknown SKU rollback", afterUnknown, afterReplay); err != nil {
		return err
	}

	artifact := bson.D{
		{Key: "request", Value: purchase.RequestDoc(request)},
		{Key: "rollback", Value: rollbackArtifact},
		{Key: "success", Value: bson.D{
			{Key: "order", Value: order},
			{Key: "state", Value: successState},
		}},
		{Key: "replay", Value: bson.D{
			{Key: "order", Value: replayed},
			{Key: "before", Value: beforeReplay},
			{Key: "after", Value: afterReplay},
		}},
		{Key: "conflict", Value: bson.D{
			{Key: "error", Value: hwerror.Code(conflictErr)},
			{Key: "after", Value: afterConflict},
		}},
		{Key: "missingError", Value: hwerror.Code(missingErr)},
		{Key: "unknownError", Value: hwerror.Code(unknownErr)},
		{Key: "trace", Value: trace},
		{Key: "checks", Value: r.current.Checks},
	}

	return r.save("purchase", artifact)
}

func itemPrices(items []domain.OrderItem) bson.A {
	prices := make(bson.A, 0, len(items))
	for _, item := range items {
		prices = append(prices, item.Price.String())
	}

	return prices
}

func itemsTriple(items []domain.OrderItem) bson.A {
	triples := make(bson.A, 0, len(items))
	for _, item := range items {
		triples = append(triples, bson.D{
			{Key: "sku", Value: item.SKU},
			{Key: "size", Value: item.Size},
			{Key: "qty", Value: item.Qty},
		})
	}

	return triples
}

func requestItemsTriple(request domain.PurchaseRequest) bson.A {
	items := make(bson.A, 0, len(request.Items))
	for _, item := range request.Items {
		items = append(items, bson.D{
			{Key: "sku", Value: item.SKU},
			{Key: "size", Value: item.Size},
			{Key: "qty", Value: item.Qty},
		})
	}

	return items
}

func storedTypesCorrect(rows []bson.M) bool {
	if len(rows) != 1 {
		return false
	}

	row := rows[0]

	userType, _ := row["userId"].(string)
	createdAtType, _ := row["createdAt"].(string)
	if userType != "int" && userType != "long" {
		return false
	}
	if createdAtType != "date" {
		return false
	}

	items, _ := row["items"].(bson.A)
	for _, rawItem := range items {
		item, _ := rawItem.(bson.D)
		if item == nil {
			return false
		}

		for _, field := range item {
			switch field.Key {
			case "qty":
				qtyType, _ := field.Value.(string)
				if qtyType != "int" && qtyType != "long" {
					return false
				}
			case "price":
				priceType, _ := field.Value.(string)
				if priceType != "decimal" {
					return false
				}
			}
		}
	}

	return true
}
