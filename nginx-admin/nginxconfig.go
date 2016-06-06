package main

import (
	"errors"
	"flag"
	"io/ioutil"

	"github.com/turbinelabs/agent/confagent/nginxconfig"
)

type reloader func() error

func newFromFlags(flagset *flag.FlagSet, reload reloader) *fromFlags {
	ff := &fromFlags{reload: reload}

	flagset.StringVar(
		&ff.configFile,
		"config-file",
		"/etc/nginx.conf",
		"[REQUIRED] Location of nginx config file",
	)

	return ff
}

type fromFlags struct {
	configFile string
	reload     reloader
}

func (f *fromFlags) Validate() error {
	if f.configFile == "" {
		return errors.New("missing config-file")
	}

	return nil
}

func (f *fromFlags) Make() (nginxconfig.NginxConfig, error) {
	return &nginxConfig{f}, nil
}

type nginxConfig struct {
	*fromFlags
}

func (c *nginxConfig) Write(config string) error {
	oldConfigBytes, err := ioutil.ReadFile(c.configFile)
	if err == nil && config == string(oldConfigBytes) {
		// short circuit: successful read and no change on disk
		return nil
	}

	err = ioutil.WriteFile(c.configFile, []byte(config), 0664)
	if err != nil {
		return err
	}

	return c.reload()
}
