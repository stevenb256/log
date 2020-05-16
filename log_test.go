package log

import (
	"os"
	"testing"
	"time"
)

// ErrTest1
var errTest1 = NewError(1000, "logtest", "test1 error")
var errTest2 = NewError(1001, "logtest", "test2 error")

// complex struct to use on reflection
type complexTest struct {
	T    string `log:"t-string" json:"t-string"`
	x    time.Time
	f    bool
	i64  int64
	ui64 uint64
	b    []byte
	Rgi  [3]int     `log:"int-array"`
	Y    *time.Time `log:"y-time"`
	c    *map[string]string
}

// TestLog tests logging
func TestLog(t *testing.T) {

	// test struct
	//ct := &complexTest{T: "test", x: time.Now(), b: []byte("123"), c: new(map[string]string)}

	// test vars
	int1 := 1
	int2 := 2

	// fail to start log
	err := StartLog("/foobar/test.log", "1.0", true, true)
	if nil == err {
		panic("should have failed to start log with invalid path\n")
	}

	// start log with correct information
	err = StartLog("testlog.log", "1.0", true, true)
	if nil != err {
		panic("failed to create log when it should have succeeded\n")
	}
	defer os.Remove("testlog.log")

	// test check
	f := Check(errTest1)
	if false == f {
		panic(nil)
	} else if true != f {
		panic(nil)
	}

	// fabricate a test trace
	trace := &trace{
		time:  time.Now(),
		kind:  strError,
		call:  getCaller(1),
		stack: "",
		data:  []interface{}{errTest1, nil},
		//	Data:   []interface{}{int1, int2, ct},
	}

	// get as string
	writeLog(trace)
	//writeConsole(trace)

	// print a warning
	Trace(&complexTest{T: "test-string"})
	Fail(errTest1, "test fail", int1, int2)
	Warning(errTest1, "test warning", int1, int2)
	Debug(errTest1, "test debug", int1, int2)
	Info(errTest1, "test info", int1, int2)
	Assert(true, errTest1, "test info", int1, int2)

	// Close the log file
	CloseLog()
}
