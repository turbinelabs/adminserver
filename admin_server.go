package adminserver

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/turbinelabs/proc"
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
}

// Creates a new AdminServer on the given IP address and port,
// wrapping the given ManagedProc.
func New(ip string, port int, managedProc proc.ManagedProc) AdminServer {
	hostPort := fmt.Sprintf("%s:%d", ip, port)

	adminServer := &adminServer{
		lastRequestedSignal: NoRequestedSignal,
		managedProc:         managedProc,
	}

	router := mux.NewRouter()
	router.StrictSlash(true)

	router.Methods("GET").Path("/admin/kill").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			adminServer.kill(w, r)
		})

	router.Methods("GET").Path("/admin/quit").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			adminServer.quit(w, r)
		})

	router.Methods("GET").Path("/admin/reload").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			adminServer.reload(w, r)
		})

	adminServer.server = &http.Server{
		Addr:         hostPort,
		Handler:      router,
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
	l, err := net.Listen("tcp", adminServer.server.Addr)
	if err != nil {
		return err
	}

	adminServer.listener = l
	return adminServer.server.Serve(l)
}

func (adminServer *adminServer) Close() error {
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
		w.Write([]byte(fmt.Sprintf("FAILED: %s\n", err.Error())))
	} else {
		w.Write([]byte("OK\n"))
	}
}

func (adminServer *adminServer) quit(w http.ResponseWriter, request *http.Request) {
	adminServer.lastRequestedSignal = RequestedQuitSignal
	w.Header().Set("Content-Type", "text/plain")
	if err := adminServer.managedProc.Quit(); err != nil {
		w.Write([]byte(fmt.Sprintf("FAILED: %s\n", err.Error())))
	} else {
		w.Write([]byte("OK\n"))
	}
}

func (adminServer *adminServer) reload(w http.ResponseWriter, request *http.Request) {
	adminServer.lastRequestedSignal = RequestedHangupSignal
	w.Header().Set("Content-Type", "text/plain")
	if err := adminServer.managedProc.Hangup(); err != nil {
		w.Write([]byte(fmt.Sprintf("FAILED: %s\n", err.Error())))
	} else {
		w.Write([]byte("OK\n"))
	}
}
