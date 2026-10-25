package purchase_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/rs/zerolog"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/mock/gomock"

	"hw2-mongo/internal/domain"
	"hw2-mongo/internal/hwerror"
	"hw2-mongo/internal/service/purchase"
	"hw2-mongo/internal/service/purchase/mocks"
	"hw2-mongo/internal/storage/mongo/fixtures"
)

func newTestService(t *testing.T) (
	*purchase.Service,
	*mocks.MockProductsStorage,
	*mocks.MockOrdersStorage,
	*txRecorder,
) {
	t.Helper()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	products := mocks.NewMockProductsStorage(ctrl)
	orders := mocks.NewMockOrdersStorage(ctrl)
	tx := &txRecorder{}

	service := purchase.New(zerolog.Nop(), products, orders, tx)

	return service, products, orders, tx
}

type txRecorder struct {
	Called  bool
	Process func(ctx context.Context) error
}

func (tx *txRecorder) Do(ctx context.Context, process func(ctx context.Context) error) error {
	tx.Called = true
	tx.Process = process

	return process(ctx)
}

func sampleRequest() domain.PurchaseRequest {
	return domain.PurchaseRequest{
		OperationID: "purchase-001",
		UserID:      2500,
		Items: []domain.PurchaseItem{
			{SKU: "A", Size: "M", Qty: 2},
			{SKU: "B", Size: "L", Qty: 1},
		},
	}
}

func storedOrder(request domain.PurchaseRequest) domain.Order {
	order := domain.Order{
		ID: request.OperationID, UserID: request.UserID,
		CreatedAt: fixtures.ReportFrom, Status: domain.OrderStatusCreated,
		Items: []domain.OrderItem{
			{SKU: "A", Size: "M", Qty: 2, Price: fixtures.Decimal("100.00")},
			{SKU: "B", Size: "L", Qty: 1, Price: fixtures.Decimal("80.00")},
		},
	}

	return order
}

func TestPurchaseReplayReturnsStoredOrder(t *testing.T) {
	service, products, orders, tx := newTestService(t)
	request := sampleRequest()
	stored := storedOrder(request)

	orders.EXPECT().FindByID(gomock.Any(), request.OperationID).Return(stored, nil)
	products.EXPECT().ReserveStock(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Times(0)

	got, err := service.Purchase(context.Background(), request, func(purchase.UpdateTrace) {})

	if err != nil {
		t.Fatalf("replay: unexpected error %v", err)
	}
	if tx.Called {
		t.Error("replay не должен открывать транзакцию")
	}
	if got.ID != stored.ID || got.CreatedAt != stored.CreatedAt {
		t.Errorf("replay вернул не сохранённый заказ: %+v", got)
	}
}

func TestPurchaseConflictOnDifferentParams(t *testing.T) {
	service, _, orders, tx := newTestService(t)
	request := sampleRequest()
	stored := storedOrder(request)
	stored.Items[0].Qty = 9

	orders.EXPECT().FindByID(gomock.Any(), request.OperationID).Return(stored, nil)

	_, err := service.Purchase(context.Background(), request, func(purchase.UpdateTrace) {})

	if hwerror.Code(err) != hwerror.CodeRequestConflict {
		t.Fatalf("ожидался REQUEST_CONFLICT, получено %v", err)
	}
	if tx.Called {
		t.Error("конфликт не должен открывать транзакцию")
	}
}

func TestPurchaseItemUnavailableAborts(t *testing.T) {
	service, products, orders, tx := newTestService(t)
	request := sampleRequest()
	request.OperationID = "rollback-001"
	request.Items[1].Qty = 6

	orders.EXPECT().
		FindByID(gomock.Any(), request.OperationID).
		Return(domain.Order{}, fmtNoDocuments())

	products.EXPECT().
		ReserveStock(gomock.Any(), "A", "M", int32(2)).
		Return(int64(1), int64(1), nil)
	products.EXPECT().
		ReserveStock(gomock.Any(), "B", "L", int32(6)).
		Return(int64(0), int64(0), nil)

	var traces []purchase.UpdateTrace
	_, err := service.Purchase(context.Background(), request, func(event purchase.UpdateTrace) {
		traces = append(traces, event)
	})

	if hwerror.Code(err) != hwerror.CodeItemUnavailable {
		t.Fatalf("ожидался ITEM_UNAVAILABLE, получено %v", err)
	}
	if !tx.Called {
		t.Error("покупка должна выполниться в транзакции")
	}
	if len(traces) != 2 || traces[0].SKU != "A" || traces[0].Modified != 1 {
		t.Errorf("трейс должен показать успешное первое списание: %+v", traces)
	}
}

func TestPurchaseSuccessCreatesOrder(t *testing.T) {
	service, products, orders, _ := newTestService(t)
	request := sampleRequest()

	orders.EXPECT().
		FindByID(gomock.Any(), request.OperationID).
		Return(domain.Order{}, fmtNoDocuments())
	products.EXPECT().
		ReserveStock(gomock.Any(), "A", "M", int32(2)).
		Return(int64(1), int64(1), nil)
	products.EXPECT().
		ReserveStock(gomock.Any(), "B", "L", int32(1)).
		Return(int64(1), int64(1), nil)

	productsFromFixtures := fixtures.Products()
	products.EXPECT().FindByID(gomock.Any(), "A").Return(productsFromFixtures[0], nil)
	products.EXPECT().FindByID(gomock.Any(), "B").Return(productsFromFixtures[1], nil)

	orders.EXPECT().InsertOne(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, order domain.Order) error {
			if order.ID != request.OperationID || order.Status != domain.OrderStatusCreated {
				t.Errorf("неверный сохраняемый заказ: %+v", order)
			}

			return nil
		})

	got, err := service.Purchase(context.Background(), request, func(purchase.UpdateTrace) {})
	if err != nil {
		t.Fatalf("success: unexpected error %v", err)
	}

	if len(got.Items) != 2 || got.Items[0].SKU != "A" || got.Items[1].SKU != "B" {
		t.Errorf("позиции должны сохранить порядок входа: %+v", got.Items)
	}
	if got.Items[0].Price.String() != "100.00" || got.Items[1].Price.String() != "80.00" {
		t.Errorf("цены должны зафиксироваться из каталога: %+v", got.Items)
	}
}

func TestPurchaseReplayIgnoresSavedPrices(t *testing.T) {
	service, _, orders, _ := newTestService(t)
	request := sampleRequest()
	stored := storedOrder(request)
	stored.Items[0].Price = fixtures.Decimal("777.00")

	orders.EXPECT().FindByID(gomock.Any(), request.OperationID).Return(stored, nil)

	got, err := service.Purchase(context.Background(), request, func(purchase.UpdateTrace) {})
	if err != nil {
		t.Fatalf("replay: unexpected error %v", err)
	}
	if got.Items[0].Price.String() != "777.00" {
		t.Error("replay должен вернуть сохранённый заказ как есть")
	}
}

func fmtNoDocuments() error {
	return fmt.Errorf("orders.FindOne %s: %w", "purchase-001", mongo.ErrNoDocuments)
}
