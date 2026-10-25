package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	OrderStatusCreated   = "created"
	OrderStatusPaid      = "paid"
	OrderStatusCancelled = "cancelled"
)

type Variant struct {
	Size  string `bson:"size"`
	Stock int32  `bson:"stock"`
}

type Product struct {
	ID       string          `bson:"_id"`
	Name     string          `bson:"name"`
	Category string          `bson:"category"`
	Price    bson.Decimal128 `bson:"price"`
	Variants []Variant       `bson:"variants"`
}

type OrderItem struct {
	SKU   string          `bson:"sku"`
	Size  string          `bson:"size"`
	Qty   int32           `bson:"qty"`
	Price bson.Decimal128 `bson:"price"`
}

type Order struct {
	ID        string      `bson:"_id"`
	UserID    int32       `bson:"userId"`
	CreatedAt time.Time   `bson:"createdAt"`
	Status    string      `bson:"status"`
	Items     []OrderItem `bson:"items"`
}

type PurchaseItem struct {
	SKU  string
	Size string
	Qty  int32
}

type PurchaseRequest struct {
	OperationID string
	UserID      int32
	Items       []PurchaseItem
}

type SearchInput struct {
	Category string
	Size     string
	Qty      int32
}

type HistoryInput struct {
	UserID int32
	Status string
}

type ReportRow struct {
	SKU     string          `bson:"sku"`
	Name    string          `bson:"name"`
	Qty     int64           `bson:"qty"`
	Revenue bson.Decimal128 `bson:"revenue"`
}
