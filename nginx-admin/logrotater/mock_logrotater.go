// Automatically generated by MockGen. DO NOT EDIT!
// Source: logrotater.go

package logrotater

import (
	gomock "github.com/golang/mock/gomock"
	log "log"
)

// Mock of FromFlags interface
type MockFromFlags struct {
	ctrl     *gomock.Controller
	recorder *_MockFromFlagsRecorder
}

// Recorder for MockFromFlags (not exported)
type _MockFromFlagsRecorder struct {
	mock *MockFromFlags
}

func NewMockFromFlags(ctrl *gomock.Controller) *MockFromFlags {
	mock := &MockFromFlags{ctrl: ctrl}
	mock.recorder = &_MockFromFlagsRecorder{mock}
	return mock
}

func (_m *MockFromFlags) EXPECT() *_MockFromFlagsRecorder {
	return _m.recorder
}

func (_m *MockFromFlags) Validate() error {
	ret := _m.ctrl.Call(_m, "Validate")
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockFromFlagsRecorder) Validate() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Validate")
}

func (_m *MockFromFlags) Make(_param0 *log.Logger, _param1 ReopenLogsFunc) LogRotater {
	ret := _m.ctrl.Call(_m, "Make", _param0, _param1)
	ret0, _ := ret[0].(LogRotater)
	return ret0
}

func (_mr *_MockFromFlagsRecorder) Make(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Make", arg0, arg1)
}

// Mock of LogRotater interface
type MockLogRotater struct {
	ctrl     *gomock.Controller
	recorder *_MockLogRotaterRecorder
}

// Recorder for MockLogRotater (not exported)
type _MockLogRotaterRecorder struct {
	mock *MockLogRotater
}

func NewMockLogRotater(ctrl *gomock.Controller) *MockLogRotater {
	mock := &MockLogRotater{ctrl: ctrl}
	mock.recorder = &_MockLogRotaterRecorder{mock}
	return mock
}

func (_m *MockLogRotater) EXPECT() *_MockLogRotaterRecorder {
	return _m.recorder
}

func (_m *MockLogRotater) Start(pathname string) error {
	ret := _m.ctrl.Call(_m, "Start", pathname)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockLogRotaterRecorder) Start(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Start", arg0)
}

func (_m *MockLogRotater) StopAll() {
	_m.ctrl.Call(_m, "StopAll")
}

func (_mr *_MockLogRotaterRecorder) StopAll() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "StopAll")
}