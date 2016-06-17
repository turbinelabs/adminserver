package main

import (
	"io/ioutil"
	"testing"

	"github.com/turbinelabs/test/assert"
	"github.com/turbinelabs/test/tempfile"
)

const (
	configData = "some-configuration-data"
)

func TestNginxConfigWrite(t *testing.T) {
	tmpName, cleanup := tempfile.Make(t, "nginx-admin-conf")
	defer cleanup()

	calls := 0
	reload := func() error {
		calls++
		return nil
	}

	nx := &nginxConfig{&fromFlags{configFile: tmpName}, reload}

	err := nx.Write(configData)
	assert.Nil(t, err)
	assert.Equal(t, calls, 1)

	contents, err := ioutil.ReadFile(tmpName)
	assert.Nil(t, err)
	assert.Equal(t, string(contents), configData)
}

func TestNginxConfigWriteShortCircuit(t *testing.T) {
	tmpName, cleanup := tempfile.Write(t, configData, "nginx-admin-conf")
	defer cleanup()

	reload := func() error {
		assert.Tracing(t).Error("unexpected call to reload")
		return nil
	}

	nx := &nginxConfig{&fromFlags{configFile: tmpName}, reload}

	err := nx.Write(configData)
	assert.Nil(t, err)

	contents, err := ioutil.ReadFile(tmpName)
	assert.Nil(t, err)
	assert.Equal(t, string(contents), configData)
}

func TestNginxConfigWriteFailure(t *testing.T) {
	reload := func() error {
		assert.Tracing(t).Error("unexpected call to reload")
		return nil
	}

	nx := &nginxConfig{&fromFlags{configFile: "/nope/nope/nope.conf"}, reload}

	err := nx.Write("changed")
	assert.ErrorContains(t, err, "no such file")
}
