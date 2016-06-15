package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	"github.com/turbinelabs/agent/confagent"
	"github.com/turbinelabs/cli/command"
	"github.com/turbinelabs/proc"
	"github.com/turbinelabs/test/assert"
)

func TestCmd(t *testing.T) {
	cmd := Cmd()
	cmd.Flags.Parse(
		[]string{
			"-ip=10.0.0.0",
			"-port=9999",
			"-nginx=sleep",
		})
	runner := cmd.Runner.(*runner)
	assert.Equal(t, runner.ListenIP, "10.0.0.0")
	assert.Equal(t, runner.ListenPort, 9999)
	assert.Equal(t, runner.Nginx, "sleep")
	assert.NonNil(t, runner.confAgentConfig)
}

func mkMockBashRunner(t *testing.T) (*gomock.Controller, *runner) {
	return mkMockRunner(t, "bash")
}

func mkMockRunner(t *testing.T, cmd string) (*gomock.Controller, *runner) {
	ctrl := gomock.NewController(assert.Tracing(t))

	confAgentFromFlags := confagent.NewMockFromFlags(ctrl)
	confAgent := confagent.NewMockConfAgent(ctrl)

	confAgentFromFlags.EXPECT().Validate().Return(nil)
	confAgentFromFlags.EXPECT().Make().Return(confAgent, nil)
	confAgent.EXPECT().Poll().AnyTimes().Return(nil)

	reloadFunc := func() error {
		return nil
	}

	runner := &runner{
		ListenPort:      0,
		Nginx:           cmd,
		confAgentConfig: confAgentFromFlags,
		nginxConfig:     &fromFlags{configFile: "/some/file", reload: reloadFunc},
	}

	return ctrl, runner
}

func testRunWithSignal(t *testing.T, adminCmd string) {
	var waitGroup sync.WaitGroup

	ctrl, runner := mkMockBashRunner(t)

	args := []string{"-c", "sleep $1", "--", "60"}

	var cmdErr command.CmdErr
	waitGroup.Add(1)
	go func() {
		cmdErr = runner.Run(Cmd(), args)
		waitGroup.Done()
	}()

	for runner.adminServer == nil || !runner.adminServer.Listening() {
		time.Sleep(10 * time.Millisecond)
	}

	addr := runner.adminServer.Addr()

	resp, err := http.Get(fmt.Sprintf("http://%s/admin/%s", addr, adminCmd))
	assert.Nil(t, err)
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, string(body), "OK\n")

	waitGroup.Wait()

	assert.DeepEqual(t, cmdErr, command.NoError())

	ctrl.Finish()
}

func TestRunWithQuit(t *testing.T) {
	testRunWithSignal(t, "quit")
}

func TestRunWithKill(t *testing.T) {
	testRunWithSignal(t, "kill")
}

func TestRunProcExitsNormally(t *testing.T) {
	var waitGroup sync.WaitGroup

	ctrl, runner := mkMockBashRunner(t)

	args := []string{"-c", "true", "--"}

	var cmdErr command.CmdErr
	waitGroup.Add(1)
	go func() {
		cmdErr = runner.Run(Cmd(), args)
		waitGroup.Done()
	}()
	waitGroup.Wait()

	assert.DeepEqual(t, cmdErr, command.NoError())

	ctrl.Finish()
}

func TestRunProcExitsWithFailure(t *testing.T) {
	var waitGroup sync.WaitGroup

	ctrl, runner := mkMockBashRunner(t)

	args := []string{"-c", "false", "--"}

	var cmdErr command.CmdErr
	waitGroup.Add(1)
	go func() {
		cmdErr = runner.Run(Cmd(), args)
		waitGroup.Done()
	}()
	waitGroup.Wait()

	assert.NotDeepEqual(t, cmdErr, command.NoError())
	assert.MatchesRegex(t, cmdErr.Message, "exit status 1")

	ctrl.Finish()
}

func TestRunProcExitsWithBadCommand(t *testing.T) {
	ctrl, runner := mkMockRunner(t, "nopenopenope")

	args := []string{}

	cmdErr := runner.Run(Cmd(), args)
	assert.NotDeepEqual(t, cmdErr, command.NoError())
	assert.MatchesRegex(t, cmdErr.Message, "not found")

	ctrl.Finish()
}

func TestRunProcExitsWithInvalidConfAgent(t *testing.T) {
	ctrl := gomock.NewController(assert.Tracing(t))

	confAgentFromFlags := confagent.NewMockFromFlags(ctrl)
	confAgentFromFlags.EXPECT().Validate().Return(fmt.Errorf("boom"))

	runner := &runner{
		ListenPort:      0,
		Nginx:           "bash",
		confAgentConfig: confAgentFromFlags,
	}

	args := []string{}

	cmdErr := runner.Run(Cmd(), args)
	assert.NotDeepEqual(t, cmdErr, command.NoError())
	assert.MatchesRegex(t, cmdErr.Message, "boom")

	ctrl.Finish()
}

func TestRunProcExitsWithConfAgentMakeError(t *testing.T) {
	ctrl := gomock.NewController(assert.Tracing(t))

	confAgentFromFlags := confagent.NewMockFromFlags(ctrl)
	confAgentFromFlags.EXPECT().Validate().Return(nil)
	confAgentFromFlags.EXPECT().Make().Return(nil, fmt.Errorf("make error"))

	runner := &runner{
		ListenPort:      0,
		Nginx:           "bash",
		confAgentConfig: confAgentFromFlags,
	}

	args := []string{}

	cmdErr := runner.Run(Cmd(), args)
	assert.NotDeepEqual(t, cmdErr, command.NoError())
	assert.MatchesRegex(t, cmdErr.Message, "make error")

	ctrl.Finish()
}

func TestRunnerReload(t *testing.T) {
	ctrl := gomock.NewController(assert.Tracing(t))

	managedProc := proc.NewMockManagedProc(ctrl)
	managedProc.EXPECT().Hangup().Return(fmt.Errorf("hangup error"))
	managedProc.EXPECT().Hangup().Return(nil)

	runner := &runner{}

	err := runner.reload()
	assert.ErrorContains(t, err, "no running nginx process")

	runner.managedProc = managedProc

	err = runner.reload()
	assert.ErrorContains(t, err, "hangup error")

	err = runner.reload()
	assert.Nil(t, err)

	ctrl.Finish()
}
