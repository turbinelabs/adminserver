package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/turbinelabs/adminserver"
	"github.com/turbinelabs/agent/confagent"
	"github.com/turbinelabs/cli"
	"github.com/turbinelabs/cli/command"
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
	r.adminServerConfig = adminserver.NewFromFlags(&cmd.Flags)
	r.confAgentConfig = confagent.NewFromFlags(&cmd.Flags)

	return cmd
}

type runner struct {
	config            FromFlags
	adminServerConfig adminserver.FromFlags
	confAgentConfig   confagent.FromFlags

	adminServerStarted bool
}

func (r *runner) Run(cmd *command.Cmd, args []string) command.CmdErr {
	if err := r.config.Validate(); err != nil {
		return command.BadInput(err.Error())
	}

	if err := r.confAgentConfig.Validate(); err != nil {
		return command.BadInput(err.Error())
	}

	if err := r.adminServerConfig.Validate(); err != nil {
		return command.BadInput(err.Error())
	}

	var managedProc proc.ManagedProc
	reload := func() error {
		if managedProc == nil {
			return errors.New("no running nginx process")
		}

		return managedProc.Hangup()
	}

	confAgent, err := r.confAgentConfig.Make(r.config.MakeNginxConfig(reload))
	if err != nil {
		return command.Error(err.Error())
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
		return command.Error(err.Error())
	}

	go func() {
		for {
			if err := confAgent.Poll(); err != nil {
				fmt.Fprintf(os.Stderr, "api polling error: %s", err.Error())
			}
		}
	}()

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
				cmdErr = command.Error(errMsg)
			}

		case adminserver.RequestedQuitSignal:
			if !strings.HasPrefix(errMsg, "signal: quit") {
				cmdErr = command.Error(errMsg)
			}

		default:
			cmdErr = command.Error(errMsg)
		}

		return cmdErr
	}

	if err != nil && !managedProc.Completed() {
		// AdminServer failed to start. Ignore if process
		// completed, because onExit may close the server
		// before it even starts causing a spurrious error.
		return command.Error(err.Error())
	}

	return command.NoError()
}

func main() {
	app := cli.New("0.1", Cmd())
	app.Main()
}
