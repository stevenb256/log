package log

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
)

// strings
const strAssert string = "assert"
const strError string = "error"
const strDebug string = "debug"
const strInfo string = "info"
const strWarning string = "warning"

// OnLog callback any time something is being logged
type OnLog func(trace *Trace)

// Tag used to describe an object for tracing
type Tag struct {
	name  string
	value interface{}
}

// Tag quick create of tag structure
func tag(name string, value interface{}) *Tag {
	return &Tag{name: name, value: value}
}
func _f(format string, a ...interface{}) string {
	return fmt.Sprintf(format, a...)
}

// Dbg writes object to stdout
func Dbg(o interface{}) {
	fmt.Printf("%+v\n", o)
}

// used to hold source line and function name
type caller struct {
	Line     int
	Function string
	File     string
}

// Trace used to hold trace information
type Trace struct {
	Kind   string        `json:"kind"`
	Build  string        `json:"build"`
	Data   []interface{} `json:"data"`
	Stack  string        `json:"stack"`
	Time   time.Time     `json:"time"`
	Caller *caller       `json:"caller"`
	Error  error         `json:"error"`
}

// Log interal log structure
type Log struct {
	console bool
	onLog   OnLog
	build   string
	file    *os.File
	chTrace chan *Trace
	chExit  chan bool
}

// global log set once initialized
var _log *Log

// Check checks if err is a failure; if so logs and returns true; or false
func Check(err error, a ...interface{}) bool {
	if nil != err {
		if nil != _log {
			_log.chTrace <- &Trace{
				Time:   time.Now(),
				Kind:   strError,
				Build:  _log.build,
				Caller: getCaller(2),
				Stack:  stack(),
				Error:  err,
				Data:   a,
			}
		}
		return true
	}
	return false
}

// Fail checks if err is a failure; if so logs and returns true; or false
func Fail(err error, a ...interface{}) error {
	if nil != err {
		if nil != _log {
			_log.chTrace <- &Trace{
				Time:   time.Now(),
				Kind:   strError,
				Build:  _log.build,
				Caller: getCaller(2),
				Stack:  stack(),
				Error:  err,
				Data:   a,
			}
		}
	}
	return err
}

// Assert if condition is false; trace and panic
func Assert(condition bool, a ...interface{}) {
	if false == condition {
		t := &Trace{
			Time:   time.Now(),
			Kind:   strAssert,
			Build:  "<assert>",
			Caller: getCaller(2),
			Stack:  stack(),
			Data:   a,
		}
		if nil != _log {
			_log.close()
		}
		panic(t.asString()) // using nil prevents auto-restart from happening
	}
}

// Warning log a warning
func Warning(a ...interface{}) {
	if nil != _log {
		_log.chTrace <- &Trace{
			Time:   time.Now(),
			Kind:   strWarning,
			Build:  _log.build,
			Caller: getCaller(2),
			Data:   a,
		}
	}
}

// Info log info
func Info(a ...interface{}) {
	if nil != _log {
		_log.chTrace <- &Trace{
			Time:   time.Now(),
			Kind:   strInfo,
			Build:  _log.build,
			Caller: getCaller(2),
			Data:   a,
		}
	}
}

// Debug write a debug message
func Debug(a ...interface{}) {
	if nil != _log {
		_log.chTrace <- &Trace{
			Time:   time.Now(),
			Kind:   strDebug,
			Build:  _log.build,
			Caller: getCaller(2),
			Stack:  stack(),
			Data:   a,
		}
	}
}

// StartLog initiates and begins logging system
func StartLog(path, build string, console bool, onLog OnLog) error {
	if nil == _log {
		chError := make(chan error)
		_log = new(Log)
		go _log.run(path, build, console, onLog, chError)
		err := <-chError
		close(chError)
		if nil != err {
			_log.close()
			_log = nil
			return err
		}
	}
	return nil
}

// CloseLog shuts down and flushes log
func CloseLog() {
	if nil != _log {
		_log.close()
		_log = nil
	}
}

// starts log waiter and initializes stuff (runs on own routine)
func (l *Log) run(
	logFile, build string, console bool,
	onLog OnLog, chError chan error) {
	if "" != logFile {
		var err error
		l.file, err = os.OpenFile(
			logFile,
			os.O_RDWR|os.O_APPEND|os.O_CREATE|os.O_TRUNC,
			os.ModePerm)
		if nil != err {
			chError <- err
			return
		}
		defer l.file.Close()
	}
	chError <- nil
	l.chTrace = make(chan *Trace, 100)
	l.chExit = make(chan bool)
	l.build = build
	l.onLog = onLog
	l.console = console
	done := false
	for done == false {
		select {
		case trace := <-l.chTrace:
			l.write(trace)
		case <-l.chExit:
			done = true
		}
	}
}

// Close the logger
func (l *Log) close() {
	if nil != l.chTrace {
		l.chExit <- true
		// flush whatever is left
		done := false
		for done == false {
			select {
			case trace := <-l.chTrace:
				l.write(trace)
			default:
				done = true
			}
		}
		close(l.chExit)
		close(l.chTrace)
	}
	if nil != l.file {
		l.file.Close()
		l.file = nil
	}
	_log = nil
}

// writes trace info; don't use error handling functions in here
func (l *Log) write(trace *Trace) {
	if true == l.console || strDebug == trace.Kind {
		l.toConsole(trace)
	}
	if nil != l.file {
		l.file.WriteString(trace.asString())
		l.file.WriteString("\n")
	}
	if nil != l.onLog {
		l.onLog(trace)
	}
}

// write trace to console
func (l *Log) toConsole(trace *Trace) {
	if strDebug == trace.Kind {
		color.Set(color.FgHiMagenta)
	} else if strWarning == trace.Kind {
		color.Set(color.FgHiYellow)
	} else if strInfo == trace.Kind {
		color.Set(color.FgHiCyan)
	} else if strError == trace.Kind {
		color.Set(color.FgHiRed)
	}
	os.Stdout.WriteString(trace.asString())
	os.Stdout.WriteString("\n")
	color.Unset()
}

// convert trace to json
func (t *Trace) asJSON() string {
	s, err := json.MarshalIndent(t, " ", " ")
	if nil != err {
		return fmt.Sprintf("error-converting: %v", t)
	}
	return string(s)
}

// return trace as human understable string
func (t *Trace) asString() string {
	source := fmt.Sprintf("%s(%d): %s(%d): %s",
		t.Build, syscall.Getpid(),
		t.Caller.File, t.Caller.Line, t.Caller.Function)
	message := sliceAsString(t.Data)
	if nil != t.Error {
		message = fmt.Sprintf("%s: %s", t.Error.Error(), message)
	}
	return fmt.Sprintf("%02d/%02d/%04d %02d:%02d:%02d: [%s] %s: %s",
		t.Time.Month(), t.Time.Day(), t.Time.Year(),
		t.Time.Hour(), t.Time.Minute(), t.Time.Second(),
		t.Kind, source, message)
}

// gets full stack trace
func stack() string {
	buf := make([]byte, 1024)
	cb := runtime.Stack(buf, false)
	return string(buf[:cb])
}

// returns Caller stack, function, source, line information
func getCaller(level int) *caller {
	var function string
	pc, file, line, ok := runtime.Caller(level)
	if true == ok {
		details := runtime.FuncForPC(pc)
		if details != nil {
			names := strings.Split(details.Name(), ".")
			if 1 == len(names) {
				function = names[0]
			} else if 2 == len(names) {
				function = names[1]
			} else if 3 == len(names) {
				function = names[2]
			} else if 4 == len(names) {
				function = names[3]
			} else {
				function = details.Name()
			}
		}
	}
	return &caller{File: filepath.Base(file), Line: line, Function: function}
}

/* Code to create google logger

ctx := context.Background()
l.er, err = er.NewClient(
	ctx, ProjectID,
	er.Config{
		ServiceName:    glBuildInfo.Name(),
		ServiceVersion: Itoa(int64(glBuildInfo.Version)),
	},
	option.WithCredentialsFile(Join(*flagHome, "keys.ini")))
if nil != err {
	chError <- err
	return
}
defer l.er.Close()
l.lc, err = lr.NewClient(
	ctx, ProjectID,
	option.WithCredentialsFile(Join(*flagHome, "keys.ini")))
if nil != err {
	chError <- err
	return
}
defer l.lc.Close()
l.lr = l.lc.Logger("qloak")
if strError == trace.Kind {
	if nil != l.er {
		l.er.Report(er.Entry{Error: l.Error, Stack: l.Stack})
	}


	else if nil != l.lr {
		l.lr.log(lr.Entry{Severity: Sev(t), Payload: l.AsJson()})

		er      *er.Client
		lc      *lr.Client
		lr      *lr.Logger

		func sev(t *trace) lr.Severity {
			switch l.Kind {
			case stringError:
				return lr.Error
			case stringTrace:
				return lr.Info
			case stringAssert:
				return lr.Critical
			case stringDebug:
				return lr.Debug
			case stringWarning:
				return lr.Warning
			}
			return lr.Info
		}

*/
