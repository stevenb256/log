package log

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// CSVLog - used to hold a csv log file
type CSVLog struct {
	mux  sync.Mutex
	file *os.File
}

// OpenCSV - opens a CSV file used to detailed logs
func OpenCSV(path string, headers []interface{}) (*CSVLog, error) {
	var err error
	c := new(CSVLog)
	c.file, err = os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if Check(err) {
		return nil, err
	}
	c.Write(headers)
	return c, err
}

// Close the csv log
func (c *CSVLog) Close() {
	c.mux.Lock()
	if nil != c.file {
		c.file.Close()
		c.file = nil
	}
	c.mux.Unlock()
}

// Write - writes a row to the csv file
func (c *CSVLog) Write(row []interface{}) {
	c.mux.Lock()
	if nil != c.file {
		for i, o := range row {
			switch v := o.(type) {
			case string:
				c.file.WriteString(v)
			case int:
				c.file.WriteString(fmt.Sprintf("%d", v))
			case uint8:
				c.file.WriteString(fmt.Sprintf("%d", v))
			case uint32:
				c.file.WriteString(fmt.Sprintf("%d", v))
			case Hex32:
				c.file.WriteString(fmt.Sprintf("%#x", int64(v)))
			case uint16:
				c.file.WriteString(fmt.Sprintf("%d", v))
			case bool:
				c.file.WriteString(fmt.Sprintf("%t", v))
			case time.Duration:
				c.file.WriteString(fmt.Sprintf("%d ms", v.Milliseconds()))
			default:
				Assert(false, v)
			}
			if i < len(row)-1 {
				c.file.WriteString("\t")
			}
		}
		c.file.WriteString("\n")
	}
	c.mux.Unlock()
}
