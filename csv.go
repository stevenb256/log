package log

import (
	"fmt"
	"os"
	"time"
)

// CSVLog - used to hold a csv log file
type CSVLog struct {
	file    *os.File
	chWrite chan []interface{}
}

// OpenCSV - opens a CSV file used to detailed logs
func OpenCSV(path string, headers []interface{}) (*CSVLog, error) {
	var err error
	c := new(CSVLog)
	c.file, err = os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if Check(err) {
		return nil, err
	}
	c.chWrite = make(chan []interface{})
	go c.logRoutine()
	c.Write(headers)
	return c, err
}

// Close the csv log
func (c *CSVLog) Close() {
	if nil == c.chWrite {
		close(c.chWrite)
		c.chWrite = nil
	}
	if nil != c.file {
		c.file.Close()
		c.file = nil
	}
}

// Write - writes a row to the csv file
func (c *CSVLog) Write(row []interface{}) {
	if nil != c.chWrite {
		c.chWrite <- row
	}
}

// csv logRoutine
func (c *CSVLog) logRoutine() {
	for {
		row, more := <-c.chWrite
		if false == more {
			break
		}
		c.write(row)
	}
}

// WriteCSV - writes a row to the csv file
func (c *CSVLog) write(values []interface{}) {
	for i, o := range values {
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
			c.file.WriteString(fmt.Sprintf("0x%x", v))
		case uint16:
			c.file.WriteString(fmt.Sprintf("%d", v))
		case time.Duration:
			c.file.WriteString(fmt.Sprintf("%dms", v.Milliseconds()))
		default:
			Assert(false, v)
		}
		if i < len(values)-1 {
			c.file.WriteString("\t")
		}
	}
	c.file.WriteString("\n")
}
