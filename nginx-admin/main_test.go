package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/turbinelabs/cli/command"
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
}

func testRunWithSignal(t *testing.T, adminCmd string) {
	var waitGroup sync.WaitGroup

	cmd := Cmd()
	cmd.Flags.Parse(
		[]string{
			"-port=0",
			"-nginx=bash",
		})
	runner := cmd.Runner.(*runner)

	args := []string{"-c", "sleep $1", "--", "60"}

	var cmdErr command.CmdErr
	waitGroup.Add(1)
	go func() {
		cmdErr = runner.Run(cmd, args)
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

	assert.Equal(t, string(body), "OK\n")

	waitGroup.Wait()

	assert.DeepEqual(t, cmdErr, command.NoError())
}

func TestRunWithQuit(t *testing.T) {
	testRunWithSignal(t, "quit")
}

func TestRunWithKill(t *testing.T) {
	testRunWithSignal(t, "kill")
}

func TestRunProcExitsNormally(t *testing.T) {
	var waitGroup sync.WaitGroup

	cmd := Cmd()
	cmd.Flags.Parse(
		[]string{
			"-port=0",
			"-nginx=bash",
		})
	runner := cmd.Runner.(*runner)

	args := []string{"-c", "true", "--"}

	var cmdErr command.CmdErr
	waitGroup.Add(1)
	go func() {
		cmdErr = runner.Run(cmd, args)
		waitGroup.Done()
	}()
	waitGroup.Wait()

	assert.DeepEqual(t, cmdErr, command.NoError())
}

func TestRunProcExitsWithFailure(t *testing.T) {
	var waitGroup sync.WaitGroup

	cmd := Cmd()
	cmd.Flags.Parse(
		[]string{
			"-port=0",
			"-nginx=bash",
		})
	runner := cmd.Runner.(*runner)

	args := []string{"-c", "false", "--"}

	var cmdErr command.CmdErr
	waitGroup.Add(1)
	go func() {
		cmdErr = runner.Run(cmd, args)
		waitGroup.Done()
	}()
	waitGroup.Wait()

	assert.NotDeepEqual(t, cmdErr, command.NoError())
}

func TestRunProcExitsWithBadCommand(t *testing.T) {
	cmd := Cmd()
	cmd.Flags.Parse(
		[]string{
			"-port=0",
			"-nginx=nopenopenope",
		})
	runner := cmd.Runner.(*runner)

	args := []string{}

	cmdErr := runner.Run(cmd, args)
	assert.NotDeepEqual(t, cmdErr, command.NoError())
}
