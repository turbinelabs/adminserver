package adminserver

//go:generate mockgen -source $GOFILE -destination mock_$GOFILE -package $GOPACKAGE

import (
	"flag"
	"fmt"
	"net"

	"github.com/turbinelabs/nonstdlib/proc"
)

const (
	DefaultListenIP   = "127.0.0.1"
	DefaultListenPort = 9000
)

type FromFlags interface {
	Validate() error
	Make(managedProc proc.ManagedProc) AdminServer
}

type fromFlags struct {
	ip   string
	port int
}

func NewFromFlags(flags *flag.FlagSet) FromFlags {
	ff := &fromFlags{}
	flags.StringVar(&ff.ip, "ip", DefaultListenIP, "What IP should we listen on")
	flags.IntVar(&ff.port, "port", DefaultListenPort, "What port should we listen on")
	return ff
}

func (ff *fromFlags) Validate() error {
	if net.ParseIP(ff.ip) == nil {
		return fmt.Errorf("invalid ip address: %s", ff.ip)
	}

	if ff.port <= 0 || ff.port > 65535 {
		return fmt.Errorf("invalid port: %d", ff.port)
	}

	return nil
}

func (ff *fromFlags) Make(managedProc proc.ManagedProc) AdminServer {
	return New(ff.ip, ff.port, managedProc)
}
