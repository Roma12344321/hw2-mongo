package lab

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"hw2-mongo/internal/domain"
)

func TestNormalizedSortsKeysAndKeepsArrays(t *testing.T) {
	a := bson.D{{Key: "b", Value: 1}, {Key: "a", Value: bson.A{int32(1), int32(2)}}}
	b := bson.D{{Key: "a", Value: bson.A{int32(1), int32(2)}}, {Key: "b", Value: 1}}
	c := bson.D{{Key: "a", Value: bson.A{int32(2), int32(1)}}, {Key: "b", Value: 1}}

	na, err := normalized(a)
	if err != nil {
		t.Fatal(err)
	}
	nb, _ := normalized(b)
	nc, _ := normalized(c)

	if na != nb {
		t.Errorf("ключи должны сравниваться независимо от порядка: %s != %s", na, nb)
	}
	if na == nc {
		t.Error("порядок элементов массива должен учитываться")
	}
}

func TestNormalizedEncodesBSONScalars(t *testing.T) {
	decimal, _ := bson.ParseDecimal128("100.00")
	date := time.Date(2026, 1, 31, 23, 59, 59, 999000000, time.UTC)

	doc := bson.D{
		{Key: "price", Value: decimal},
		{Key: "createdAt", Value: date},
	}

	got, err := normalized(doc)
	if err != nil {
		t.Fatal(err)
	}

	want := `{"createdAt":{"$date":"2026-01-31T23:59:59.999Z"},"price":{"$numberDecimal":"100.00"}}`
	if got != want {
		t.Errorf("получено %s, ожидалось %s", got, want)
	}
}

func TestNormalizedEmptySlice(t *testing.T) {
	orders := make([]domain.Order, 0)
	got, err := normalized(orders)
	if err != nil {
		t.Fatal(err)
	}
	if got != "[]" {
		t.Errorf("пустой срез должен быть [], получено %s", got)
	}
}

func TestHasStageRecursive(t *testing.T) {
	plan := bson.M{
		"queryPlanner": bson.M{
			"winningPlan": bson.M{
				"stage": "FETCH",
				"inputStage": bson.M{
					"stage":     "IXSCAN",
					"indexName": "history_user_status_date_id",
				},
			},
		},
	}

	if !hasStage(plan, "IXSCAN") {
		t.Error("IXSCAN должен найтись во вложении")
	}
	if hasStage(plan, "SORT") {
		t.Error("SORT в плане нет")
	}
}

func TestPlanRowExtractsExecutionStats(t *testing.T) {
	plan := bson.M{
		"queryPlanner": bson.M{"winningPlan": bson.M{
			"stage": "COLLSCAN", "direction": "forward",
		}},
		"executionStats": bson.M{
			"nReturned":         int32(20),
			"totalKeysExamined": int32(0),
			"totalDocsExamined": int32(10000),
		},
	}

	row, err := planRow(plan)
	if err != nil {
		t.Fatal(err)
	}

	if rowField(row, "nReturned") != 20 || rowField(row, "totalDocsExamined") != 10000 {
		t.Errorf("некорректная строка плана: %v", row)
	}
}
