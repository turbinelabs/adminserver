package adminserver

import (
	"flag"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/turbinelabs/proc"
	"github.com/turbinelabs/test/assert"
)

func TestFromFlags(t *testing.T) {
	flagset := flag.NewFlagSet("adminserver options", flag.PanicOnError)
	ff := NewFromFlags(flagset)
	ffImpl := ff.(*fromFlags)
	assert.Equal(t, ffImpl.ip, DefaultListenIP)
	assert.Equal(t, ffImpl.port, DefaultListenPort)

	flagset.Parse([]string{"-ip=4.5.6.7", "-port=9999"})

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
