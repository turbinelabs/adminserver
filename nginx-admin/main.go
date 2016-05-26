package main

import (
	"strings"
	"sync"

	"github.com/turbinelabs/adminserver"
	"github.com/turbinelabs/cli"
	"github.com/turbinelabs/cli/command"
	"github.com/turbinelabs/proc"
)

const (
	DefaultNginxExecutable = "nginx"
	DefaultListenIP        = "127.0.0.1"
	DefaultListenPort      = 9000
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

	cmd.Flags.StringVar(&r.ListenIP, "ip", DefaultListenIP, "What IP should we listen on")
	cmd.Flags.IntVar(&r.ListenPort, "port", DefaultListenPort, "What port should we listen on")
	cmd.Flags.StringVar(&r.Nginx, "nginx", DefaultNginxExecutable, "How to run nginx")

	return cmd
}

type runner struct {
	ListenIP   string
	ListenPort int
	Nginx      string

	managedProc proc.ManagedProc
	adminServer adminserver.AdminServer
	procErr     error
	waitGroup   sync.WaitGroup
}

func (r *runner) onExit(err error) {
	r.procErr = err
	r.adminServer.Close()
	r.waitGroup.Done()
}

func (r *runner) Run(cmd *command.Cmd, args []string) command.CmdErr {
	mergedArgs := append(args, "-g", "daemon off;")

	r.waitGroup.Add(1)

	var err error
	r.managedProc, err = proc.NewManagedProc(r.Nginx, mergedArgs, r.onExit)
	if err != nil {
		return command.Error(err.Error())
	}

	r.adminServer = adminserver.New(r.ListenIP, r.ListenPort, r.managedProc)

	err = r.adminServer.Start()

	r.waitGroup.Wait()

	if r.procErr != nil {
		// Process exited with error, but ignore signals that
		// we sent on purpose.
		errMsg := r.procErr.Error()
		cmdErr := command.NoError()

		switch r.adminServer.LastRequestedSignal() {
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

	if err != nil && !r.managedProc.Completed() {
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
