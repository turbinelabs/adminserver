package main

import (
	"flag"
	"os"
	"testing"
	"time"

	"github.com/turbinelabs/test/assert"
	"github.com/turbinelabs/test/tempfile"
)

func TestNewFromFlags(t *testing.T) {
	flagset := flag.NewFlagSet("fromFlag options", flag.PanicOnError)
	ff := newFromFlags(flagset)
	ffImpl := ff.(*fromFlags)
	assert.Equal(t, ffImpl.configFile, DefaultNginxConfigFile)
	assert.Equal(t, ffImpl.nginx, DefaultNginxExecutable)

	flagset.Parse([]string{"-nginx=/other/path", "-config-file=/other/conf"})

	assert.Equal(t, ffImpl.configFile, "/other/conf")
	assert.Equal(t, ffImpl.nginx, "/other/path")
}

func TestFromFlagsValidate(t *testing.T) {
	ff := &fromFlags{}
	err := ff.Validate()
	assert.ErrorContains(t, err, "invalid metric source")

	ff.source = "src"
	ff.configFile = "/nope/nope/nope"
	err = ff.Validate()
	assert.ErrorContains(t, err, "config-file does not exist")

	configFile, cleanup := tempfile.Make(t, "nginx-config-file")
	defer cleanup()

	ff.configFile = configFile
	ff.nginx = "/nope/nope/nope"
	err = ff.Validate()
	assert.ErrorContains(t, err, "nginx does not exist")

	nginx, cleanup2 := tempfile.Make(t, "nginx-fake-bin")
	defer cleanup2()

	ff.nginx = nginx
	err = ff.Validate()
	assert.Nil(t, err)
}

func TestFromFlagsSource(t *testing.T) {
	ff := &fromFlags{source: "s"}

	source := ff.Source()
	assert.Equal(t, source.Source(), "s")
	metric, err := source.NewMetric("m")
	assert.Nil(t, err)
	assert.Equal(t, metric.Name(), "m")

	ff.metricPrefix = "p"
	source = ff.Source()
	assert.Equal(t, source.Source(), "s")
	metric, err = source.NewMetric("m")
	assert.Nil(t, err)
	assert.Equal(t, metric.Name(), "p.m")
}

func TestFromFlagsMakeNginxConfig(t *testing.T) {
	calls := 0
	reload := func() error {
		calls++
		return nil
	}

	ff := &fromFlags{configFile: "the-conf"}

	nx := ff.MakeNginxConfig(reload)
	nxImpl := nx.(*nginxConfig)
	assert.Equal(t, nxImpl.configFile, "the-conf")
	nxImpl.reload()
	assert.Equal(t, calls, 1)
}

func TestFromFlagsMakeManagedProc(t *testing.T) {
	truth := []string{"/usr/bin/true", "/bin/true"}
	var trueExe string
	for _, t := range truth {
		s, err := os.Stat(t)
		if err == nil && s.Mode().Perm()&0111 != 0 {
			trueExe = t
			break
		}
	}
	if trueExe == "" {
		t.Fatalf("cannot find 'true' executable")
	}

	ff := &fromFlags{nginx: trueExe}

	calls := 0
	var recordedError error
	onExit := func(err error) {
		calls++
		recordedError = err
	}

	mp := ff.MakeManagedProc(onExit, []string{""})
	assert.Nil(t, mp.Start())

	for calls == 0 {
		time.Sleep(10 * time.Millisecond)
	}

	assert.Equal(t, calls, 1)
	assert.Nil(t, recordedError)
}
