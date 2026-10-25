package lab

import (
	"encoding/json"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"hw2-mongo/internal/domain"
)

func normalized(value any) (string, error) {
	tree, err := toTree(value)
	if err != nil {
		return "", fmt.Errorf("toTree: %w", err)
	}

	data, err := json.Marshal(tree)
	if err != nil {
		return "", fmt.Errorf("json.Marshal: %w", err)
	}

	return string(data), nil
}

func toTree(value any) (any, error) {
	raw, err := bson.Marshal(bson.D{{Key: "$wrap", Value: value}})
	if err != nil {
		return nil, fmt.Errorf("bson.Marshal: %w", err)
	}

	var wrapper bson.D
	if err := bson.Unmarshal(raw, &wrapper); err != nil {
		return nil, fmt.Errorf("bson.Unmarshal: %w", err)
	}

	return normalizeValue(wrapper[0].Value), nil
}

func normalizeValue(value any) any {
	switch typed := value.(type) {
	case bson.D:
		normalizedMap := make(map[string]any, len(typed))
		for _, element := range typed {
			normalizedMap[element.Key] = normalizeValue(element.Value)
		}

		return normalizedMap
	case bson.A:
		return normalizeSlice(typed)
	case bson.M:
		normalizedMap := make(map[string]any, len(typed))
		for key, item := range typed {
			normalizedMap[key] = normalizeValue(item)
		}

		return normalizedMap
	case map[string]any:
		normalizedMap := make(map[string]any, len(typed))
		for key, item := range typed {
			normalizedMap[key] = normalizeValue(item)
		}

		return normalizedMap
	case []any:
		return normalizeSlice(typed)
	case bson.Decimal128:
		return map[string]any{"$numberDecimal": typed.String()}
	case bson.DateTime:
		return map[string]any{"$date": typed.Time().UTC().Format(time.RFC3339Nano)}
	case time.Time:
		return map[string]any{"$date": typed.UTC().Format(time.RFC3339Nano)}
	default:
		return value
	}
}

func normalizeSlice(items []any) []any {
	normalized := make([]any, 0, len(items))
	for _, item := range items {
		normalized = append(normalized, normalizeValue(item))
	}

	return normalized
}

func hasStage(plan any, stage string) bool {
	switch typed := plan.(type) {
	case bson.D:
		for _, element := range typed {
			if element.Key == "stage" && element.Value == stage {
				return true
			}

			if hasStage(element.Value, stage) {
				return true
			}
		}
	case bson.M:
		for key, value := range typed {
			if key == "stage" && value == stage {
				return true
			}

			if hasStage(value, stage) {
				return true
			}
		}
	case map[string]any:
		for key, value := range typed {
			if key == "stage" && value == stage {
				return true
			}

			if hasStage(value, stage) {
				return true
			}
		}
	case bson.A:
		for _, item := range typed {
			if hasStage(item, stage) {
				return true
			}
		}
	case []any:
		for _, item := range typed {
			if hasStage(item, stage) {
				return true
			}
		}
	}

	return false
}

func toInt64(value any) (int64, bool) {
	switch typed := value.(type) {
	case int32:
		return int64(typed), true
	case int64:
		return typed, true
	case int:
		return int64(typed), true
	case float64:
		return int64(typed), true
	default:
		return 0, false
	}
}

func asDocument(value any) map[string]any {
	switch typed := value.(type) {
	case bson.M:
		return typed
	case bson.D:
		document := make(map[string]any, len(typed))
		for _, element := range typed {
			document[element.Key] = element.Value
		}

		return document
	default:
		return nil
	}
}

func planRow(plan bson.M) (bson.D, error) {
	stats := asDocument(plan["executionStats"])
	if stats == nil {
		return nil, fmt.Errorf("план без executionStats")
	}

	nReturned, ok := toInt64(stats["nReturned"])
	if !ok {
		return nil, fmt.Errorf("nReturned не число: %T", stats["nReturned"])
	}

	keysExamined, _ := toInt64(stats["totalKeysExamined"])
	docsExamined, _ := toInt64(stats["totalDocsExamined"])

	return bson.D{
		{Key: "nReturned", Value: nReturned},
		{Key: "totalKeysExamined", Value: keysExamined},
		{Key: "totalDocsExamined", Value: docsExamined},
		{Key: "hasSORT", Value: hasStage(plan["queryPlanner"], "SORT")},
	}, nil
}

func ProductIDs(products []domain.Product) []string {
	ids := make([]string, 0, len(products))
	for _, product := range products {
		ids = append(ids, product.ID)
	}

	return ids
}

func OrderIDs(orders []domain.Order) []string {
	ids := make([]string, 0, len(orders))
	for _, order := range orders {
		ids = append(ids, order.ID)
	}

	return ids
}

func productFields(product domain.Product) bson.D {
	variants := make(bson.A, 0, len(product.Variants))
	for _, variant := range product.Variants {
		variants = append(variants, bson.D{
			{Key: "size", Value: variant.Size},
			{Key: "stock", Value: variant.Stock},
		})
	}

	return bson.D{
		{Key: "_id", Value: product.ID},
		{Key: "name", Value: product.Name},
		{Key: "category", Value: product.Category},
		{Key: "price", Value: product.Price},
		{Key: "variants", Value: variants},
	}
}

func ProductFieldsList(products []domain.Product) bson.A {
	fields := make(bson.A, 0, len(products))
	for _, product := range products {
		fields = append(fields, productFields(product))
	}

	return fields
}

func orderFields(order domain.Order) bson.D {
	items := make(bson.A, 0, len(order.Items))
	for _, item := range order.Items {
		items = append(items, bson.D{
			{Key: "sku", Value: item.SKU},
			{Key: "size", Value: item.Size},
			{Key: "qty", Value: item.Qty},
			{Key: "price", Value: item.Price},
		})
	}

	return bson.D{
		{Key: "_id", Value: order.ID},
		{Key: "userId", Value: order.UserID},
		{Key: "createdAt", Value: order.CreatedAt},
		{Key: "status", Value: order.Status},
		{Key: "items", Value: items},
	}
}

func OrderFieldsList(orders []domain.Order) bson.A {
	fields := make(bson.A, 0, len(orders))
	for _, order := range orders {
		fields = append(fields, orderFields(order))
	}

	return fields
}

func reportRows(rows []domain.ReportRow) bson.A {
	out := make(bson.A, 0, len(rows))
	for _, row := range rows {
		out = append(out, bson.D{
			{Key: "sku", Value: row.SKU},
			{Key: "name", Value: row.Name},
			{Key: "qty", Value: row.Qty},
			{Key: "revenue", Value: row.Revenue.String()},
		})
	}

	return out
}

func toBsonD(value any) bson.D {
	raw, err := bson.Marshal(value)
	if err != nil {
		return bson.D{}
	}

	var doc bson.D
	if err := bson.Unmarshal(raw, &doc); err != nil {
		return bson.D{}
	}

	return doc
}
