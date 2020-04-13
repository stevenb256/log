package log

import (
	"fmt"
	"io"
	"net"
	"strconv"
)

// ErrInvalidArg generic error for invalid arguments
var ErrInvalidArg = NewError(100, "generic", "invalid argument")

// ErrClosed - generic error for a closed resource, like a file or network
var ErrClosed = NewError(101, "gen", "resource closed")

// Error used to hold error and error descriptoin
type Error struct {
	error
	Facility string
	Code     int
	Message  string
}

// returns message of the error
func (e *Error) Error() string {
	return e.Message
}

// list of errors to be ignored
var _ignoredErrors = map[error]bool{
	io.EOF:    true,
	ErrClosed: true,
	//ErrTimeout:      true,
	//ErrNotConnected: true,
	//ErrCancelled:    true,
}

// isIgnored
func isIgnored(err error) bool {
	return _ignoredErrors[MapError(err)]
}

// list of all errors that have been registered
var _errorTable = make(map[string]*Error)

// NewError used to register a new error e.g. socket-100: message
func NewError(code int, facility, msg string) error {
	e := new(Error)
	e.Code = code
	e.Facility = facility
	prefix := facility + "-" + strconv.FormatInt(int64(code), 10)
	e.Message = prefix + ": " + msg
	if nil != _errorTable[prefix] {
		panic(_f("error already registered: %s", prefix))
	}
	_errorTable[prefix] = e
	return e
}

// MapError tries to map a generic err into a registered error
func MapError(err interface{}) error {
	switch val := err.(type) {
	case *net.OpError:
		if val.Err.Error() == "use of closed network connection" {
			return ErrClosed
		}
	case *Error:
		return val
	case string:
		var e Error
		fmt.Sscanf(val, "%s-%d: %s", &e.Facility, &e.Code, &e.Message)
		return &e
	}
	return &Error{Message: fmt.Sprintf("%v", err)}
}

/*
func maperror(err error) error {
	if nil != err {
		switch v := err.(type) {
		case sl.Error:
			switch v.Code {
			case sl.ErrConstraint:
				err = ErrDuplicate
			case sl.ErrError:
				err = ErrNotFound
			case sl.ErrBusy:
				err = fail(ErrDatabaseLocked)
			default:
				trace(fmt.Sprintf("sqlite error: %d", v.Code))
			}
		case *net.OpError:
			//debug("*net.OpError")
		case *googleapi.Error:
			//debug("googleapi.Error")
		default:
			//t := reflect.TypeOf(v)
			//if reflect.Ptr == l.Kind() {
			//	t = l.Elem()
			//}
			//debug(l.Kind().String(), l.Name())
		}
		s := err.Error()
		if strings.Contains(s, "timeout") {
			err = ErrTimeout
		} else if strings.Contains(s, "closed") {
			err = ErrNotConnected
		} else if strings.Contains(s, "context canceled") {
			err = ErrCancelled
		} else if strings.Contains(s, "no route to host") {
			err = ErrNotConnected
		} else if strings.Contains(s, "no such host") {
			err = ErrNotConnected
		} else if strings.Contains(s, "connection refused") {
			err = ErrNotConnected
		}
	}
	return err
}
*/
