package log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/google/uuid"
)

// strings
const strAssert string = "assert"
const strError string = "error"
const strDebug string = "debug"
const strInfo string = "info"
const strWarning string = "warning"

// Hex32 -special type to declare hex
type Hex32 uint32

// F - shortcut for fmt.Sprintf
func F(format string, a ...interface{}) string {
	return fmt.Sprintf(format, a...)
}

// used to hold source line and function name
type caller struct {
	line     int
	function string
	file     string
}

// Trace used to hold trace information
type trace struct {
	kind  string
	data  []interface{}
	stack string
	time  time.Time
	call  *caller
	trace bool
}

// global log set once initialized
var _logMux sync.Mutex
var _build string
var _file *os.File
var _consoleInfo bool
var _consoleTrace bool
var _currentDir string

// Check checks if err is a failure; if so logs and returns true; or false
func Check(a ...interface{}) bool {
	if nil != a[0] && nil != a[0].(error) {
		if false == isIgnored(a[0].(error)) {
			writeLog(&trace{
				time:  time.Now(),
				kind:  strError,
				call:  getCaller(2),
				stack: Stack(false),
				data:  a,
			})
		}
		return true
	}
	return false
}

// Fail checks if err is a failure; if so logs and returns true; or false
func Fail(a ...interface{}) error {
	writeLog(&trace{
		time:  time.Now(),
		kind:  strError,
		call:  getCaller(2),
		stack: Stack(false),
		data:  a,
	})
	if nil == a[0] {
		return nil
	}
	return a[0].(error)
}

// Assert if condition is false; trace and panic
func Assert(condition bool, a ...interface{}) {
	if false == condition {
		writeLog(&trace{
			time:  time.Now(),
			kind:  strAssert,
			call:  getCaller(2),
			stack: Stack(false),
			data:  a,
		})
		CloseLog()
		panic(nil) // using nil prevents auto-restart from happening
	}
}

// Warning log a warning
func Warning(a ...interface{}) {
	writeLog(&trace{
		time: time.Now(),
		kind: strWarning,
		call: getCaller(2),
		data: a,
	})
}

// Info log info
func Info(a ...interface{}) {
	writeLog(&trace{
		time: time.Now(),
		kind: strInfo,
		call: getCaller(2),
		data: a,
	})
}

// Debug write a debug message
func Debug(a ...interface{}) {
	writeLog(&trace{
		time:  time.Now(),
		kind:  strDebug,
		call:  getCaller(2),
		stack: Stack(false),
		data:  a,
	})
}

// Trace write a trace message
func Trace(a ...interface{}) {
	writeLog(&trace{
		time:  time.Now(),
		kind:  getStructName(a[0]),
		call:  getCaller(2),
		data:  a,
		trace: true,
	})
}

// StartLog initiates and begins logging system
func StartLog(logFile, build string, consoleInfo, consoleTrace bool) error {
	err := openLogFile(logFile)
	if nil != err {
		return err
	}
	_currentDir, _ = os.Getwd()
	_build = build
	_consoleInfo = consoleInfo
	_consoleTrace = consoleTrace
	return nil
}

// CloseLog shuts down and flushes log
func CloseLog() {
	_logMux.Lock()
	if nil != _file {
		_file.Close()
		_file = nil
	}
	_logMux.Unlock()
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
func writeLog(t *trace) {
	_logMux.Lock()
	if strDebug == t.kind || strAssert == t.kind || strError == t.kind || strWarning == t.kind {
		writeConsole(t)
	} else if true == _consoleInfo && strInfo == t.kind {
		writeConsole(t)
	} else if true == _consoleTrace && true == t.trace {
		writeConsole(t)
	}
	if nil != _file {
		writeFile(t)
	}
	_logMux.Unlock()
}

// writeFile - writes tab deliminted log file entry
func writeFile(t *trace) {
	dir, file := filepath.Split(t.call.file)
	fmt.Fprintf(_file, "[%s]\t%02d/%02d/%04d\t%02d:%02d:%02d\t%s:%d\t%s\t",
		t.kind,
		t.time.Month(), t.time.Day(), t.time.Year(),
		t.time.Hour(), t.time.Minute(), t.time.Second(),
		filepath.Base(dir)+"/"+file, t.call.line, t.call.function)
	writeFields(_file, t, "\t")
	fmt.Fprintf(_file, "\n")
}

// write trace to console
func writeConsole(t *trace) {
	if strDebug == t.kind {
		color.Set(color.FgHiMagenta)
	} else if strWarning == t.kind {
		color.Set(color.FgHiYellow)
	} else if strInfo == t.kind {
		color.Set(color.FgHiCyan)
	} else if strError == t.kind {
		color.Set(color.FgHiRed)
	} else if true == t.trace {
		color.Set(color.FgHiGreen)
	}
	fmt.Fprintf(os.Stdout, "[%s] %s:%d %s: ",
		t.kind,
		t.call.file, t.call.line, t.call.function)
	writeFields(os.Stdout, t, " ")
	fmt.Fprintf(os.Stdout, "\n")
	color.Unset()
}

// write field
func writeFields(w io.Writer, t *trace, delim string) {
	if nil != t.data {
		for i, o := range t.data {
			writeField(w, o, delim)
			if i < len(t.data)-1 {
				fmt.Fprintf(w, delim)
			}
		}
	}
}

// writeField - writes an object as its type to a stream
func writeField(w io.Writer, o interface{}, delim string) {
	if isInterfaceNil(o) {
		fmt.Fprintf(w, "<nil>")
		return
	}
	switch v := o.(type) {
	case error:
		fmt.Fprintf(w, "'error: %s'", v.Error())
	case string:
		if strings.Contains(v, " ") {
			fmt.Fprintf(w, "'%s'", v)
		} else {
			fmt.Fprintf(w, "%s", v)
		}
	case time.Time:
		fmt.Fprintf(w, v.Format(time.RFC1123))
	case int:
		fmt.Fprintf(w, "%d", v)
	case uint8:
		fmt.Fprintf(w, "%d", v)
	case uint32:
		fmt.Fprintf(w, "%d", v)
	case Hex32:
		fmt.Fprintf(w, "%#x", int64(v))
	case uint16:
		fmt.Fprintf(w, "%d", v)
	case uuid.UUID:
		fmt.Fprintf(w, "%s", v.String())
	case bool:
		fmt.Fprintf(w, "%t", v)
	case time.Duration:
		fmt.Fprintf(w, "%dms", v.Microseconds())
	default:
		writeUnknownField(w, v, delim)
	}
}

// returns struct field as a string
func writeUnknownField(w io.Writer, o interface{}, delim string) {
	if false == isInterfaceNil(o) {
		t, v := reflectDeref(o)
		if reflect.Struct == t.Kind() {
			for i := 0; i < t.NumField(); i++ {
				if v.Field(i).CanSet() {
					name, found := t.Field(i).Tag.Lookup("log")
					if true == found {
						if "hide" == name {
							continue
						} else if "" != name {
							fmt.Fprintf(w, "%s:", name)
						}
					} else {
						fmt.Fprintf(w, "%s:", t.Field(i).Name)
					}
					writeField(w, v.Field(i).Interface(), delim)
					if i < t.NumField()-1 {
						fmt.Fprint(w, delim)
					}
				}
			}
		} else {
			fmt.Fprintf(w, "%+v", o)
		}
	} else {
		fmt.Fprintf(w, "<nil>")
	}
}

// getObjectName - gets name of object at o if struct
func getStructName(o interface{}) string {
	if false == isInterfaceNil(o) {
		t, _ := reflectDeref(o)
		if reflect.Struct == t.Kind() {
			return t.Name()
		}
	}
	return "<unknown>"
}

// decides if an interface is nil
func isInterfaceNil(o interface{}) bool {
	if nil == o || (reflect.Ptr == reflect.TypeOf(o).Kind() && reflect.ValueOf(o).IsNil()) {
		return true
	}
	return false
}

// defers a point type value in reflection to get to pointed to item
func reflectDeref(obj interface{}) (reflect.Type, reflect.Value) {
	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)
	if reflect.Ptr == t.Kind() {
		t = t.Elem()
		v = v.Elem()
	}
	return t, v
}

// decides if object is a ptr and is nil
func isNil(v reflect.Value) bool {
	if reflect.Ptr == reflect.TypeOf(v).Kind() && reflect.ValueOf(v).IsNil() {
		return true
	}
	return false
}

// Stack full stack trace
func Stack(all bool) string {
	buf := make([]byte, 4096)
	cb := runtime.Stack(buf, all)
	return string(buf[:cb])
}

// returns Caller stack, function, source, line information
func getCaller(level int) *caller {
	var function string
	pc, file, line, ok := runtime.Caller(level)
	if true == ok {
		details := runtime.FuncForPC(pc)
		function = details.Name()
	}
	f := strings.Split(filepath.Base(function), ".")
	if _currentDir == filepath.Dir(file) {
		return &caller{file: "./" + filepath.Base(file), line: line, function: f[len(f)-1]}
	}
	return &caller{file: file, line: line, function: f[len(f)-1]}
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
