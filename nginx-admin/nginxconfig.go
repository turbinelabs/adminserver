package main

import (
	"io/ioutil"

	"github.com/turbinelabs/agent/confagent/nginxconfig"
)

type nginxConfig struct {
	*fromFlags
	reload reloader
}

var _ nginxconfig.NginxConfig = &nginxConfig{}

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
