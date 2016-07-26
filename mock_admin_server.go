// Automatically generated by MockGen. DO NOT EDIT!
// Source: admin_server.go

package adminserver

import (
	gomock "github.com/golang/mock/gomock"
)

// Mock of AdminServer interface
type MockAdminServer struct {
	ctrl     *gomock.Controller
	recorder *_MockAdminServerRecorder
}

// Recorder for MockAdminServer (not exported)
type _MockAdminServerRecorder struct {
	mock *MockAdminServer
}

func NewMockAdminServer(ctrl *gomock.Controller) *MockAdminServer {
	mock := &MockAdminServer{ctrl: ctrl}
	mock.recorder = &_MockAdminServerRecorder{mock}
	return mock
}

func (_m *MockAdminServer) EXPECT() *_MockAdminServerRecorder {
	return _m.recorder
}

func (_m *MockAdminServer) Start() error {
	ret := _m.ctrl.Call(_m, "Start")
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockAdminServerRecorder) Start() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Start")
}

func (_m *MockAdminServer) Close() error {
	ret := _m.ctrl.Call(_m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockAdminServerRecorder) Close() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Close")
}

func (_m *MockAdminServer) Listening() bool {
	ret := _m.ctrl.Call(_m, "Listening")
	ret0, _ := ret[0].(bool)
	return ret0
}

func (_mr *_MockAdminServerRecorder) Listening() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Listening")
}

func (_m *MockAdminServer) Addr() string {
	ret := _m.ctrl.Call(_m, "Addr")
	ret0, _ := ret[0].(string)
	return ret0
}

func (_mr *_MockAdminServerRecorder) Addr() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Addr")
}

func (_m *MockAdminServer) LastRequestedSignal() RequestedSignalType {
	ret := _m.ctrl.Call(_m, "LastRequestedSignal")
	ret0, _ := ret[0].(RequestedSignalType)
	return ret0
}

func (_mr *_MockAdminServerRecorder) LastRequestedSignal() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "LastRequestedSignal")
}