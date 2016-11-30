package main

import (
	"errors"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	"github.com/turbinelabs/adminserver"
	"github.com/turbinelabs/adminserver/nginx-admin/logrotater"
	"github.com/turbinelabs/agent/confagent"
	"github.com/turbinelabs/agent/confagent/nginxconfig"
	apiflags "github.com/turbinelabs/api/client/flags"
	statsapi "github.com/turbinelabs/api/service/stats"
	"github.com/turbinelabs/cli/command"
	"github.com/turbinelabs/configwriter"
	"github.com/turbinelabs/logparser"
	"github.com/turbinelabs/logparser/metric"
	"github.com/turbinelabs/nonstdlib/executor"
	"github.com/turbinelabs/nonstdlib/proc"
	"github.com/turbinelabs/stats/client"
	"github.com/turbinelabs/test/assert"
)

const (
	configFilePath = "/some/file"
)

func dummyReloadFunc() error {
	return nil
}

type runnerConfig struct {
	args []string

	mainValidateErr  error
	mainMakeNginxErr error
	mainMakeProcErr  error

	procStartErr error

	apiConfigValidateErr error

	confAgentValidateErr error
	confAgentMakeErr     error

	logRotaterValidateErr error

	accessLogParserValidateErr error
	accessLogParserMakeErr     error
	accessLogStartRotateErr    error

	upstreamLogParserValidateErr error
	upstreamLogParserMakeErr     error
	upstreamLogStartRotateErr    error

	adminServerValidateErr error
	adminServerStartErr    error

	statsClientValidateErr error
	statsClientMakeErr     error
}

type runnerTestCase struct {
	args                []string
	t                   *testing.T
	ctrl                *gomock.Controller
	runner              *runner
	adminServer         *adminserver.MockAdminServer
	executor            *executor.MockExecutor
	statsSvc            *statsapi.MockStatsService
	managedProc         *proc.MockManagedProc
	accessLogParser     *logparser.MockLogParser
	accessLogTailDone   bool
	upstreamLogParser   *logparser.MockLogParser
	upstreamLogTailDone bool
	onExit              func(error)
	reload              reloader
	reopenLogs          logrotater.ReopenLogsFunc
}

func (tc *runnerTestCase) start() (wait func() command.CmdErr) {
	var waitGroup sync.WaitGroup
	var cmdErr command.CmdErr

	runnerDone := false
	onRunnerDone := func() {
		waitGroup.Done()
		runnerDone = true
	}

	waitGroup.Add(1)
	go func() {
		// Mock errors result in a call to runtime.Goexit
		// (stops execution, triggers deferred functions, is
		// not a panic)
		defer onRunnerDone()
		cmdErr = tc.runner.Run(Cmd(), tc.args)
	}()

	for !tc.runner.adminServerStarted && !runnerDone {
		time.Sleep(10 * time.Millisecond)
	}

	if !tc.runner.adminServerStarted {
		tc.t.Fatal("admin server didn't start (mock errors?)")
	}

	limit := 5 * time.Second
	start := time.Now()
	for (!tc.accessLogTailDone || !tc.upstreamLogTailDone) && time.Since(start) < limit {
		time.Sleep(10 * time.Millisecond)
	}

	return func() command.CmdErr {
		waitGroup.Wait()
		return cmdErr
	}
}

func mkMockRunner(t *testing.T, config *runnerConfig) (testcase *runnerTestCase) {
	ctrl := gomock.NewController(assert.Tracing(t))

	if config.args == nil {
		config.args = []string{}
	}

	testcase = &runnerTestCase{
		t:                   t,
		args:                config.args,
		accessLogTailDone:   true,
		upstreamLogTailDone: true,
	}

	recordOnExit := func(onExit func(error), args []string) {
		testcase.onExit = onExit
	}

	recordReload := func(reload reloader) {
		testcase.reload = reload
	}

	recordReopenLogs := func(_ *log.Logger, reopenLogs logrotater.ReopenLogsFunc) {
		testcase.reopenLogs = reopenLogs
	}

	source, err := metric.NewSource(logparser.DefaultSource(), "")
	if !assert.Nil(t, err) {
		t.FailNow()
	}

	mainFromFlags := NewMockFromFlags(ctrl)
	managedProc := proc.NewMockManagedProc(ctrl)
	nginxConfig := nginxconfig.NewMockNginxConfig(ctrl)

	apiConfigFromFlags := apiflags.NewMockAPIConfigFromFlags(ctrl)

	confAgentFromFlags := confagent.NewMockFromFlags(ctrl)
	confAgent := confagent.NewMockConfAgent(ctrl)

	executorFromFlags := executor.NewMockFromFlags(ctrl)
	executor := executor.NewMockExecutor(ctrl)

	statsClientFromFlags := client.NewMockFromFlags(ctrl)
	statsSvc := statsapi.NewMockStatsService(ctrl)
	stats := statsapi.AsStats(statsSvc, source.Source(), "executor")

	logRotaterFromFlags := logrotater.NewMockFromFlags(ctrl)
	logRotater := logrotater.NewMockLogRotater(ctrl)

	accessLogParserFromFlags := logparser.NewMockFromFlags(ctrl)
	accessLogParser := logparser.NewMockLogParser(ctrl)

	upstreamLogParserFromFlags := logparser.NewMockFromFlags(ctrl)
	upstreamLogParser := logparser.NewMockLogParser(ctrl)

	adminServerFromFlags := adminserver.NewMockFromFlags(ctrl)
	adminServer := adminserver.NewMockAdminServer(ctrl)

	runner := &runner{
		config:                  mainFromFlags,
		apiConfig:               apiConfigFromFlags,
		adminServerConfig:       adminServerFromFlags,
		confAgentConfig:         confAgentFromFlags,
		executorConfig:          executorFromFlags,
		statsClientConfig:       statsClientFromFlags,
		logRotaterConfig:        logRotaterFromFlags,
		accessLogParserConfig:   accessLogParserFromFlags,
		upstreamLogParserConfig: upstreamLogParserFromFlags,
	}

	testcase.ctrl = ctrl
	testcase.runner = runner
	testcase.adminServer = adminServer
	testcase.managedProc = managedProc
	testcase.accessLogParser = accessLogParser
	testcase.upstreamLogParser = upstreamLogParser

	calls := []*gomock.Call{}
	deferredCalls := []*gomock.Call{}
	defer func() {
		for i := len(deferredCalls) - 1; i >= 0; i-- {
			calls = append(calls, deferredCalls[i])
		}

		gomock.InOrder(calls...)
	}()

	calls = append(calls, mainFromFlags.EXPECT().Validate().Return(config.mainValidateErr))
	if config.mainValidateErr != nil {
		return
	}

	calls = append(
		calls,
		apiConfigFromFlags.EXPECT().Validate().Return(config.apiConfigValidateErr),
	)
	if config.apiConfigValidateErr != nil {
		return
	}

	calls = append(
		calls,
		confAgentFromFlags.EXPECT().Validate().Return(config.confAgentValidateErr),
	)
	if config.confAgentValidateErr != nil {
		return
	}

	calls = append(
		calls,
		adminServerFromFlags.EXPECT().Validate().Return(config.adminServerValidateErr),
	)
	if config.adminServerValidateErr != nil {
		return
	}

	calls = append(
		calls,
		logRotaterFromFlags.EXPECT().Validate().Return(config.logRotaterValidateErr),
	)
	if config.logRotaterValidateErr != nil {
		return
	}

	calls = append(
		calls,
		accessLogParserFromFlags.EXPECT().
			Validate().
			Return(config.accessLogParserValidateErr),
	)
	if config.accessLogParserValidateErr != nil {
		return
	}

	calls = append(
		calls,
		upstreamLogParserFromFlags.EXPECT().
			Validate().
			Return(config.upstreamLogParserValidateErr),
	)
	if config.upstreamLogParserValidateErr != nil {
		return
	}

	calls = append(
		calls,
		statsClientFromFlags.EXPECT().Validate().Return(config.statsClientValidateErr),
	)
	if config.statsClientValidateErr != nil {
		return
	}

	calls = append(
		calls,
		executorFromFlags.EXPECT().Make(gomock.Any()).Return(executor),
	)

	if config.statsClientMakeErr == nil {
		calls = append(
			calls,
			statsClientFromFlags.EXPECT().
				Make(executor, gomock.Any()).
				Return(statsSvc, nil),
			mainFromFlags.EXPECT().Source().Return(source),
			executor.EXPECT().SetStats(stats),
		)
	} else {
		calls = append(
			calls,
			statsClientFromFlags.EXPECT().
				Make(executor, gomock.Any()).
				Return(nil, config.statsClientMakeErr),
		)
		return
	}

	calls = append(
		calls,
		mainFromFlags.EXPECT().
			MakeNginxConfig(gomock.Any()).
			Do(recordReload).
			Return(nginxConfig),
	)

	if config.confAgentMakeErr == nil {
		calls = append(
			calls,
			confAgentFromFlags.EXPECT().Make(nginxConfig).Return(confAgent, nil),
		)
	} else {
		calls = append(
			calls,
			confAgentFromFlags.EXPECT().
				Make(nginxConfig).
				Return(nil, config.confAgentMakeErr),
		)
		return
	}

	paths := configwriter.Paths{AccessLog: "the-access-log", UpstreamLog: "the-upstream-log"}

	calls = append(
		calls,
		logRotaterFromFlags.EXPECT().
			Make(gomock.Any(), gomock.Any()).
			Do(recordReopenLogs).
			Return(logRotater),
		confAgent.EXPECT().GetPaths().Return(paths),
		logRotater.EXPECT().
			Start(paths.AccessLog).
			Return(config.accessLogStartRotateErr),
	)
	deferredCalls = append(deferredCalls, logRotater.EXPECT().StopAll())

	if config.accessLogStartRotateErr != nil {
		return
	}

	calls = append(
		calls,
		logRotater.EXPECT().
			Start(paths.UpstreamLog).
			Return(config.upstreamLogStartRotateErr),
	)

	if config.upstreamLogStartRotateErr != nil {
		return
	}

	calls = append(
		calls,
		mainFromFlags.EXPECT().
			MakeManagedProc(gomock.Any(), config.args).
			Do(recordOnExit).
			Return(managedProc),
		managedProc.EXPECT().Start().Return(config.procStartErr),
	)
	if config.procStartErr != nil {
		return
	}

	deferredCalls = append(deferredCalls, managedProc.EXPECT().Kill().Return(nil))

	// may be out of order
	confAgent.EXPECT().Poll().AnyTimes().Return(nil)

	if config.accessLogParserMakeErr == nil {
		calls = append(
			calls,
			accessLogParserFromFlags.EXPECT().
				Make(gomock.Any(), source).
				Return(accessLogParser, nil),
		)
	} else {
		calls = append(
			calls,
			accessLogParserFromFlags.EXPECT().
				Make(gomock.Any(), source).
				Return(nil, config.accessLogParserMakeErr),
		)
		return
	}

	deferredCalls = append(deferredCalls, accessLogParser.EXPECT().Close().Return(nil))

	// may be out of order
	testcase.accessLogTailDone = false
	accessLogParser.EXPECT().Tail(paths.AccessLog).Do(func(_ string) {
		testcase.accessLogTailDone = true
	}).Return(nil)

	if config.upstreamLogParserMakeErr == nil {
		calls = append(
			calls,
			upstreamLogParserFromFlags.EXPECT().
				Make(gomock.Any(), source).
				Return(upstreamLogParser, nil),
		)
	} else {
		calls = append(
			calls,
			upstreamLogParserFromFlags.EXPECT().
				Make(gomock.Any(), source).
				Return(nil, config.upstreamLogParserMakeErr),
		)
		return
	}

	deferredCalls = append(deferredCalls, upstreamLogParser.EXPECT().Close().Return(nil))

	// may be out of order
	testcase.upstreamLogTailDone = false
	upstreamLogParser.EXPECT().Tail(paths.UpstreamLog).Do(func(_ string) {
		testcase.upstreamLogTailDone = true
	}).Return(nil)

	calls = append(
		calls,
		adminServerFromFlags.EXPECT().Make(managedProc).Return(adminServer),
		adminServer.EXPECT().Start().Return(config.adminServerStartErr),
	)

	return
}

func testRunWithSignal(t *testing.T, adminCmd string, wrongProcErr bool) {
	test := mkMockRunner(t, &runnerConfig{args: []string{"a", "b", "c"}})

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

	wait := test.start()

	if wrongProcErr {
		test.onExit(errors.New("unexpected error"))
	} else {
		test.onExit(errors.New(exitError))
	}

	cmdErr := wait()

	if wrongProcErr {
		assert.Equal(t, cmdErr.Message, "nginx-admin: unexpected error")
	} else {
		assert.DeepEqual(t, cmdErr, command.NoError())
	}

	test.ctrl.Finish()
}

func TestCLI(t *testing.T) {
	assert.Nil(t, mkCLI().Validate())
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
	test := mkMockRunner(t, &runnerConfig{})

	test.adminServer.EXPECT().Close().Return(nil)

	wait := test.start()

	test.onExit(nil)

	cmdErr := wait()

	assert.DeepEqual(t, cmdErr, command.NoError())

	test.ctrl.Finish()
}

func TestRunProcExitsWithFailure(t *testing.T) {
	test := mkMockRunner(t, &runnerConfig{})

	test.adminServer.EXPECT().LastRequestedSignal().Return(adminserver.NoRequestedSignal)
	test.adminServer.EXPECT().Close().Return(nil)

	wait := test.start()

	test.onExit(errors.New("boom: exit status 1"))

	cmdErr := wait()

	assert.NotDeepEqual(t, cmdErr, command.NoError())
	assert.MatchesRegex(t, cmdErr.Message, "exit status 1")

	test.ctrl.Finish()
}

func TestRunProcExitsWithBadCommand(t *testing.T) {
	testConfig := &runnerConfig{
		args:         []string{"a", "b"},
		procStartErr: errors.New("file not found"),
	}
	test := mkMockRunner(t, testConfig)

	cmdErr := test.runner.Run(Cmd(), test.args)
	assert.NotDeepEqual(t, cmdErr, command.NoError())
	assert.MatchesRegex(t, cmdErr.Message, "not found")

	test.ctrl.Finish()
}

func TestRunProcExitsWithInvalidConfig(t *testing.T) {
	test := mkMockRunner(t, &runnerConfig{mainValidateErr: errors.New("boom")})
	cmdErr := test.runner.Run(Cmd(), test.args)
	assert.NotDeepEqual(t, cmdErr, command.NoError())
	assert.MatchesRegex(t, cmdErr.Message, "boom")

	test.ctrl.Finish()
}

func TestRunProcExitsWithInvalidApiConfig(t *testing.T) {
	test := mkMockRunner(t, &runnerConfig{apiConfigValidateErr: errors.New("boom")})
	cmdErr := test.runner.Run(Cmd(), test.args)
	assert.NotDeepEqual(t, cmdErr, command.NoError())
	assert.MatchesRegex(t, cmdErr.Message, "boom")

	test.ctrl.Finish()
}

func TestRunProcExitsWithInvalidConfAgent(t *testing.T) {
	test := mkMockRunner(t, &runnerConfig{confAgentValidateErr: errors.New("boom")})
	cmdErr := test.runner.Run(Cmd(), test.args)
	assert.NotDeepEqual(t, cmdErr, command.NoError())
	assert.MatchesRegex(t, cmdErr.Message, "boom")

	test.ctrl.Finish()
}

func TestRunProcExitsWithInvalidLogRotater(t *testing.T) {
	test := mkMockRunner(t, &runnerConfig{logRotaterValidateErr: errors.New("boom")})
	cmdErr := test.runner.Run(Cmd(), test.args)
	assert.NotDeepEqual(t, cmdErr, command.NoError())
	assert.MatchesRegex(t, cmdErr.Message, "boom")

	test.ctrl.Finish()
}

func TestRunProcExitsWithInvalidAdminServer(t *testing.T) {
	test := mkMockRunner(t, &runnerConfig{adminServerValidateErr: errors.New("boom")})
	cmdErr := test.runner.Run(Cmd(), test.args)
	assert.NotDeepEqual(t, cmdErr, command.NoError())
	assert.MatchesRegex(t, cmdErr.Message, "boom")

	test.ctrl.Finish()
}

func TestRunProcExistsWithInvalidAccessLogConfig(t *testing.T) {
	test := mkMockRunner(t, &runnerConfig{accessLogParserValidateErr: errors.New("boom")})
	cmdErr := test.runner.Run(Cmd(), test.args)
	assert.NotDeepEqual(t, cmdErr, command.NoError())
	assert.MatchesRegex(t, cmdErr.Message, "boom")

	test.ctrl.Finish()
}

func TestRunProcExistsWithInvalidUpstreamLogConfig(t *testing.T) {
	test := mkMockRunner(t, &runnerConfig{upstreamLogParserValidateErr: errors.New("boom")})
	cmdErr := test.runner.Run(Cmd(), test.args)
	assert.NotDeepEqual(t, cmdErr, command.NoError())
	assert.MatchesRegex(t, cmdErr.Message, "boom")

	test.ctrl.Finish()
}

func TestRunExitsWithConfAgentMakeError(t *testing.T) {
	test := mkMockRunner(t, &runnerConfig{confAgentMakeErr: errors.New("make error")})
	cmdErr := test.runner.Run(Cmd(), test.args)
	assert.NotDeepEqual(t, cmdErr, command.NoError())
	assert.MatchesRegex(t, cmdErr.Message, "make error")

	test.ctrl.Finish()
}

func TestRunExistsWithUnrotatableAccessLog(t *testing.T) {
	test := mkMockRunner(
		t,
		&runnerConfig{accessLogStartRotateErr: errors.New("rotate access log error")},
	)
	cmdErr := test.runner.Run(Cmd(), test.args)
	assert.NotDeepEqual(t, cmdErr, command.NoError())
	assert.MatchesRegex(t, cmdErr.Message, "rotate access log error")

	test.ctrl.Finish()

}

func TestRunExistsWithUnrotatableUpstreamLog(t *testing.T) {
	test := mkMockRunner(
		t,
		&runnerConfig{upstreamLogStartRotateErr: errors.New("rotate upstream log error")},
	)
	cmdErr := test.runner.Run(Cmd(), test.args)
	assert.NotDeepEqual(t, cmdErr, command.NoError())
	assert.MatchesRegex(t, cmdErr.Message, "rotate upstream log error")

	test.ctrl.Finish()

}

func TestRunExitsWithAccessLogParserMakeError(t *testing.T) {
	test := mkMockRunner(t, &runnerConfig{accessLogParserMakeErr: errors.New("make error")})
	cmdErr := test.runner.Run(Cmd(), test.args)
	assert.NotDeepEqual(t, cmdErr, command.NoError())
	assert.MatchesRegex(t, cmdErr.Message, "make error")

	test.ctrl.Finish()
}

func TestRunExitsWithUpstreamLogParserMakeError(t *testing.T) {
	test := mkMockRunner(t, &runnerConfig{upstreamLogParserMakeErr: errors.New("make error")})
	cmdErr := test.runner.Run(Cmd(), test.args)

	limit := 5 * time.Second
	start := time.Now()
	for !test.accessLogTailDone && time.Since(start) < limit {
		time.Sleep(10 * time.Millisecond)
	}

	assert.NotDeepEqual(t, cmdErr, command.NoError())
	assert.MatchesRegex(t, cmdErr.Message, "make error")

	test.ctrl.Finish()
}

func TestRunAdminServerFailsToStart(t *testing.T) {
	test := mkMockRunner(
		t,
		&runnerConfig{adminServerStartErr: errors.New("something something network")},
	)

	test.managedProc.EXPECT().Completed().Return(false)
	test.adminServer.EXPECT().Close().Return(nil)

	wait := test.start()

	test.onExit(nil) // process exits without error

	cmdErr := wait()

	assert.NotDeepEqual(t, cmdErr, command.NoError())
	assert.Equal(t, cmdErr.Message, "nginx-admin: something something network")

	test.ctrl.Finish()
}

func TestRunnerReload(t *testing.T) {
	test := mkMockRunner(t, &runnerConfig{args: []string{"a", "b", "c"}})

	wait := test.start()

	test.managedProc.EXPECT().Hangup().Return(nil)
	test.reload()

	test.adminServer.EXPECT().Close().Return(nil)
	test.onExit(nil)

	wait()

	test.ctrl.Finish()
}

func TestRunnerReopenLogs(t *testing.T) {
	test := mkMockRunner(t, &runnerConfig{args: []string{"a", "b", "c"}})

	wait := test.start()

	test.managedProc.EXPECT().Usr1().Return(nil)
	test.accessLogParser.EXPECT().Restart()
	test.upstreamLogParser.EXPECT().Restart()
	test.reopenLogs()

	test.adminServer.EXPECT().Close().Return(nil)
	test.onExit(nil)

	wait()

	test.ctrl.Finish()
}
