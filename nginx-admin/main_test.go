package main

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	"github.com/turbinelabs/adminserver"
	"github.com/turbinelabs/agent/confagent"
	"github.com/turbinelabs/agent/confagent/nginxconfig"
	"github.com/turbinelabs/cli/command"
	"github.com/turbinelabs/proc"
	"github.com/turbinelabs/test/assert"
)

const (
	configFilePath = "/some/file"
)

func dummyReloadFunc() error {
	return nil
}

type runnerTestCase struct {
	ctrl        *gomock.Controller
	runner      *runner
	adminServer *adminserver.MockAdminServer
	managedProc *proc.MockManagedProc

	onExit func(error)
	reload reloader
}

func mkMockRunner(t *testing.T, args []string) *runnerTestCase {
	ctrl := gomock.NewController(assert.Tracing(t))

	testcase := &runnerTestCase{}

	recordOnExit := func(onExit func(error), args []string) {
		testcase.onExit = onExit
	}

	recordReload := func(reload reloader) {
		testcase.reload = reload
	}

	mainFromFlags := NewMockFromFlags(ctrl)
	managedProc := proc.NewMockManagedProc(ctrl)
	nginxConfig := nginxconfig.NewMockNginxConfig(ctrl)

	confAgentFromFlags := confagent.NewMockFromFlags(ctrl)
	confAgent := confagent.NewMockConfAgent(ctrl)

	adminServerFromFlags := adminserver.NewMockFromFlags(ctrl)
	adminServer := adminserver.NewMockAdminServer(ctrl)

	mainFromFlags.EXPECT().Validate().Return(nil)
	confAgentFromFlags.EXPECT().Validate().Return(nil)
	adminServerFromFlags.EXPECT().Validate().Return(nil)

	mainFromFlags.EXPECT().
		MakeNginxConfig(gomock.Any()).
		Do(recordReload).
		Return(nginxConfig)

	confAgentFromFlags.EXPECT().Make(nginxConfig).Return(confAgent, nil)

	mainFromFlags.EXPECT().
		MakeManagedProc(gomock.Any(), args).
		Do(recordOnExit).
		Return(managedProc)

	adminServerFromFlags.EXPECT().Make(managedProc).Return(adminServer)

	confAgent.EXPECT().Poll().AnyTimes().Return(nil)

	runner := &runner{
		config:            mainFromFlags,
		adminServerConfig: adminServerFromFlags,
		confAgentConfig:   confAgentFromFlags,
	}

	testcase.ctrl = ctrl
	testcase.runner = runner
	testcase.adminServer = adminServer
	testcase.managedProc = managedProc

	return testcase
}

func testRunWithSignal(t *testing.T, adminCmd string, wrongProcErr bool) {
	var waitGroup sync.WaitGroup

	args := []string{"a", "b", "c"}

	test := mkMockRunner(t, args)

	test.managedProc.EXPECT().Start().Return(nil)

	test.adminServer.EXPECT().Start().Return(nil)

	var exitError string
	if adminCmd == "quit" {
		exitError = "signal: quit"
		test.adminServer.EXPECT().LastRequestedSignal().Return(
			adminserver.RequestedQuitSignal,
		)
	} else if adminCmd == "kill" {
		exitError = "signal: killed"
		test.adminServer.EXPECT().LastRequestedSignal().Return(
			adminserver.RequestedKillSignal,
		)
	} else {
		assert.Tracing(t).Fatalf("unknown admin cmd: %s", adminCmd)
	}

	test.adminServer.EXPECT().Close().Return(nil)

	var cmdErr command.CmdErr
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		cmdErr = test.runner.Run(Cmd(), args)
	}()

	for !test.runner.adminServerStarted {
		time.Sleep(10 * time.Millisecond)
	}

	if wrongProcErr {
		test.onExit(errors.New("unexpected error"))
	} else {
		test.onExit(errors.New(exitError))
	}

	waitGroup.Wait()

	if wrongProcErr {
		assert.DeepEqual(t, cmdErr, command.Errorf("unexpected error"))
	} else {
		assert.DeepEqual(t, cmdErr, command.NoError())
	}

	test.ctrl.Finish()
}

func TestCmd(t *testing.T) {
	cmd := Cmd()
	cmd.Flags.Parse(
		[]string{
			"-ip=10.0.0.0",
			"-port=9999",
			"-nginx=sleep",
		})
	runner := cmd.Runner.(*runner)
	assert.NonNil(t, runner.adminServerConfig)
	assert.NonNil(t, runner.confAgentConfig)
	assert.NonNil(t, runner.config)
}

func TestRunWithQuit(t *testing.T) {
	testRunWithSignal(t, "quit", false)
}

func TestRunWithQuitReturningWrongError(t *testing.T) {
	testRunWithSignal(t, "quit", true)
}

func TestRunWithKill(t *testing.T) {
	testRunWithSignal(t, "kill", false)
}

func TestRunWithKillReturningWrongError(t *testing.T) {
	testRunWithSignal(t, "kill", true)
}

func TestRunProcExitsNormally(t *testing.T) {
	var waitGroup sync.WaitGroup

	test := mkMockRunner(t, []string{})

	test.managedProc.EXPECT().Start().Return(nil)

	test.adminServer.EXPECT().Start().Return(nil)
	test.adminServer.EXPECT().Close().Return(nil)

	var cmdErr command.CmdErr
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		cmdErr = test.runner.Run(Cmd(), []string{})
	}()

	for !test.runner.adminServerStarted {
		time.Sleep(10 * time.Millisecond)
	}

	test.onExit(nil)

	waitGroup.Wait()

	assert.DeepEqual(t, cmdErr, command.NoError())

	test.ctrl.Finish()
}

func TestRunProcExitsWithFailure(t *testing.T) {
	var waitGroup sync.WaitGroup

	test := mkMockRunner(t, []string{})

	test.managedProc.EXPECT().Start().Return(nil)
	test.adminServer.EXPECT().Start().Return(nil)
	test.adminServer.EXPECT().LastRequestedSignal().Return(adminserver.NoRequestedSignal)
	test.adminServer.EXPECT().Close().Return(nil)

	var cmdErr command.CmdErr
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		cmdErr = test.runner.Run(Cmd(), []string{})
	}()

	for !test.runner.adminServerStarted {
		time.Sleep(10 * time.Millisecond)
	}

	test.onExit(errors.New("boom: exit status 1"))

	waitGroup.Wait()

	assert.NotDeepEqual(t, cmdErr, command.NoError())
	assert.MatchesRegex(t, cmdErr.Message, "exit status 1")

	test.ctrl.Finish()
}

func TestRunProcExitsWithBadCommand(t *testing.T) {
	args := []string{"a", "b"}
	ctrl := gomock.NewController(assert.Tracing(t))

	mainFromFlags := NewMockFromFlags(ctrl)
	managedProc := proc.NewMockManagedProc(ctrl)
	nginxConfig := nginxconfig.NewMockNginxConfig(ctrl)

	confAgentFromFlags := confagent.NewMockFromFlags(ctrl)
	confAgent := confagent.NewMockConfAgent(ctrl)

	adminServerFromFlags := adminserver.NewMockFromFlags(ctrl)

	mainFromFlags.EXPECT().Validate().Return(nil)
	confAgentFromFlags.EXPECT().Validate().Return(nil)
	adminServerFromFlags.EXPECT().Validate().Return(nil)

	mainFromFlags.EXPECT().MakeNginxConfig(gomock.Any()).Return(nginxConfig)
	confAgentFromFlags.EXPECT().Make(nginxConfig).Return(confAgent, nil)
	mainFromFlags.EXPECT().MakeManagedProc(gomock.Any(), args).Return(managedProc)

	managedProc.EXPECT().Start().Return(errors.New("file not found"))

	runner := &runner{
		config:            mainFromFlags,
		adminServerConfig: adminServerFromFlags,
		confAgentConfig:   confAgentFromFlags,
	}

	cmdErr := runner.Run(Cmd(), args)
	assert.NotDeepEqual(t, cmdErr, command.NoError())
	assert.MatchesRegex(t, cmdErr.Message, "not found")

	ctrl.Finish()
}

func TestRunProcExitsWithInvalidConfig(t *testing.T) {
	ctrl := gomock.NewController(assert.Tracing(t))

	mainFromFlags := NewMockFromFlags(ctrl)
	mainFromFlags.EXPECT().Validate().Return(errors.New("boom"))

	runner := &runner{
		config: mainFromFlags,
	}

	args := []string{}

	cmdErr := runner.Run(Cmd(), args)
	assert.NotDeepEqual(t, cmdErr, command.NoError())
	assert.MatchesRegex(t, cmdErr.Message, "boom")

	ctrl.Finish()
}

func TestRunProcExitsWithInvalidConfAgent(t *testing.T) {
	ctrl := gomock.NewController(assert.Tracing(t))

	mainFromFlags := NewMockFromFlags(ctrl)
	mainFromFlags.EXPECT().Validate().Return(nil)

	confAgentFromFlags := confagent.NewMockFromFlags(ctrl)
	confAgentFromFlags.EXPECT().Validate().Return(errors.New("boom"))

	runner := &runner{
		config:          mainFromFlags,
		confAgentConfig: confAgentFromFlags,
	}

	args := []string{}

	cmdErr := runner.Run(Cmd(), args)
	assert.NotDeepEqual(t, cmdErr, command.NoError())
	assert.MatchesRegex(t, cmdErr.Message, "boom")

	ctrl.Finish()
}

func TestRunProcExitsWithInvalidAdminServer(t *testing.T) {
	ctrl := gomock.NewController(assert.Tracing(t))

	mainFromFlags := NewMockFromFlags(ctrl)
	mainFromFlags.EXPECT().Validate().Return(nil)

	confAgentFromFlags := confagent.NewMockFromFlags(ctrl)
	confAgentFromFlags.EXPECT().Validate().Return(nil)

	adminServerFromFlags := adminserver.NewMockFromFlags(ctrl)
	adminServerFromFlags.EXPECT().Validate().Return(errors.New("boom"))

	runner := &runner{
		config:            mainFromFlags,
		adminServerConfig: adminServerFromFlags,
		confAgentConfig:   confAgentFromFlags,
	}

	args := []string{}

	cmdErr := runner.Run(Cmd(), args)
	assert.NotDeepEqual(t, cmdErr, command.NoError())
	assert.MatchesRegex(t, cmdErr.Message, "boom")

	ctrl.Finish()
}

func TestRunExitsWithConfAgentMakeError(t *testing.T) {
	ctrl := gomock.NewController(assert.Tracing(t))

	mainFromFlags := NewMockFromFlags(ctrl)
	mainFromFlags.EXPECT().Validate().Return(nil)

	nginxConfig := nginxconfig.NewMockNginxConfig(ctrl)
	mainFromFlags.EXPECT().MakeNginxConfig(gomock.Any()).Return(nginxConfig)

	confAgentFromFlags := confagent.NewMockFromFlags(ctrl)
	confAgentFromFlags.EXPECT().Validate().Return(nil)
	confAgentFromFlags.EXPECT().Make(nginxConfig).Return(nil, errors.New("make error"))

	adminServerFromFlags := adminserver.NewMockFromFlags(ctrl)
	adminServerFromFlags.EXPECT().Validate().Return(nil)

	runner := &runner{
		config:            mainFromFlags,
		adminServerConfig: adminServerFromFlags,
		confAgentConfig:   confAgentFromFlags,
	}

	args := []string{}

	cmdErr := runner.Run(Cmd(), args)
	assert.NotDeepEqual(t, cmdErr, command.NoError())
	assert.MatchesRegex(t, cmdErr.Message, "make error")

	ctrl.Finish()
}

func TestRunAdminServerFailsToStart(t *testing.T) {
	var waitGroup sync.WaitGroup

	test := mkMockRunner(t, []string{})

	test.managedProc.EXPECT().Start().Return(nil)
	test.managedProc.EXPECT().Completed().Return(false)

	test.adminServer.EXPECT().Start().Return(errors.New("something something network"))
	test.adminServer.EXPECT().Close().Return(nil)

	var cmdErr command.CmdErr
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		cmdErr = test.runner.Run(Cmd(), []string{})
	}()

	for !test.runner.adminServerStarted {
		time.Sleep(10 * time.Millisecond)
	}

	test.onExit(nil) // process exits without error

	waitGroup.Wait()

	assert.NotDeepEqual(t, cmdErr, command.NoError())
	assert.Equal(t, cmdErr.Message, "something something network")

	test.ctrl.Finish()
}

func TestRunnerReload(t *testing.T) {
	var waitGroup sync.WaitGroup

	args := []string{"a", "b", "c"}

	test := mkMockRunner(t, args)

	test.managedProc.EXPECT().Start().Return(nil)

	test.adminServer.EXPECT().Start().Return(nil)

	var cmdErr command.CmdErr
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		cmdErr = test.runner.Run(Cmd(), args)
	}()

	for !test.runner.adminServerStarted {
		time.Sleep(10 * time.Millisecond)
	}

	test.managedProc.EXPECT().Hangup().Return(nil)

	test.reload()

	test.adminServer.EXPECT().Close().Return(nil)

	test.onExit(nil)

	waitGroup.Wait()

	test.ctrl.Finish()
}
