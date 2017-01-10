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
	"flag"
	"fmt"
	"net"

	"github.com/turbinelabs/nonstdlib/proc"
)

const (
	DefaultListenIP   = "127.0.0.1"
	DefaultListenPort = 9000
)

// The FromFlags interface models creation of an AdminServer from a
// flag.FlagSet, and validation of the relevant Flags in that FlagSet.
type FromFlags interface {
	// Validate ensures that the Flags are properly specified.
	Validate() error

	// Make produces an AdminServer from the configured Flags.
	Make(managedProc proc.ManagedProc) AdminServer
}

type fromFlags struct {
	ip   string
	port int
}

// NewFromFlags installs the Flags necessary to configure an AdminServer into
// the provided flag.FlagSet, and returns a FromFlags.
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
