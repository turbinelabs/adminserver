package logrotater

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	tbnflag "github.com/turbinelabs/nonstdlib/flag"
	tbnos "github.com/turbinelabs/nonstdlib/os"
	"github.com/turbinelabs/test/assert"
	"github.com/turbinelabs/test/log"
	"github.com/turbinelabs/test/matcher"
	"github.com/turbinelabs/test/tempfile"
)

func TestNewFromFlags(t *testing.T) {
	underlying := flag.NewFlagSet("logrotater options", flag.PanicOnError)
	flagset := tbnflag.NewPrefixedFlagSet(underlying, "logrotate", "test")

	ff := NewFromFlags(flagset)
	ffImpl := ff.(*fromFlags)

	assert.Equal(t, ffImpl.frequency, 24*time.Hour)
	assert.Equal(t, ffImpl.keepCount, 10)

	flagset.Parse([]string{
		"-logrotate.frequency=5m",
		"-logrotate.keep=5",
	})

	assert.Equal(t, ffImpl.frequency, 5*time.Minute)
	assert.Equal(t, ffImpl.keepCount, 5)
}

func TestFromFlagsValidate(t *testing.T) {
	ff := &fromFlags{frequency: 59 * time.Second, keepCount: 0}
	assert.ErrorContains(t, ff.Validate(), "frequency must be at least 1 minute")

	ff.frequency = 1 * time.Minute
	assert.ErrorContains(t, ff.Validate(), "keep count must be at least 1")

	ff.keepCount = 1
	assert.Nil(t, ff.Validate())
}

func TestFromFlagsMake(t *testing.T) {
	ff := &fromFlags{frequency: 1 * time.Minute, keepCount: 2}
	assert.Nil(t, ff.Validate())

	logger := log.NewNoopLogger()
	reopener := func() error { return nil }

	lr := ff.Make(logger, reopener)
	lrImpl := lr.(*logRotater)
	assert.SameInstance(t, lrImpl.logger, logger)
	assert.Equal(t, lrImpl.frequency, ff.frequency)
	assert.Equal(t, lrImpl.keepCount, ff.keepCount)
	assert.DeepEqual(t, lrImpl.pathnames, []string{})
	assert.SameInstance(t, lrImpl.reopenLogs, ReopenLogsFunc(reopener))
	assert.NonNil(t, lrImpl.os)
	assert.NonNil(t, lrImpl.starter)
	assert.NonNil(t, lrImpl.mutex)
}

func TestRotatedName(t *testing.T) {
	names := [][]string{
		{"foo.log", `foo\.[0-9]{8}-[0-9]{6}\.log`},
		{"bar", `bar\.[0-9]{8}-[0-9]{6}`},
		{"foo.bar.qux.log", `foo\.bar\.qux\.[0-9]{8}-[0-9]{6}\.log`},
		{"/path/to/foo.log", `/path/to/foo\.[0-9]{8}-[0-9]{6}\.log`},
	}

	for _, testcases := range names {
		name := testcases[0]
		rotated := rotatedName(name)
		assert.MatchesRegex(t, rotated, "^"+testcases[1]+"$")
	}
}

type rotateTestCase struct {
	statErr           error
	statErrIsNotExist bool
	emptyFile         bool
	renameErr         error
	openFileErr       error
}

func (r rotateTestCase) run(t *testing.T) {
	ctrl := gomock.NewController(assert.Tracing(t))
	defer ctrl.Finish()

	pathname := "/foo/bar.log"

	osMock := tbnos.NewMockOS(ctrl)

	var rotatedPathnameCaptor *matcher.ValueCaptor
	var openFileCaptor *matcher.ValueCaptor

	expectedResult, expectedError, cleanup := func() (bool, error, func()) {
		if r.statErr != nil {
			osMock.EXPECT().Stat(pathname).Return(nil, r.statErr)
			osMock.EXPECT().IsNotExist(r.statErr).Return(r.statErrIsNotExist)

			if !r.statErrIsNotExist {
				return false, r.statErr, nil
			}
			return false, nil, nil
		}

		fileInfoMock := tbnos.NewMockFileInfo(ctrl)
		if r.emptyFile {
			fileInfoMock.EXPECT().Size().Return(int64(0))
		} else {
			fileInfoMock.EXPECT().Size().Return(int64(1000))
		}

		osMock.EXPECT().Stat(pathname).Return(fileInfoMock, nil)

		if r.emptyFile {
			return false, nil, nil
		}

		rotatedPathnameCaptor = matcher.CaptureAny()
		osMock.EXPECT().Rename(pathname, rotatedPathnameCaptor).Return(r.renameErr)
		if r.renameErr != nil {
			return false, r.renameErr, nil
		}

		openFileCaptor = matcher.CaptureAny()
		openFlags := os.O_CREATE | os.O_WRONLY | os.O_APPEND
		openFileMode := os.FileMode(0666)

		if r.openFileErr != nil {
			osMock.EXPECT().
				OpenFile(openFileCaptor, openFlags, openFileMode).
				Return(nil, r.openFileErr)
			return true, r.openFileErr, nil
		}

		tempFile, cleanup := tempfile.Make(t, "rotate")

		openedFile, err := os.OpenFile(tempFile, openFlags, openFileMode)
		if err != nil {
			cleanup()
			panic(err)
		}

		osMock.EXPECT().
			OpenFile(openFileCaptor, openFlags, openFileMode).
			Return(openedFile, nil)

		return true, nil, cleanup
	}()
	if cleanup != nil {
		defer cleanup()
	}

	lr := &logRotater{
		logger: log.NewNoopLogger(),
		os:     osMock,
	}

	rotated, err := lr.rotate(pathname)

	assert.Equal(t, rotated, expectedResult)
	assert.DeepEqual(t, err, expectedError)

	if rotatedPathnameCaptor != nil {
		assert.MatchesRegex(
			t,
			rotatedPathnameCaptor.V.(string),
			`^/foo/bar\.[0-9]{8}-[0-9]{6}\.log$`,
		)
	}

	if openFileCaptor != nil {
		assert.Equal(t, openFileCaptor.V, pathname)
	}

	ctrl.Finish()
}

func TestLogRotater(t *testing.T) {
	rotateTestCase{}.run(t)
}

func TestLogRotaterNoSuchFile(t *testing.T) {
	rotateTestCase{
		statErr:           errors.New("nope"),
		statErrIsNotExist: true,
	}.run(t)
}

func TestLogRotaterStatError(t *testing.T) {
	rotateTestCase{
		statErr:           errors.New("nope"),
		statErrIsNotExist: false,
	}.run(t)
}

func TestLogRotaterEmptyFile(t *testing.T) {
	rotateTestCase{
		emptyFile: true,
	}.run(t)
}

func TestLogRotaterRenameError(t *testing.T) {
	rotateTestCase{
		renameErr: errors.New("nope"),
	}.run(t)
}

func TestLogRotaterCreateFileError(t *testing.T) {
	rotateTestCase{
		openFileErr: errors.New("nope"),
	}.run(t)
}

func TestLogRotaterAddPath(t *testing.T) {
	lr := &logRotater{
		pathnames: []string{},
		mutex:     &sync.Mutex{},
	}

	for v := 1; v <= 10; v++ {
		lr.addPath(fmt.Sprintf("foo-%d", v))
	}

	assert.Equal(t, len(lr.pathnames), 10)
	assert.Equal(t, lr.pathnames[0], "foo-1")
	assert.Equal(t, lr.pathnames[9], "foo-10")
}

func TestLogRotaterCopyPathnames(t *testing.T) {
	lr := &logRotater{
		pathnames: []string{"foo", "bar"},
		mutex:     &sync.Mutex{},
	}

	copied := lr.copyPathnames()
	assert.DeepEqual(t, copied, []string{"foo", "bar"})
	assert.NotSameInstance(t, copied, lr.pathnames)
}

func TestLogRotaterDelayForNext(t *testing.T) {
	testcases := []time.Duration{
		1 * time.Minute,
		10 * time.Minute,
		1 * time.Hour,
		24 * time.Hour,
	}

	for _, f := range testcases {
		lr := &logRotater{
			frequency: f,
		}

		delay := lr.delayForNext()
		assert.True(t, delay <= f)
		assert.True(t, delay >= 0)
	}
}

type cleanupTestCase struct {
	keepCount   int
	dirname     string
	filename    string
	dirReadErr  error
	files       []string
	removeFiles []string
	removeErrs  []error

	testDirEntryFilter func(tbnos.DirEntryFilter)
}

func (c cleanupTestCase) run(t *testing.T) {
	ctrl := gomock.NewController(assert.Tracing(t))
	defer ctrl.Finish()

	pathname := path.Join(c.dirname, c.filename)

	var filterCaptor *matcher.ValueCaptor

	osMock := tbnos.NewMockOS(ctrl)
	dirReaderMock := tbnos.NewMockDirReader(ctrl)

	expectedError := func() error {
		osMock.EXPECT().NewDirReader(c.dirname).Return(dirReaderMock)

		if c.dirReadErr != nil {
			dirReaderMock.EXPECT().Filter(gomock.Any()).Return(nil, c.dirReadErr)
			return c.dirReadErr
		}

		filterCaptor = matcher.CaptureAny()
		dirReaderMock.EXPECT().Filter(filterCaptor).Return(c.files, nil)

		if len(c.removeFiles) == 0 {
			return nil
		}

		if len(c.removeFiles) != len(c.removeErrs) {
			panic("bad config: needs matching remove files and errors")
		}

		removeCalls := []*gomock.Call{}
		var firstError error
		for i := range c.removeFiles {
			file := c.removeFiles[i]
			err := c.removeErrs[i]
			if firstError == nil {
				firstError = err
			}

			removeCalls = append(
				removeCalls,
				osMock.EXPECT().Remove(path.Join(c.dirname, file)).Return(err),
			)
		}

		gomock.InOrder(removeCalls...)

		return firstError
	}()

	lr := &logRotater{
		keepCount: c.keepCount,
		logger:    log.NewNoopLogger(),
		os:        osMock,
	}

	err := lr.cleanup(pathname)
	assert.DeepEqual(t, err, expectedError)

	if c.testDirEntryFilter != nil {
		if filterCaptor == nil {
			assert.Failed(t, "did not capture filter function, cannot test it")
		} else {
			c.testDirEntryFilter(filterCaptor.V.(tbnos.DirEntryFilter))
		}
	}

	ctrl.Finish()
}

func TestLogRotaterCleanup(t *testing.T) {
	cleanupTestCase{
		keepCount: 1,
		dirname:   "/foo",
		filename:  "bar.log",
		files: []string{
			"bar.101.log",
			"bar.102.log",
			"bar.100.log",
		},
		removeFiles: []string{
			"bar.101.log",
			"bar.100.log",
		},
		removeErrs: []error{
			nil,
			nil,
		},
	}.run(t)
}

func TestLogRotaterCleanupDirError(t *testing.T) {
	cleanupTestCase{
		keepCount:  1,
		dirname:    "/foo",
		filename:   "bar.log",
		dirReadErr: errors.New("boom"),
	}.run(t)
}

func TestLogRotaterCleanupTooFewFiles(t *testing.T) {
	cleanupTestCase{
		keepCount: 1,
		dirname:   "/foo",
		filename:  "bar.log",
		files:     []string{"bar.100.log"},
	}.run(t)
}

func TestLogRotaterCleanupRemoveError(t *testing.T) {
	cleanupTestCase{
		keepCount: 1,
		dirname:   "/foo",
		filename:  "bar.log",
		files: []string{
			"bar.100.log",
			"bar.101.log",
		},
		removeFiles: []string{
			"bar.100.log",
		},
		removeErrs: []error{errors.New("boom")},
	}.run(t)
}

func TestLogRotaterCleanupRemoveErrorReturnsFirst(t *testing.T) {
	cleanupTestCase{
		keepCount: 1,
		dirname:   "/foo",
		filename:  "bar.log",
		files: []string{
			"bar.100.log",
			"bar.101.log",
			"bar.102.log",
			"bar.103.log",
		},
		removeFiles: []string{
			"bar.102.log",
			"bar.101.log",
			"bar.100.log",
		},
		removeErrs: []error{
			nil,
			errors.New("boom1"),
			errors.New("boom2"),
		},
	}.run(t)
}

func TestLogRotaterCleanupDirEntryFilter(t *testing.T) {
	ctrl := gomock.NewController(assert.Tracing(t))
	defer ctrl.Finish()

	mockFileInfo := func(isDir bool, name string) *tbnos.MockFileInfo {
		mock := tbnos.NewMockFileInfo(ctrl)
		mock.EXPECT().IsDir().Return(isDir)
		if !isDir {
			mock.EXPECT().Name().Return(name)
		}
		return mock
	}

	filterRejects := []os.FileInfo{
		mockFileInfo(true, "ignored"),
		mockFileInfo(false, "something.txt"),
		mockFileInfo(false, "something.log"),
		mockFileInfo(false, "bar.qux.NOTADATE.log"),
		mockFileInfo(false, "bar.qux.2016.log"),
	}

	filterAccepts := []os.FileInfo{
		mockFileInfo(false, "bar.qux.20160101-000000.log"),
		mockFileInfo(false, "bar.qux.20160101-235959.log"),
	}

	cleanupTestCase{
		keepCount: 1,
		dirname:   "/foo",
		filename:  "bar.qux.log",
		files:     []string{"bar.qux.100.log"},
		testDirEntryFilter: func(f tbnos.DirEntryFilter) {
			for _, reject := range filterRejects {
				assert.False(t, f(reject))
			}

			for _, accept := range filterAccepts {
				assert.True(t, f(accept))
			}
		},
	}.run(t)
}

func prepRotateLoopTest(
	t *testing.T,
	ctrl *gomock.Controller,
	lr *logRotater,
	pathname string,
	triggerQuit bool,
) *logRotater {
	var osMock *tbnos.MockOS
	if lr == nil {
		osMock = tbnos.NewMockOS(ctrl)

		lr = &logRotater{
			logger:     log.NewNoopLogger(),
			frequency:  100 * time.Millisecond,
			keepCount:  10,
			pathnames:  []string{pathname},
			reopenLogs: func() error { return nil },
			mutex:      &sync.Mutex{},
			os:         osMock,
			quit:       make(chan bool, 1),
		}
	} else {
		osMock = lr.os.(*tbnos.MockOS)
	}

	dirname := path.Dir(pathname)
	fileInfoMock := tbnos.NewMockFileInfo(ctrl)
	dirReaderMock := tbnos.NewMockDirReader(ctrl)

	// queue up a quit message on this first expected call to insure a single loop
	call := fileInfoMock.EXPECT().Size().Return(int64(1000))

	if triggerQuit {
		call.Do(func() { lr.quit <- true })
	}

	osMock.EXPECT().Stat(pathname).Return(fileInfoMock, nil)
	osMock.EXPECT().Rename(pathname, gomock.Any()).Return(nil)

	tempFile, cleanup := tempfile.Make(t, "rotate")
	defer cleanup()

	openFlags := os.O_CREATE | os.O_WRONLY | os.O_APPEND
	openFileMode := os.FileMode(0666)
	openedFile, err := os.OpenFile(tempFile, openFlags, openFileMode)
	if err != nil {
		panic(err)
	}

	osMock.EXPECT().
		OpenFile(gomock.Any(), openFlags, openFileMode).
		Return(openedFile, nil)

	osMock.EXPECT().NewDirReader(dirname).Return(dirReaderMock)

	dirReaderMock.EXPECT().
		Filter(gomock.Any()).
		Return([]string{}, nil)

	return lr
}

func TestLogRotaterRotateLoop(t *testing.T) {
	ctrl := gomock.NewController(assert.Tracing(t))
	defer ctrl.Finish()

	lr := prepRotateLoopTest(t, ctrl, nil, "/foo/bar.log", true)

	reopenCalls := 0
	lr.reopenLogs = func() error {
		reopenCalls++
		return nil
	}

	lr.rotateLoop()

	assert.Equal(t, reopenCalls, 1)
}

func TestLogRotaterStart(t *testing.T) {
	ctrl := gomock.NewController(assert.Tracing(t))
	defer ctrl.Finish()

	pathname := "/foo/bar.log"
	lr := prepRotateLoopTest(t, ctrl, nil, pathname, true)

	lr.starter = &sync.Once{}

	assert.Nil(t, lr.Start(pathname))

	for !lr.stopped {
		time.Sleep(100 * time.Millisecond)
	}
}

func TestLogRotaterStartError(t *testing.T) {
	ctrl := gomock.NewController(assert.Tracing(t))
	defer ctrl.Finish()

	pathname := "/foo/bar.log"
	osMock := tbnos.NewMockOS(ctrl)

	lr := &logRotater{
		logger:    log.NewNoopLogger(),
		frequency: 100 * time.Millisecond,
		keepCount: 10,
		pathnames: []string{pathname},
		mutex:     &sync.Mutex{},
		os:        osMock,
		quit:      make(chan bool, 1),
	}

	err := errors.New("rotation failed")

	osMock.EXPECT().Stat(pathname).Return(nil, err)
	osMock.EXPECT().IsNotExist(err).Return(false)

	assert.DeepEqual(t, lr.Start(pathname), err)
}

func TestLogRotaterMultipleStarts(t *testing.T) {
	ctrl := gomock.NewController(assert.Tracing(t))
	defer ctrl.Finish()

	pathname1 := "/foo/bar.log"
	pathname2 := "/foo/qux.log"
	lr := prepRotateLoopTest(t, ctrl, nil, pathname1, false)
	prepRotateLoopTest(t, ctrl, lr, pathname2, true)

	lr.pathnames = []string{}
	lr.frequency = time.Minute
	lr.starter = &sync.Once{}

	assert.Nil(t, lr.Start(pathname1))
	assert.False(t, lr.stopped)
	assert.Nil(t, lr.Start(pathname2))
	assert.DeepEqual(t, lr.pathnames, []string{pathname1, pathname2})

	for !lr.stopped {
		time.Sleep(100 * time.Millisecond)
	}
}

func TestLogRotaterStopAll(t *testing.T) {
	lr := &logRotater{
		timer: time.NewTimer(time.Minute),
		quit:  make(chan bool, 1),
	}

	lr.StopAll()

	assert.Equal(t, len(lr.quit), 1)

	// timer.Reset returns true if timer was active (e.g. not stopped)
	assert.False(t, lr.timer.Reset(time.Minute))
}

func TestLogRotaterStopAllOnUnstarted(t *testing.T) {
	lr := &logRotater{
		quit: make(chan bool, 1),
	}

	lr.StopAll()

	assert.Equal(t, len(lr.quit), 1)
}

func TestLogRotaterRestart(t *testing.T) {
	lr := &logRotater{
		stopped: true,
	}

	assert.ErrorContains(t, lr.Start("/foo/bar.log"), "cannot restart")
}
