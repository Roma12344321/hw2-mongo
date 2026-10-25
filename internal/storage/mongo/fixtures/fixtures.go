package fixtures

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"hw2-mongo/internal/domain"
)

const DatasetVersion = "2026-10-05"

var (
	ReportFrom = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ReportTo   = time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
)

func Decimal(value string) bson.Decimal128 {
	d, err := bson.ParseDecimal128(value)
	if err != nil {
		panic(fmt.Sprintf("fixtures: bad decimal %q: %v", value, err))
	}

	return d
}

func product(sku, name, category, price string, stockL, stockM int32) domain.Product {
	return domain.Product{
		ID: sku, Name: name, Category: category, Price: Decimal(price),
		Variants: []domain.Variant{
			{Size: "L", Stock: stockL},
			{Size: "M", Stock: stockM},
		},
	}
}

func Products() []domain.Product {
	return []domain.Product{
		product("A", "Футболка", "clothes", "100.00", 0, 10),
		product("B", "Рубашка", "clothes", "80.00", 5, 4),
		product("C", "Толстовка", "clothes", "90.00", 3, 2),
		product("D", "Ремень", "accessories", "120.00", 4, 6),
		product("E", "Сумка", "accessories", "200.00", 2, 1),
		product("F", "Шарф", "accessories", "150.00", 1, 8),
	}
}

func item(sku, size string, qty int32, price string) domain.OrderItem {
	return domain.OrderItem{SKU: sku, Size: size, Qty: qty, Price: Decimal(price)}
}

func order(id string, createdAt time.Time, items []domain.OrderItem, status string, userID int32) domain.Order {
	return domain.Order{
		ID: id, UserID: userID, CreatedAt: createdAt, Status: status, Items: items,
	}
}

func Orders() []domain.Order {
	orders := []domain.Order{
		order("sale-start", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			[]domain.OrderItem{item("A", "M", 2, "100.00"), item("B", "L", 4, "80.00")},
			domain.OrderStatusPaid, 2500),
		order("sale-middle", time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC),
			[]domain.OrderItem{item("A", "L", 1, "120.00"), item("C", "M", 3, "90.00")},
			domain.OrderStatusPaid, 2500),
		order("sale-last", time.Date(2026, 1, 31, 23, 59, 59, 999000000, time.UTC),
			[]domain.OrderItem{
				item("D", "L", 2, "120.00"),
				item("E", "M", 1, "200.00"),
				item("F", "L", 1, "150.00"),
			},
			domain.OrderStatusPaid, 2500),
		order("excluded-end", time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			[]domain.OrderItem{item("A", "M", 999, "100.00")},
			domain.OrderStatusPaid, 2500),
		order("excluded-before", time.Date(2025, 12, 31, 23, 59, 59, 999000000, time.UTC),
			[]domain.OrderItem{item("A", "M", 999, "100.00")},
			domain.OrderStatusPaid, 2500),
		order("excluded-created", time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
			[]domain.OrderItem{item("A", "M", 999, "100.00")},
			domain.OrderStatusCreated, 2500),
		order("excluded-cancelled", time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
			[]domain.OrderItem{item("A", "M", 999, "100.00")},
			domain.OrderStatusCancelled, 2500),
	}

	for i := range 26 {
		createdAt := time.Date(2026, 3, 1, i/2, 0, 0, 0, time.UTC)
		orders = append(orders, order(
			fmt.Sprintf("H%02d", i), createdAt,
			[]domain.OrderItem{item("F", "M", 1, "150.00")},
			domain.OrderStatusPaid, 2500,
		))
	}

	orders = append(orders,
		order("other-user", time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			[]domain.OrderItem{item("F", "M", 1, "150.00")},
			domain.OrderStatusPaid, 2501),
		order("other-status", time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			[]domain.OrderItem{item("F", "M", 1, "150.00")},
			domain.OrderStatusCancelled, 2500),
	)

	return orders
}

func LargeOrders() []domain.Order {
	orders := make([]domain.Order, 0, 10000)
	base := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	for i := range 10000 {
		status := domain.OrderStatusCreated
		if i%4 < 2 {
			status = domain.OrderStatusPaid
		}

		userID := int32(3000 + i%71)
		if i%100 < 30 {
			userID = 2500
		}

		orders = append(orders, order(
			fmt.Sprintf("G%05d", i), base.Add(time.Duration(i/2)*time.Minute),
			[]domain.OrderItem{item("F", "M", 1, "150.00")},
			status, userID,
		))
	}

	return orders
}

func HistoryExpected() []string {
	expected := make([]string, 0, 20)
	for hour := 12; hour >= 3; hour-- {
		expected = append(expected, fmt.Sprintf("H%02d", hour*2), fmt.Sprintf("H%02d", hour*2+1))
	}

	return expected
}

type SearchCase struct {
	Category   string
	Size       string
	Qty        int32
	ExpectedID []string
}

func SearchCases() []SearchCase {
	return []SearchCase{
		{Category: "clothes", Size: "L", Qty: 2, ExpectedID: []string{"B", "C"}},
		{Category: "clothes", Size: "M", Qty: 5, ExpectedID: []string{"A"}},
		{Category: "accessories", Size: "L", Qty: 3, ExpectedID: []string{"D"}},
		{Category: "clothes", Size: "L", Qty: 99, ExpectedID: []string{}},
	}
}

type ReportExpectation struct {
	SKU     string
	Name    string
	Qty     int64
	Revenue string
}

func ReportExpected() []ReportExpectation {
	return []ReportExpectation{
		{SKU: "A", Name: "Футболка", Qty: 3, Revenue: "320.00"},
		{SKU: "B", Name: "Рубашка", Qty: 4, Revenue: "320.00"},
		{SKU: "C", Name: "Толстовка", Qty: 3, Revenue: "270.00"},
		{SKU: "D", Name: "Ремень", Qty: 2, Revenue: "240.00"},
		{SKU: "E", Name: "Сумка", Qty: 1, Revenue: "200.00"},
	}
}
