package logrotater

//go:generate mockgen -source $GOFILE -destination mock_$GOFILE -package $GOPACKAGE

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	tbnflag "github.com/turbinelabs/stdlib/flag"
	tbnos "github.com/turbinelabs/stdlib/os"
)

const suffixFormat = "20060102-150405"

// Function invoked after log rotation is completed. Typically used to
// reopen log files with the correct name after rotation.
type ReopenLogsFunc func() error

// FromFlags validates and constructs a LogRotater from command
// line flags.
type FromFlags interface {
	// Validates the LogRotater flags.
	Validate() error

	// Constructs a LogRotater with the given Logger and function
	// from command line flags. The ReopenLogsFunc function is
	// invoked after log rotation to allow logging process to
	// reopen its log files with the correct name.
	Make(*log.Logger, ReopenLogsFunc) LogRotater
}

// Rotates one or more log files, by path name, on a schedule.
type LogRotater interface {
	// Rotates the log file immediately, creates a new empty log
	// file, and rotates the log file on a configured schedule
	// thereafter. Returns an error if the log file could not be
	// rotated immediately.
	Start(pathname string) error

	// Stops all log rotation.
	StopAll()
}

// Constructs a new FromFlags from the given tbnflag.PrefixedFlagSet.
func NewFromFlags(flagset *tbnflag.PrefixedFlagSet) FromFlags {
	ff := &fromFlags{}

	flagset.DurationVar(
		&ff.frequency,
		"frequency",
		24*time.Hour,
		"Sets the frequency at which {{NAME}} are rotated. Minimum 1 minute.",
	)

	flagset.IntVar(
		&ff.keepCount,
		"keep",
		10,
		"Sets the number of old {{NAME}} kept after rotation. Minimum 1.",
	)

	return ff
}

type fromFlags struct {
	frequency time.Duration
	keepCount int
}

func (ff *fromFlags) Validate() error {
	if ff.frequency < time.Minute {
		return errors.New("log rotation frequency must be at least 1 minute")
	}

	if ff.keepCount < 1 {
		return errors.New("log rotation keep count must be at least 1")
	}

	return nil
}

func (ff *fromFlags) Make(logger *log.Logger, f ReopenLogsFunc) LogRotater {
	return &logRotater{
		logger:     logger,
		frequency:  ff.frequency,
		keepCount:  ff.keepCount,
		pathnames:  []string{},
		reopenLogs: f,
		os:         tbnos.New(),
		starter:    &sync.Once{},
		mutex:      &sync.Mutex{},
		quit:       make(chan bool, 1),
	}
}

type logRotater struct {
	logger    *log.Logger
	frequency time.Duration
	keepCount int
	pathnames []string

	reopenLogs ReopenLogsFunc

	os      tbnos.OS
	starter *sync.Once
	mutex   *sync.Mutex
	timer   *time.Timer
	quit    chan bool
	stopped bool
}

func (r *logRotater) addPath(pathname string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.pathnames = append(r.pathnames, pathname)
}

func (r *logRotater) copyPathnames() []string {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	copiedPathnames := make([]string, len(r.pathnames))
	copy(copiedPathnames, r.pathnames)
	return copiedPathnames
}

func (r *logRotater) Start(pathname string) error {
	if r.stopped {
		return errors.New("cannot restart stopped LogRotater")
	}

	r.logger.Printf("adding %s to log rotation", pathname)

	if err := r.rotateAndCleanup(pathname); err != nil {
		return err
	}

	r.addPath(pathname)
	r.starter.Do(func() { go r.rotateLoop() })
	return nil
}

func (r *logRotater) StopAll() {
	if r.timer != nil {
		r.timer.Stop()
	}
	r.quit <- true
}

func (r *logRotater) rotateLoop() {
	r.timer = time.NewTimer(r.delayForNext())

	for true {
		select {
		case <-r.timer.C:
			pathnames := r.copyPathnames()
			for _, pathname := range pathnames {
				r.rotateAndCleanup(pathname)
			}
			if err := r.reopenLogs(); err != nil {
				r.logger.Printf("failed to reopen logs: %s", err.Error())
			}
			r.timer.Reset(r.delayForNext())

		case <-r.quit:
			r.stopped = true
			return
		}
	}
}

func (r *logRotater) delayForNext() time.Duration {
	now := time.Now().UTC().UnixNano()

	offset := now % int64(r.frequency)
	return r.frequency - time.Duration(offset)
}

// calls rotate and then cleanup; fails only if rotate fails
func (r *logRotater) rotateAndCleanup(pathname string) error {
	rotated, err := r.rotate(pathname)
	if err != nil {
		r.logger.Printf(
			"error rotating %s: %s",
			pathname,
			err.Error(),
		)
		return err
	}

	if rotated {
		err := r.cleanup(pathname)
		if err != nil {
			r.logger.Printf(
				"error cleaning up %s: %s",
				pathname,
				err.Error(),
			)
		}
	}

	return nil
}

func (r *logRotater) rotate(pathname string) (bool, error) {
	fileInfo, err := r.os.Stat(pathname)
	if err != nil {
		if r.os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	if fileInfo.Size() == 0 {
		return false, nil
	}

	newPathname := rotatedName(pathname)
	err = r.os.Rename(pathname, newPathname)
	if err != nil {
		return false, err
	}

	// recreate log file
	file, err := r.os.OpenFile(pathname, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return true, err
	}
	defer file.Close()

	return true, nil
}

func rotatedName(pathname string) string {
	ext := path.Ext(pathname)
	pathname = pathname[0 : len(pathname)-len(ext)]
	now := time.Now().UTC().Format(suffixFormat)
	return fmt.Sprintf("%s.%s%s", pathname, now, ext)
}

func (r *logRotater) cleanup(pathname string) error {
	ext := path.Ext(pathname)
	pathname = pathname[0 : len(pathname)-len(ext)]
	basename := path.Base(pathname) + "."
	dirname := path.Dir(pathname)

	dirReader := r.os.NewDirReader(dirname)
	files, err := dirReader.Filter(func(fileInfo os.FileInfo) bool {
		if fileInfo.IsDir() {
			return false
		}

		thisName := fileInfo.Name()
		thisExt := path.Ext(thisName)
		if thisExt != ext {
			return false
		}

		thisBasename := thisName[0 : len(thisName)-len(thisExt)]
		if !strings.HasPrefix(thisBasename, basename) {
			return false
		}

		datePart := thisBasename[len(basename):]
		_, parseErr := time.Parse(suffixFormat, datePart)
		return parseErr == nil
	})
	if err != nil {
		return err
	}

	if len(files) <= r.keepCount {
		return nil
	}

	sort.Sort(sort.Reverse(sort.StringSlice(files)))

	err = nil
	for _, extra := range files[r.keepCount:] {
		if e := r.os.Remove(path.Join(dirname, extra)); e != nil && err == nil {
			err = e
		}
	}

	return err
}
