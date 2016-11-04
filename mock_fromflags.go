// Automatically generated by MockGen. DO NOT EDIT!
// Source: fromflags.go

package adminserver

import (
	gomock "github.com/golang/mock/gomock"
	proc "github.com/turbinelabs/stdlib/proc"
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

func (_m *MockFromFlags) Make(managedProc proc.ManagedProc) AdminServer {
	ret := _m.ctrl.Call(_m, "Make", managedProc)
	ret0, _ := ret[0].(AdminServer)
	return ret0
}

func (_mr *_MockFromFlagsRecorder) Make(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Make", arg0)
}
