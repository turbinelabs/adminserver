/*
Copyright 2017 Turbine Labs, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package adminserver

//go:generate mockgen -source $GOFILE -destination mock_$GOFILE -package $GOPACKAGE

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/turbinelabs/nonstdlib/proc"
)

type RequestedSignalType int

const (
	NoRequestedSignal RequestedSignalType = iota
	RequestedKillSignal
	RequestedQuitSignal
	RequestedHangupSignal
)

// AdminServer is an HTTP server that wraps a process. The admin
// server can send signals to the process. The server terminates when
// the wrapped process terminates. The server responds to the following
// URI paths:
//     /admin/reload
//     /admin/quit
//     /admin/kill
type AdminServer interface {
	// Starts the HTTP server.
	Start() error

	// Stops the HTTP server.
	Close() error

	// If true, the HTTP server is up and listening for connections.
	Listening() bool

	// The host:port the HTTP server is listening on.
	Addr() string

	// The last signal sent to the process, if any.
	LastRequestedSignal() RequestedSignalType
}

type adminServer struct {
	lastRequestedSignal RequestedSignalType
	managedProc         proc.ManagedProc
	listener            net.Listener
	server              *http.Server
	closeMutex          sync.Mutex
}

// Creates a new AdminServer on the given IP address and port,
// wrapping the given ManagedProc.
func New(ip string, port int, managedProc proc.ManagedProc) AdminServer {
	hostPort := fmt.Sprintf("%s:%d", ip, port)

	adminServer := &adminServer{
		lastRequestedSignal: NoRequestedSignal,
		managedProc:         managedProc,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.Write([]byte("404\n"))
		}

		switch r.URL.String() {
		case "/admin/kill":
			adminServer.kill(w, r)
		case "/admin/quit":
			adminServer.quit(w, r)
		case "/admin/reload":
			adminServer.reload(w, r)
		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("NOT FOUND\n"))
		}
	})

	adminServer.server = &http.Server{
		Addr:         hostPort,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return adminServer
}

func (adminServer *adminServer) Addr() string {
	if !adminServer.Listening() {
		return ""
	}

	return adminServer.listener.Addr().String()
}

func (adminServer *adminServer) Listening() bool {
	return adminServer.listener != nil
}

func (adminServer *adminServer) Start() error {
	// As much as possible, prevent Close() from occurring while
	// adminServer is starting.
	adminServer.closeMutex.Lock()

	// Unlock the mutex once: either at return or just before the
	// blocking call to Serve.
	var unlockOnce sync.Once
	defer func() {
		unlockOnce.Do(adminServer.closeMutex.Unlock)
	}()

	if adminServer.server == nil {
		return errors.New("already closed")
	}

	l, err := net.Listen("tcp", adminServer.server.Addr)
	if err != nil {
		return err
	}

	adminServer.listener = l

	unlockOnce.Do(adminServer.closeMutex.Unlock)
	return adminServer.server.Serve(l)
}

func (adminServer *adminServer) Close() error {
	adminServer.closeMutex.Lock()
	defer func() {
		adminServer.server = nil
		adminServer.closeMutex.Unlock()
	}()

	l := adminServer.listener
	if l == nil {
		return nil
	}

	adminServer.listener = nil
	return l.Close()
}

func (adminServer *adminServer) LastRequestedSignal() RequestedSignalType {
	return adminServer.lastRequestedSignal
}

func (adminServer *adminServer) kill(w http.ResponseWriter, request *http.Request) {
	adminServer.lastRequestedSignal = RequestedKillSignal
	w.Header().Set("Content-Type", "text/plain")
	if err := adminServer.managedProc.Kill(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("FAILED: %s\n", err.Error())))
	} else {
		w.Write([]byte("OK\n"))
	}
}

func (adminServer *adminServer) quit(w http.ResponseWriter, request *http.Request) {
	adminServer.lastRequestedSignal = RequestedQuitSignal
	w.Header().Set("Content-Type", "text/plain")
	if err := adminServer.managedProc.Quit(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("FAILED: %s\n", err.Error())))
	} else {
		w.Write([]byte("OK\n"))
	}
}

func (adminServer *adminServer) reload(w http.ResponseWriter, request *http.Request) {
	adminServer.lastRequestedSignal = RequestedHangupSignal
	w.Header().Set("Content-Type", "text/plain")
	if err := adminServer.managedProc.Hangup(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("FAILED: %s\n", err.Error())))
	} else {
		w.Write([]byte("OK\n"))
	}
}
