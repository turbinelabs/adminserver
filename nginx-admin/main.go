package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/turbinelabs/adminserver"
	"github.com/turbinelabs/adminserver/nginx-admin/logrotater"
	"github.com/turbinelabs/agent/confagent"
	apiflags "github.com/turbinelabs/api/service/http/flags"
	"github.com/turbinelabs/cli"
	"github.com/turbinelabs/cli/command"
	"github.com/turbinelabs/cli/flags"
	"github.com/turbinelabs/logparser"
	"github.com/turbinelabs/logparser/forwarder"
	"github.com/turbinelabs/logparser/parser"
	"github.com/turbinelabs/proc"
)

func Cmd() *command.Cmd {
	r := &runner{}

	cmd := &command.Cmd{
		Name:        "nginx-admin",
		Summary:     "Turbine Labs Nginx Admin Wrapper",
		Usage:       "[OPTIONS] -- [nginx options]",
		Description: "Runs nginx in non-daemon mode and implements an admin port to allow the nginx process to be controlled.",
		Runner:      r,
	}

	r.config = newFromFlags(&cmd.Flags)
	r.apiConfig = apiflags.NewAPIConfigFromFlags(&cmd.Flags)
	r.adminServerConfig = adminserver.NewFromFlags(&cmd.Flags)
	r.confAgentConfig = confagent.NewFromFlagsWithSharedAPIConfig(&cmd.Flags, r.apiConfig)

	r.accessLogParserConfig = logparser.NewFromFlags(
		flags.NewPrefixedFlagSet(
			&cmd.Flags,
			"accesslog",
			"access log",
		),
		logparser.APIConfig(r.apiConfig),
		logparser.DefaultParserType(parser.NoopParserType),
		logparser.DefaultForwarderType(forwarder.NoopForwarderType),
	)

	r.upstreamLogParserConfig = logparser.NewFromFlags(
		flags.NewPrefixedFlagSet(
			&cmd.Flags,
			"upstreamlog",
			"upstream log",
		),
		logparser.APIConfig(r.apiConfig),
		logparser.DefaultParserType(parser.NoopParserType),
		logparser.DefaultForwarderType(forwarder.NoopForwarderType),
	)

	r.logRotaterConfig = logrotater.NewFromFlags(
		flags.NewPrefixedFlagSet(
			&cmd.Flags,
			"logrotate",
			"nginx log files",
		),
	)

	return cmd
}

type runner struct {
	config                  FromFlags
	apiConfig               apiflags.APIConfigFromFlags
	adminServerConfig       adminserver.FromFlags
	confAgentConfig         confagent.FromFlags
	accessLogParserConfig   logparser.FromFlags
	upstreamLogParserConfig logparser.FromFlags
	logRotaterConfig        logrotater.FromFlags

	adminServerStarted bool
}

func (r *runner) Run(cmd *command.Cmd, args []string) command.CmdErr {
	if err := r.config.Validate(); err != nil {
		return cmd.BadInput(err)
	}

	if err := r.confAgentConfig.Validate(); err != nil {
		return cmd.BadInput(err)
	}

	if err := r.adminServerConfig.Validate(); err != nil {
		return cmd.BadInput(err)
	}

	if err := r.logRotaterConfig.Validate(); err != nil {
		return cmd.BadInput(err)
	}

	if err := r.accessLogParserConfig.Validate(); err != nil {
		return cmd.BadInput(err)
	}

	if err := r.upstreamLogParserConfig.Validate(); err != nil {
		return cmd.BadInput(err)
	}

	source := r.config.Source()

	var managedProc proc.ManagedProc
	reload := func() error {
		if managedProc == nil {
			return errors.New("no running nginx process")
		}

		return managedProc.Hangup()
	}
	reopen := func() error {
		if managedProc == nil {
			return errors.New("no running nginx process")
		}

		return managedProc.Usr1()
	}
	confAgent, err := r.confAgentConfig.Make(r.config.MakeNginxConfig(reload))
	if err != nil {
		return cmd.Error(err)
	}

	logRotater := r.logRotaterConfig.Make(logparser.DefaultLogger(), reopen)
	defer logRotater.StopAll()

	paths := confAgent.GetPaths()
	if err := logRotater.Start(paths.AccessLog); err != nil {
		return cmd.Error(err)
	}
	if err := logRotater.Start(paths.UpstreamLog); err != nil {
		return cmd.Error(err)
	}

	var adminServer adminserver.AdminServer
	var procErr error
	var procWaitGroup sync.WaitGroup
	onExit := func(err error) {
		defer procWaitGroup.Done()
		procErr = err
		if adminServer != nil {
			adminServer.Close()
		}
	}

	managedProc = r.config.MakeManagedProc(onExit, args)

	procWaitGroup.Add(1)

	err = managedProc.Start()
	if err != nil {
		return cmd.Error(err)
	}
	defer managedProc.Kill()

	go func() {
		for {
			if err := confAgent.Poll(); err != nil {
				fmt.Fprintf(os.Stderr, "api polling error: %s", err.Error())
			}
		}
	}()

	accessLogParser, err := r.accessLogParserConfig.Make(logparser.DefaultLogger(), source)
	if err != nil {
		return cmd.Error(err)
	}
	defer accessLogParser.Close()
	go accessLogParser.Tail(paths.AccessLog)

	upstreamLogParser, err := r.upstreamLogParserConfig.Make(logparser.DefaultLogger(), source)
	if err != nil {
		return cmd.Error(err)
	}
	defer upstreamLogParser.Close()
	go upstreamLogParser.Tail(paths.UpstreamLog)

	adminServer = r.adminServerConfig.Make(managedProc)
	r.adminServerStarted = true
	err = adminServer.Start()

	procWaitGroup.Wait()

	if procErr != nil {
		// Process exited with error, but ignore signals that
		// we sent on purpose.
		errMsg := procErr.Error()
		cmdErr := command.NoError()

		switch adminServer.LastRequestedSignal() {
		case adminserver.RequestedKillSignal:
			if !strings.HasPrefix(errMsg, "signal: killed") {
				cmdErr = cmd.Error(errMsg)
			}

		case adminserver.RequestedQuitSignal:
			if !strings.HasPrefix(errMsg, "signal: quit") {
				cmdErr = cmd.Error(errMsg)
			}

		default:
			cmdErr = cmd.Error(errMsg)
		}

		return cmdErr
	}

	if err != nil && !managedProc.Completed() {
		// AdminServer failed to start. Ignore if process
		// completed, because onExit may close the server
		// before it even starts causing a spurrious error.
		return cmd.Error(err)
	}

	return command.NoError()
}

func mkCLI() cli.CLI {
	return cli.New("0.1", Cmd())
}

func main() {
	mkCLI().Main()
}
