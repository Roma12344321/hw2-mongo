package mocks

import (
	context "context"
	domain "hw2-mongo/internal/domain"
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

type MockProductsStorage struct {
	ctrl     *gomock.Controller
	recorder *MockProductsStorageMockRecorder
	isgomock struct{}
}

type MockProductsStorageMockRecorder struct {
	mock *MockProductsStorage
}

func NewMockProductsStorage(ctrl *gomock.Controller) *MockProductsStorage {
	mock := &MockProductsStorage{ctrl: ctrl}
	mock.recorder = &MockProductsStorageMockRecorder{mock}
	return mock
}

func (m *MockProductsStorage) EXPECT() *MockProductsStorageMockRecorder {
	return m.recorder
}

func (m *MockProductsStorage) FindByID(ctx context.Context, sku string) (domain.Product, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindByID", ctx, sku)
	ret0, _ := ret[0].(domain.Product)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockProductsStorageMockRecorder) FindByID(ctx, sku any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindByID", reflect.TypeOf((*MockProductsStorage)(nil).FindByID), ctx, sku)
}

func (m *MockProductsStorage) ReserveStock(ctx context.Context, sku, size string, qty int32) (int64, int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReserveStock", ctx, sku, size, qty)
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(int64)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

func (mr *MockProductsStorageMockRecorder) ReserveStock(ctx, sku, size, qty any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReserveStock", reflect.TypeOf((*MockProductsStorage)(nil).ReserveStock), ctx, sku, size, qty)
}

type MockOrdersStorage struct {
	ctrl     *gomock.Controller
	recorder *MockOrdersStorageMockRecorder
	isgomock struct{}
}

type MockOrdersStorageMockRecorder struct {
	mock *MockOrdersStorage
}

func NewMockOrdersStorage(ctrl *gomock.Controller) *MockOrdersStorage {
	mock := &MockOrdersStorage{ctrl: ctrl}
	mock.recorder = &MockOrdersStorageMockRecorder{mock}
	return mock
}

func (m *MockOrdersStorage) EXPECT() *MockOrdersStorageMockRecorder {
	return m.recorder
}

func (m *MockOrdersStorage) FindByID(ctx context.Context, id string) (domain.Order, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindByID", ctx, id)
	ret0, _ := ret[0].(domain.Order)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockOrdersStorageMockRecorder) FindByID(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindByID", reflect.TypeOf((*MockOrdersStorage)(nil).FindByID), ctx, id)
}

func (m *MockOrdersStorage) InsertOne(ctx context.Context, order domain.Order) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InsertOne", ctx, order)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockOrdersStorageMockRecorder) InsertOne(ctx, order any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InsertOne", reflect.TypeOf((*MockOrdersStorage)(nil).InsertOne), ctx, order)
}

type MockTxManager struct {
	ctrl     *gomock.Controller
	recorder *MockTxManagerMockRecorder
	isgomock struct{}
}

type MockTxManagerMockRecorder struct {
	mock *MockTxManager
}

func NewMockTxManager(ctrl *gomock.Controller) *MockTxManager {
	mock := &MockTxManager{ctrl: ctrl}
	mock.recorder = &MockTxManagerMockRecorder{mock}
	return mock
}

func (m *MockTxManager) EXPECT() *MockTxManagerMockRecorder {
	return m.recorder
}

func (m *MockTxManager) Do(ctx context.Context, process func(context.Context) error) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Do", ctx, process)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockTxManagerMockRecorder) Do(ctx, process any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Do", reflect.TypeOf((*MockTxManager)(nil).Do), ctx, process)
}
