package lab

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"hw2-mongo/internal/domain"
	"hw2-mongo/internal/storage/mongo/fixtures"
)

func DataScenario(ctx context.Context, r *Runner) error {
	env := r.env

	if err := env.Dataset.Reset(ctx, true); err != nil {
		return fmt.Errorf("reset: %w", err)
	}

	if err := env.Catalog.InstallValidator(ctx); err != nil {
		return fmt.Errorf("installValidator: %w", err)
	}

	options, err := env.Catalog.ValidatorOptions(ctx)
	if err != nil {
		return err
	}

	if err := r.check("validator strict/error installed",
		options["validator"] != nil &&
			options["validationLevel"] == "strict" &&
			options["validationAction"] == "error"); err != nil {
		return err
	}

	base := fixtures.Products()[0]

	probes := []struct {
		id           string
		mutate       func(bson.D) bson.D
		expectedCode int64
	}{
		{"valid", func(doc bson.D) bson.D { return doc }, 0},
		{"string-price", func(doc bson.D) bson.D {
			return setField(doc, "price", "100.00")
		}, 121},
		{"negative-stock", func(doc bson.D) bson.D {
			return setField(doc, "variants", bson.A{bson.D{
				{Key: "size", Value: "L"},
				{Key: "stock", Value: int32(-1)},
			}})
		}, 121},
	}

	validation := make(bson.A, 0, len(probes))
	for _, probe := range probes {
		doc := probe.mutate(withID(toBsonD(base), probe.id))

		code, details := insertErrorCode(env.Products.InsertRaw(ctx, doc))
		if err := r.check("validation "+probe.id, code == probe.expectedCode); err != nil {
			return err
		}

		validation = append(validation, bson.D{
			{Key: "id", Value: probe.id},
			{Key: "code", Value: code},
			{Key: "details", Value: details},
		})

		if code == 0 {
			if err := env.Products.DeleteByID(ctx, probe.id); err != nil {
				return err
			}
		}
	}

	searches := make(bson.A, 0, len(fixtures.SearchCases())+1)
	for _, testCase := range fixtures.SearchCases() {
		documents, err := env.Catalog.SearchProducts(ctx, domain.SearchInput{
			Category: testCase.Category, Size: testCase.Size, Qty: testCase.Qty,
		})
		if err != nil {
			return err
		}

		expectedDocuments, err := env.Products.FindByIDs(ctx, testCase.ExpectedID)
		if err != nil {
			return err
		}

		label := fmt.Sprintf("%s/%s/%d", testCase.Category, testCase.Size, testCase.Qty)
		if err := r.equal("complete product documents "+label,
			ProductFieldsList(documents), ProductFieldsList(expectedDocuments)); err != nil {
			return err
		}

		if err := r.equal("search "+label, ProductIDs(documents), testCase.ExpectedID); err != nil {
			return err
		}

		searches = append(searches, bson.D{
			{Key: "category", Value: testCase.Category},
			{Key: "size", Value: testCase.Size},
			{Key: "qty", Value: testCase.Qty},
			{Key: "documents", Value: documents},
			{Key: "found", Value: ProductIDs(documents)},
			{Key: "expected", Value: testCase.ExpectedID},
		})
	}

	if err := env.Products.SetVariantStock(ctx, "A", "L", 2); err != nil {
		return err
	}

	changedDocuments, err := env.Catalog.SearchProducts(ctx,
		domain.SearchInput{Category: "clothes", Size: "L", Qty: 2})
	if err != nil {
		return err
	}

	if err := r.equal("L=2 includes A", ProductIDs(changedDocuments),
		[]string{"A", "B", "C"}); err != nil {
		return err
	}

	searches = append(searches, bson.D{
		{Key: "afterChange", Value: "A/L=2"},
		{Key: "documents", Value: changedDocuments},
		{Key: "found", Value: ProductIDs(changedDocuments)},
	})

	historyInput := domain.HistoryInput{UserID: 2500, Status: domain.OrderStatusPaid}
	historyDocuments, err := env.Orders.History(ctx, historyInput)
	if err != nil {
		return err
	}

	expectedHistory := ordersByIDs(fixtures.Orders(), fixtures.HistoryExpected())
	if err := r.equal("complete history documents",
		OrderFieldsList(historyDocuments), OrderFieldsList(expectedHistory)); err != nil {
		return err
	}

	if err := r.equal("history 20 IDs including date ties",
		OrderIDs(historyDocuments), fixtures.HistoryExpected()); err != nil {
		return err
	}

	otherUser, err := env.Orders.History(ctx,
		domain.HistoryInput{UserID: 2501, Status: domain.OrderStatusPaid})
	if err != nil {
		return err
	}

	if err := r.equal("history other user", OrderIDs(otherUser), []string{"other-user"}); err != nil {
		return err
	}

	otherStatus, err := env.Orders.History(ctx,
		domain.HistoryInput{UserID: 2500, Status: domain.OrderStatusCancelled})
	if err != nil {
		return err
	}

	if err := r.equal("history other status", OrderIDs(otherStatus),
		[]string{"other-status", "excluded-cancelled"}); err != nil {
		return err
	}

	noHistory, err := env.Orders.History(ctx,
		domain.HistoryInput{UserID: 9999, Status: domain.OrderStatusPaid})
	if err != nil {
		return err
	}

	if err := r.equal("history no matches", OrderIDs(noHistory), []string{}); err != nil {
		return err
	}

	fullReport, err := env.Report.Sales(ctx, fixtures.ReportFrom, fixtures.ReportTo)
	if err != nil {
		return err
	}

	actual := reportRows(fullReport)
	if err := r.equal("report top 5 amounts/order/names",
		actual, expectedReportRows()); err != nil {
		return err
	}

	emptyFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	emptyTo := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	emptyRows, err := env.Report.Sales(ctx, emptyFrom, emptyTo)
	if err != nil {
		return err
	}

	if err := r.equal("empty interval", reportRows(emptyRows), bson.A{}); err != nil {
		return err
	}

	boundaryTo := time.Date(2026, 1, 1, 0, 0, 0, 1000000, time.UTC)
	boundaryRows, err := env.Report.Sales(ctx, fixtures.ReportFrom, boundaryTo)
	if err != nil {
		return err
	}

	if err := r.equal("from included", skuRevenuePairs(boundaryRows),
		bson.A{bson.A{"B", "320.00"}, bson.A{"A", "200.00"}}); err != nil {
		return err
	}

	if err := env.Products.SetPrice(ctx, "A", fixtures.Decimal("999.00")); err != nil {
		return err
	}

	afterPriceRows, err := env.Report.Sales(ctx, fixtures.ReportFrom, fixtures.ReportTo)
	if err != nil {
		return err
	}

	if err := r.equal("catalog price does not rewrite revenue",
		reportRows(afterPriceRows), actual); err != nil {
		return err
	}

	artifact := bson.D{
		{Key: "options", Value: options},
		{Key: "validation", Value: validation},
		{Key: "searches", Value: searches},
		{Key: "history", Value: historyDocuments},
		{Key: "historyIds", Value: OrderIDs(historyDocuments)},
		{Key: "otherUser", Value: otherUser},
		{Key: "otherStatus", Value: otherStatus},
		{Key: "noHistory", Value: noHistory},
		{Key: "historyExpected", Value: fixtures.HistoryExpected()},
		{Key: "report", Value: fullReport},
		{Key: "reportParameters", Value: bson.D{
			{Key: "from", Value: fixtures.ReportFrom},
			{Key: "to", Value: fixtures.ReportTo},
		}},
		{Key: "emptyReport", Value: bson.D{
			{Key: "from", Value: emptyFrom},
			{Key: "to", Value: emptyTo},
			{Key: "rows", Value: emptyRows},
		}},
		{Key: "boundaryReport", Value: bson.D{
			{Key: "from", Value: fixtures.ReportFrom},
			{Key: "to", Value: boundaryTo},
			{Key: "rows", Value: boundaryRows},
		}},
		{Key: "afterPriceReport", Value: bson.D{
			{Key: "from", Value: fixtures.ReportFrom},
			{Key: "to", Value: fixtures.ReportTo},
			{Key: "changedProduct", Value: bson.D{
				{Key: "sku", Value: "A"},
				{Key: "price", Value: fixtures.Decimal("999.00")},
			}},
			{Key: "rows", Value: afterPriceRows},
		}},
		{Key: "expected", Value: fixtures.ReportExpected()},
		{Key: "checks", Value: r.current.Checks},
	}

	if err := r.save("data", artifact); err != nil {
		return err
	}

	Log("REPORT: [2026-01-01, 2026-02-01), paid", actual)
	Log("REPORT: empty interval", emptyRows)
	Log("REPORT: included from boundary", boundaryRows)
	Log("REPORT: after catalog price change", afterPriceRows)
	Log("HISTORY: userId=2500, status=paid", OrderIDs(historyDocuments))

	return nil
}

func withID(doc bson.D, id string) bson.D {
	return setField(doc, "_id", id)
}

func setField(doc bson.D, key string, value any) bson.D {
	for i, element := range doc {
		if element.Key == key {
			doc[i].Value = value

			return doc
		}
	}

	return append(doc, bson.E{Key: key, Value: value})
}

func ordersByIDs(all []domain.Order, ids []string) []domain.Order {
	byID := make(map[string]domain.Order, len(all))
	for _, order := range all {
		byID[order.ID] = order
	}

	selected := make([]domain.Order, 0, len(ids))
	for _, id := range ids {
		selected = append(selected, byID[id])
	}

	return selected
}

func expectedReportRows() bson.A {
	expected := make(bson.A, 0, 5)
	for _, row := range fixtures.ReportExpected() {
		expected = append(expected, bson.D{
			{Key: "sku", Value: row.SKU},
			{Key: "name", Value: row.Name},
			{Key: "qty", Value: row.Qty},
			{Key: "revenue", Value: row.Revenue},
		})
	}

	return expected
}

func skuRevenuePairs(rows []domain.ReportRow) bson.A {
	pairs := make(bson.A, 0, len(rows))
	for _, row := range rows {
		pairs = append(pairs, bson.A{row.SKU, row.Revenue.String()})
	}

	return pairs
}
