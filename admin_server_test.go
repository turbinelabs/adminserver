package adminserver

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	"github.com/turbinelabs/proc"
	"github.com/turbinelabs/test/assert"
)

func TestNew(t *testing.T) {
	ctrl := gomock.NewController(assert.Tracing(t))
	mockProc := proc.NewMockManagedProc(ctrl)

	asIface := New("localhost", 1000, mockProc)

	as := asIface.(*adminServer)

	assert.SameInstance(t, as.managedProc, mockProc)
	assert.NonNil(t, as.server)
	assert.Equal(t, as.server.Addr, "localhost:1000")
	assert.False(t, as.Listening())
	assert.Equal(t, as.Addr(), "")
	assert.Nil(t, as.Close())
	assert.Equal(t, as.LastRequestedSignal(), NoRequestedSignal)

	ctrl.Finish()
}

func mkAdminServer(t *testing.T) (AdminServer, *proc.MockManagedProc, func()) {
	ctrl := gomock.NewController(assert.Tracing(t))
	mockProc := proc.NewMockManagedProc(ctrl)

	as := New("localhost", 0, mockProc)
	go func() {
		err := as.Start()
		assert.Nil(t, err)
	}()

	for !as.Listening() {
		time.Sleep(10 * time.Millisecond)
	}

	cleanup := func() {
		as.Close()
		ctrl.Finish()

		assert.False(t, as.Listening())
	}

	return as, mockProc, cleanup
}

func TestAdminServerKill(t *testing.T) {
	as, mockProc, cleanup := mkAdminServer(t)

	mockProc.EXPECT().Kill().Return(nil)

	resp, err := http.Get(fmt.Sprintf("http://%s/admin/kill", as.Addr()))
	assert.Nil(t, err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	assert.Equal(t, string(body), "OK\n")
	assert.Equal(t, as.LastRequestedSignal(), RequestedKillSignal)

	cleanup()
}

func TestAdminServerKillError(t *testing.T) {
	as, mockProc, cleanup := mkAdminServer(t)

	mockProc.EXPECT().Kill().Return(errors.New("oops"))

	resp, err := http.Get(fmt.Sprintf("http://%s/admin/kill", as.Addr()))
	assert.Nil(t, err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	assert.Equal(t, string(body), "FAILED: oops\n")
	assert.Equal(t, as.LastRequestedSignal(), RequestedKillSignal)

	cleanup()
}

func TestAdminServerQuit(t *testing.T) {
	as, mockProc, cleanup := mkAdminServer(t)

	mockProc.EXPECT().Quit().Return(nil)

	resp, err := http.Get(fmt.Sprintf("http://%s/admin/quit", as.Addr()))
	assert.Nil(t, err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	assert.Equal(t, string(body), "OK\n")
	assert.Equal(t, as.LastRequestedSignal(), RequestedQuitSignal)

	cleanup()
}

func TestAdminServerQuitError(t *testing.T) {
	as, mockProc, cleanup := mkAdminServer(t)

	mockProc.EXPECT().Quit().Return(errors.New("oops"))

	resp, err := http.Get(fmt.Sprintf("http://%s/admin/quit", as.Addr()))
	assert.Nil(t, err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	assert.Equal(t, string(body), "FAILED: oops\n")
	assert.Equal(t, as.LastRequestedSignal(), RequestedQuitSignal)

	cleanup()
}

func TestAdminServerReload(t *testing.T) {
	as, mockProc, cleanup := mkAdminServer(t)

	mockProc.EXPECT().Hangup().Return(nil)

	resp, err := http.Get(fmt.Sprintf("http://%s/admin/reload", as.Addr()))
	assert.Nil(t, err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	assert.Equal(t, string(body), "OK\n")
	assert.Equal(t, as.LastRequestedSignal(), RequestedHangupSignal)

	cleanup()
}

func TestAdminServerReloadError(t *testing.T) {
	as, mockProc, cleanup := mkAdminServer(t)

	mockProc.EXPECT().Hangup().Return(errors.New("oops"))

	resp, err := http.Get(fmt.Sprintf("http://%s/admin/reload", as.Addr()))
	assert.Nil(t, err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	assert.Equal(t, string(body), "FAILED: oops\n")
	assert.Equal(t, as.LastRequestedSignal(), RequestedHangupSignal)

	cleanup()
}

func TestAdminServerCannotListen(t *testing.T) {
	// grab some port and listen on it
	l, err := net.Listen("tcp", "localhost:0")
	assert.Nil(t, err)
	assert.NonNil(t, l)
	go func() {
		l.Accept()
	}()
	defer l.Close()

	port := l.Addr().(*net.TCPAddr).Port

	ctrl := gomock.NewController(assert.Tracing(t))
	mockProc := proc.NewMockManagedProc(ctrl)

	as := New("localhost", port, mockProc)
	err = as.Start()
	assert.NonNil(t, err)

	ctrl.Finish()
}
