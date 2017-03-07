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
	"fmt"
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
	assert.Equal(t, ffImpl.ip, DefaultListenIP)
	assert.Equal(t, ffImpl.port, DefaultListenPort)

	flagset.Parse([]string{"-admin.ip=4.5.6.7", "-admin.port=9999"})

	assert.Equal(t, ffImpl.ip, "4.5.6.7")
	assert.Equal(t, ffImpl.port, 9999)
}

func TestFromFlagsValidate(t *testing.T) {
	ff := &fromFlags{ip: "4.5.6.7", port: 9999}
	assert.Nil(t, ff.Validate())

	ff = &fromFlags{ip: "not-an-ip", port: 9999}
	assert.ErrorContains(t, ff.Validate(), "invalid ip")

	ff = &fromFlags{ip: "4.5.6.7", port: 100000}
	assert.ErrorContains(t, ff.Validate(), "invalid port")
}

func TestFromFlagsMake(t *testing.T) {
	ctrl := gomock.NewController(assert.Tracing(t))
	managedProc := proc.NewMockManagedProc(ctrl)

	ff := &fromFlags{ip: DefaultListenIP, port: 0}

	adminServer := ff.Make(managedProc).(*adminServer)
	assert.Equal(t, adminServer.server.Addr, fmt.Sprintf("%s:%d", DefaultListenIP, 0))
}
