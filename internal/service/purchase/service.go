package purchase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"hw2-mongo/internal/domain"
	"hw2-mongo/internal/hwerror"
)

type ProductsStorage interface {
	ReserveStock(ctx context.Context, sku, size string, qty int32) (matched, modified int64, err error)
	FindByID(ctx context.Context, sku string) (domain.Product, error)
}

type OrdersStorage interface {
	FindByID(ctx context.Context, id string) (domain.Order, error)
	InsertOne(ctx context.Context, order domain.Order) error
}

type TxManager interface {
	Do(ctx context.Context, process func(ctx context.Context) error) error
}

type Service struct {
	logger    zerolog.Logger
	products  ProductsStorage
	orders    OrdersStorage
	txManager TxManager
}

func New(
	logger zerolog.Logger,
	products ProductsStorage,
	orders OrdersStorage,
	txManager TxManager,
) *Service {
	return &Service{
		logger:    logger.With().Str("component", "purchase.Service").Logger(),
		products:  products,
		orders:    orders,
		txManager: txManager,
	}
}

type UpdateTrace struct {
	Event    string `bson:"event"`
	SKU      string `bson:"sku"`
	Matched  int64  `bson:"matched"`
	Modified int64  `bson:"modified"`
}

const traceUpdateInsideTransaction = "update_inside_transaction"

func (s *Service) Purchase(
	ctx context.Context,
	request domain.PurchaseRequest,
	trace func(UpdateTrace),
) (domain.Order, error) {

	stored, err := s.orders.FindByID(ctx, request.OperationID)
	switch {
	case err == nil:
		if sameRequest(stored, request) {
			return stored, nil
		}

		return domain.Order{}, hwerror.New(hwerror.CodeRequestConflict, fmt.Sprintf(
			"operationId %s уже выполнен с другими параметрами", request.OperationID))
	case !errors.Is(err, mongo.ErrNoDocuments):
		return domain.Order{}, fmt.Errorf("orders.FindByID: %w", err)
	}

	var order domain.Order

	err = s.txManager.Do(ctx, func(txCtx context.Context) error {
		created, err := s.reserveAndCreate(txCtx, request, trace)
		if err != nil {
			return err
		}

		order = created

		return nil
	})
	if err != nil {
		return domain.Order{}, err
	}

	return order, nil
}

func (s *Service) reserveAndCreate(
	txCtx context.Context,
	request domain.PurchaseRequest,
	trace func(UpdateTrace),
) (domain.Order, error) {
	for _, item := range request.Items {
		matched, modified, err := s.products.ReserveStock(txCtx, item.SKU, item.Size, item.Qty)
		if err != nil {
			return domain.Order{}, err
		}

		trace(UpdateTrace{
			Event: traceUpdateInsideTransaction, SKU: item.SKU,
			Matched: matched, Modified: modified,
		})

		if matched == 0 || modified == 0 {
			return domain.Order{}, hwerror.New(hwerror.CodeItemUnavailable, fmt.Sprintf(
				"товар %s размер %s недоступен в количестве %d", item.SKU, item.Size, item.Qty))
		}
	}

	order := domain.Order{
		ID:     request.OperationID,
		UserID: request.UserID,

		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
		Status:    domain.OrderStatusCreated,
		Items:     make([]domain.OrderItem, 0, len(request.Items)),
	}

	for _, item := range request.Items {
		product, err := s.products.FindByID(txCtx, item.SKU)
		if err != nil {
			return domain.Order{}, fmt.Errorf("получение цены %s: %w", item.SKU, err)
		}

		order.Items = append(order.Items, domain.OrderItem{
			SKU: item.SKU, Size: item.Size, Qty: item.Qty, Price: product.Price,
		})
	}

	if err := s.orders.InsertOne(txCtx, order); err != nil {

		if isDuplicateKey(err) {
			return domain.Order{}, hwerror.New(hwerror.CodeRequestConflict, fmt.Sprintf(
				"operationId %s создан параллельным запросом", request.OperationID))
		}

		return domain.Order{}, fmt.Errorf("orders.InsertOne: %w", err)
	}

	return order, nil
}

func sameRequest(stored domain.Order, request domain.PurchaseRequest) bool {
	if stored.UserID != request.UserID || len(stored.Items) != len(request.Items) {
		return false
	}

	for i, item := range request.Items {
		saved := stored.Items[i]
		if saved.SKU != item.SKU || saved.Size != item.Size || saved.Qty != item.Qty {
			return false
		}
	}

	return true
}

func isDuplicateKey(err error) bool {

	var writeExValue mongo.WriteException
	if errors.As(err, &writeExValue) {
		return hasDuplicateCode(writeExValue.WriteErrors)
	}

	var writeExPtr *mongo.WriteException
	if errors.As(err, &writeExPtr) && writeExPtr != nil {
		return hasDuplicateCode(writeExPtr.WriteErrors)
	}

	var cmdErr *mongo.CommandError
	if errors.As(err, &cmdErr) {
		return cmdErr.Code == 11000
	}

	return false
}

func hasDuplicateCode(writeErrors []mongo.WriteError) bool {
	for _, writeErr := range writeErrors {
		if writeErr.Code == 11000 {
			return true
		}
	}

	return false
}

func RequestDoc(request domain.PurchaseRequest) bson.D {
	items := make(bson.A, 0, len(request.Items))
	for _, item := range request.Items {
		items = append(items, bson.D{
			{Key: "sku", Value: item.SKU},
			{Key: "size", Value: item.Size},
			{Key: "qty", Value: item.Qty},
		})
	}

	return bson.D{
		{Key: "operationId", Value: request.OperationID},
		{Key: "userId", Value: request.UserID},
		{Key: "items", Value: items},
	}
}
