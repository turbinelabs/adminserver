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
	tbnflag "github.com/turbinelabs/nonstdlib/flag"
	tbnnet "github.com/turbinelabs/nonstdlib/net"
	"github.com/turbinelabs/nonstdlib/proc"
)

const (
	DefaultListenAddr = "127.0.0.1:9000"
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
	addr string
}

// NewFromFlags installs the Flags necessary to configure an AdminServer into
// the provided flag.FlagSet, and returns a FromFlags.
func NewFromFlags(flags tbnflag.FlagSet) FromFlags {
	ff := &fromFlags{}
	flags.StringVar(
		&ff.addr,
		"admin.addr",
		DefaultListenAddr,
		"Specifies the `host:port` on which the admin server should listen.",
	)
	return ff
}

func (ff *fromFlags) Validate() error {
	if err := tbnnet.ValidateListenerAddr(ff.addr); err != nil {
		return err
	}

	return nil
}

func (ff *fromFlags) Make(managedProc proc.ManagedProc) AdminServer {
	return New(ff.addr, managedProc)
}
