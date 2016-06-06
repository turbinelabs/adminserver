package main

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/turbinelabs/test/assert"
)

const (
	configData = "some-configuration-data"
)

func TestFromFlagsValidate(t *testing.T) {
	ff := &fromFlags{configFile: ""}
	err := ff.Validate()
	assert.ErrorContains(t, err, "config-file")

	ff.configFile = "blah"
	err = ff.Validate()
	assert.Nil(t, err)
}

func TestFromFlagsMake(t *testing.T) {
	calls := 0
	reload := func() error {
		calls++
		return nil
	}

	ff := &fromFlags{configFile: "the-conf", reload: reload}

	nx, err := ff.Make()
	assert.Nil(t, err)

	nxImpl := nx.(*nginxConfig)
	assert.Equal(t, nxImpl.configFile, "the-conf")
	nxImpl.reload()
	assert.Equal(t, calls, 1)
}

func TestNginxConfigWrite(t *testing.T) {
	tmp, err := ioutil.TempFile("", "nginx-admin-conf.")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmp.Name())

	calls := 0
	reload := func() error {
		calls++
		return nil
	}

	nx := &nginxConfig{&fromFlags{configFile: tmp.Name(), reload: reload}}

	err = nx.Write(configData)
	assert.Nil(t, err)
	assert.Equal(t, calls, 1)

	contents, err := ioutil.ReadFile(tmp.Name())
	assert.Nil(t, err)
	assert.Equal(t, string(contents), configData)
}

func TestNginxNginxConfigWriteShortCircuit(t *testing.T) {
	tmp, err := ioutil.TempFile("", "nginx-admin-conf.")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmp.Name())

	err = ioutil.WriteFile(tmp.Name(), []byte(configData), 664)
	assert.Nil(t, err)

	reload := func() error {
		assert.Tracing(t).Error("unexpected call to reload")
		return nil
	}

	nx := &nginxConfig{&fromFlags{configFile: tmp.Name(), reload: reload}}

	err = nx.Write(configData)
	assert.Nil(t, err)

	contents, err := ioutil.ReadFile(tmp.Name())
	assert.Nil(t, err)
	assert.Equal(t, string(contents), configData)
}

func TestNginxNginxConfigWriteFailure(t *testing.T) {
	reload := func() error {
		assert.Tracing(t).Error("unexpected call to reload")
		return nil
	}

	nx := &nginxConfig{&fromFlags{configFile: "/nope/nope/nope.conf", reload: reload}}

	err := nx.Write("changed")
	assert.ErrorContains(t, err, "no such file")
}
