package main

//go:generate mockgen -source $GOFILE -destination mock_$GOFILE -package $GOPACKAGE

import (
	"flag"
	"fmt"
	"os"

	"github.com/turbinelabs/agent/confagent/nginxconfig"
	"github.com/turbinelabs/logparser"
	"github.com/turbinelabs/logparser/metric"
	"github.com/turbinelabs/nonstdlib/proc"
)

const (
	DefaultNginxExecutable = "nginx"
	DefaultNginxConfigFile = "/etc/nginx.conf"
)

type reloader func() error

type FromFlags interface {
	Validate() error

	Source() metric.MetricSource
	MakeNginxConfig(reload reloader) nginxconfig.NginxConfig
	MakeManagedProc(onExit func(error), args []string) proc.ManagedProc
}

type fromFlags struct {
	configFile   string
	nginx        string
	source       string
	metricPrefix string
}

func newFromFlags(flagset *flag.FlagSet) FromFlags {
	ff := &fromFlags{}

	flagset.StringVar(&ff.configFile,
		"config-file",
		DefaultNginxConfigFile,
		"Location of nginx config file",
	)

	flagset.StringVar(&ff.nginx, "nginx", DefaultNginxExecutable, "How to run nginx")

	flagset.StringVar(
		&ff.source,
		"source",
		logparser.DefaultSource(),
		"Sets the default source `tag` for all forwarded metrics.")

	flagset.StringVar(
		&ff.metricPrefix,
		"metric.prefix",
		"",
		"Sets a `prefix` for all forwarded metrics.")

	return ff
}

func (ff *fromFlags) Validate() error {
	if _, err := metric.NewSource(ff.source, ff.metricPrefix); err != nil {
		return err
	}

	if _, err := os.Stat(ff.configFile); os.IsNotExist(err) {
		return fmt.Errorf("config-file does not exist: %s", ff.configFile)
	}

	if _, err := os.Stat(ff.nginx); os.IsNotExist(err) {
		return fmt.Errorf("nginx does not exist: %s", ff.nginx)
	}

	return nil
}

func (ff *fromFlags) Source() metric.MetricSource {
	source, _ := metric.NewSource(ff.source, ff.metricPrefix)
	return source
}

func (ff *fromFlags) MakeNginxConfig(reload reloader) nginxconfig.NginxConfig {
	return &nginxConfig{ff, reload}
}

func (ff *fromFlags) MakeManagedProc(
	onExit func(error),
	args []string,
) proc.ManagedProc {
	mergedArgs := append(args, "-c", ff.configFile, "-g", "daemon off;")

	return proc.NewDefaultManagedProc(ff.nginx, mergedArgs, onExit)
}
