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

func (m *MockStorage) Count(ctx context.Context) (int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Count", ctx)
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockStorageMockRecorder) Count(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Count", reflect.TypeOf((*MockStorage)(nil).Count), ctx)
}

func (m *MockStorage) CreateHistoryIndex(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateHistoryIndex", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockStorageMockRecorder) CreateHistoryIndex(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateHistoryIndex", reflect.TypeOf((*MockStorage)(nil).CreateHistoryIndex), ctx)
}

func (m *MockStorage) ExplainHistory(ctx context.Context, in domain.HistoryInput) (bson.M, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ExplainHistory", ctx, in)
	ret0, _ := ret[0].(bson.M)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockStorageMockRecorder) ExplainHistory(ctx, in any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ExplainHistory", reflect.TypeOf((*MockStorage)(nil).ExplainHistory), ctx, in)
}

func (m *MockStorage) FindHistory(ctx context.Context, in domain.HistoryInput) ([]domain.Order, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindHistory", ctx, in)
	ret0, _ := ret[0].([]domain.Order)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockStorageMockRecorder) FindHistory(ctx, in any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindHistory", reflect.TypeOf((*MockStorage)(nil).FindHistory), ctx, in)
}

func (m *MockStorage) IndexNames(ctx context.Context) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IndexNames", ctx)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockStorageMockRecorder) IndexNames(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IndexNames", reflect.TypeOf((*MockStorage)(nil).IndexNames), ctx)
}

func (m *MockStorage) InsertMany(ctx context.Context, orders []domain.Order) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InsertMany", ctx, orders)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockStorageMockRecorder) InsertMany(ctx, orders any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InsertMany", reflect.TypeOf((*MockStorage)(nil).InsertMany), ctx, orders)
}
