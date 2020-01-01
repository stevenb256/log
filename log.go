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

// Tag quick create of tag structure
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

// global log set once initialized
var _build string
var _chTrace chan *Trace
var _file *os.File
var _onLog OnLog
var _console bool

// Check checks if err is a failure; if so logs and returns true; or false
func Check(err error, a ...interface{}) bool {
	if nil != err {
		_chTrace <- &Trace{
			Time:   time.Now(),
			Kind:   strError,
			Build:  _build,
			Caller: getCaller(2),
			Stack:  stack(),
			Error:  err,
			Data:   a,
		}
		return true
	}
	return false
}

// Fail checks if err is a failure; if so logs and returns true; or false
func Fail(err error, a ...interface{}) error {
	if nil != err {
		_chTrace <- &Trace{
			Time:   time.Now(),
			Kind:   strError,
			Build:  _build,
			Caller: getCaller(2),
			Stack:  stack(),
			Error:  err,
			Data:   a,
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
		CloseLog()
		panic(t.asString()) // using nil prevents auto-restart from happening
	}
}

// Warning log a warning
func Warning(a ...interface{}) {
	_chTrace <- &Trace{
		Time:   time.Now(),
		Kind:   strWarning,
		Build:  _build,
		Caller: getCaller(2),
		Data:   a,
	}
}

// Info log info
func Info(a ...interface{}) {
	_chTrace <- &Trace{
		Time:   time.Now(),
		Kind:   strInfo,
		Build:  _build,
		Caller: getCaller(2),
		Data:   a,
	}
}

// Debug write a debug message
func Debug(a ...interface{}) {
	_chTrace <- &Trace{
		Time:   time.Now(),
		Kind:   strDebug,
		Build:  _build,
		Caller: getCaller(2),
		Stack:  stack(),
		Data:   a,
	}
}

// StartLog initiates and begins logging system
func StartLog(logFile, build string, console bool, onLog OnLog) error {

	// if trace already allocated exit
	if nil != _chTrace {
		return nil
	}

	// open log file
	err := openLogFile(logFile)
	if nil != err {
		return err
	}

	// create globals
	_chTrace = make(chan *Trace, 100)
	_build = build
	_onLog = onLog
	_console = console

	// run log routine
	go logRoutine(build, console, onLog)

	// done
	return nil
}

// starts log waiter and initializes stuff (runs on own routine)
// assumes _mux.Lock is called already
func logRoutine(build string, console bool, onLog OnLog) {
	for {
		trace, more := <-_chTrace
		if false == more {
			break
		}
		writeLog(trace)
	}
}

// CloseLog shuts down and flushes log
func CloseLog() {
	if nil != _chTrace {
		close(_chTrace)
		_chTrace = nil
	}
	if nil != _file {
		_file.Close()
		_file = nil
	}
}

// open log file; assume _mux taken
func openLogFile(logFile string) error {
	var err error
	if "" != logFile {
		_file, err = os.OpenFile(logFile,
			os.O_RDWR|os.O_APPEND|os.O_CREATE|os.O_TRUNC,
			os.ModePerm)
	}
	return err
}

// writes trace info; don't use error handling functions in here
// assumes _mux.Lock taken
func writeLog(trace *Trace) {
	if true == _console || strDebug == trace.Kind {
		writeConsole(trace)
	}
	if nil != _file {
		_file.WriteString(trace.asString())
		_file.WriteString("\n")
	}
	if nil != _onLog {
		_onLog(trace)
	}
}

// write trace to console
func writeConsole(trace *Trace) {
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
