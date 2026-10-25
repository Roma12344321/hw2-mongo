package mocks

import (
	context "context"
	domain "hw2-mongo/internal/domain"
	reflect "reflect"
	time "time"

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

func (m *MockStorage) AggregateReport(ctx context.Context, from, to time.Time) ([]domain.ReportRow, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AggregateReport", ctx, from, to)
	ret0, _ := ret[0].([]domain.ReportRow)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockStorageMockRecorder) AggregateReport(ctx, from, to any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AggregateReport", reflect.TypeOf((*MockStorage)(nil).AggregateReport), ctx, from, to)
}
