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

import (
	"testing"

	"github.com/golang/mock/gomock"

	tbnflag "github.com/turbinelabs/nonstdlib/flag"
	"github.com/turbinelabs/nonstdlib/proc"
	"github.com/turbinelabs/test/assert"
)

func TestFromFlags(t *testing.T) {
	flagset := tbnflag.NewTestFlagSet()
	ff := NewFromFlags(flagset)
	ffImpl := ff.(*fromFlags)
	assert.Equal(t, ffImpl.addr.Addr(), DefaultListenAddr)

	flagset.Parse([]string{"-admin.addr=4.5.6.7:9999"})

	assert.Equal(t, ffImpl.addr.Addr(), "4.5.6.7:9999")
}

func TestFromFlagsMake(t *testing.T) {
	ctrl := gomock.NewController(assert.Tracing(t))
	managedProc := proc.NewMockManagedProc(ctrl)

	ff := &fromFlags{addr: tbnflag.NewHostPort(DefaultListenAddr)}

	adminServer := ff.Make(managedProc).(*adminServer)
	assert.Equal(t, adminServer.server.Addr, DefaultListenAddr)
}
