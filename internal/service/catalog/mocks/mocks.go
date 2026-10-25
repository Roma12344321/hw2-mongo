package mocks

import (
	context "context"
	domain "hw2-mongo/internal/domain"
	reflect "reflect"

	bson "go.mongodb.org/mongo-driver/v2/bson"
	gomock "go.uber.org/mock/gomock"
)

type MockStorage struct {
	ctrl     *gomock.Controller
	recorder *MockStorageMockRecorder
	isgomock struct{}
}

type MockStorageMockRecorder struct {
	mock *MockStorage
}

func NewMockStorage(ctrl *gomock.Controller) *MockStorage {
	mock := &MockStorage{ctrl: ctrl}
	mock.recorder = &MockStorageMockRecorder{mock}
	return mock
}

func (m *MockStorage) EXPECT() *MockStorageMockRecorder {
	return m.recorder
}

func (m *MockStorage) ApplyValidator(ctx context.Context, validator any) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ApplyValidator", ctx, validator)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockStorageMockRecorder) ApplyValidator(ctx, validator any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ApplyValidator", reflect.TypeOf((*MockStorage)(nil).ApplyValidator), ctx, validator)
}

func (m *MockStorage) CollectionOptions(ctx context.Context) (bson.M, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CollectionOptions", ctx)
	ret0, _ := ret[0].(bson.M)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockStorageMockRecorder) CollectionOptions(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CollectionOptions", reflect.TypeOf((*MockStorage)(nil).CollectionOptions), ctx)
}

func (m *MockStorage) SearchByVariant(ctx context.Context, category, size string, qty int32) ([]domain.Product, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SearchByVariant", ctx, category, size, qty)
	ret0, _ := ret[0].([]domain.Product)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockStorageMockRecorder) SearchByVariant(ctx, category, size, qty any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SearchByVariant", reflect.TypeOf((*MockStorage)(nil).SearchByVariant), ctx, category, size, qty)
}
