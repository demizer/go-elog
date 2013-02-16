// Copyright 2013 The go-logger Authors. All rights reserved.
// This code is MIT licensed. See the LICENSE file for more info.

// Package logger is a better logging system for Go than the generic log
// package in the Go Standard Library. The logger packages provides colored
// output, logging levels, custom log formatting, and simultaneous logging
// output stream to stdout, stderr, and os.File.
package logger

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"text/template"
	"time"
)

// Used for string output of the logging object
var levels = [5]string{
	"DEBUG",
	"INFO",
	"WARNING",
	"ERROR",
	"CRITICAL",
}

type level int

// Returns the string representation of the level
func (l level) String() string { return levels[l] }

// The DEBUG level is the lowest possible output level. This is meant for
// development use. The default output level is WARNING.
const (
	// DEBUG level messages should be used for development logging instead
	// of Printf calls. When used in this manner, instead of sprinkling
	// Printf calls everywhere and then having to remove them once the bug
	// is fixed, the developer can simply change to a higher logging level
	// and the debug messages will not be sent to the output stream.
	DEBUG level = iota
	// Info level messages should be used to convey more informative output
	// than debug that could be used by a user.
	INFO
	// Warning messages should be used to notify the user that something
	// worked, but the expected value was not the result.
	WARNING
	// Error messages should be used when something just did not work at
	// all.
	ERROR
	// Critical messages are used when something is completely broken and
	// unrecoverable. Critical messages are usually followed by os.Exit().
	CRITICAL
)

// These flags define which text to prefix to each log entry generated by the Logger.
const (
	// Bits or'ed together to control what's printed.
	Ldate = 1 << iota
	// full file name and line number: /a/b/c/d.go:23
	Llongfile
	// base file name and line number: d.go:23. overrides Llongfile
	Lshortfile
	// Use ansi escape sequences
	Lansi
	// initial values for the standard logger
	LstdFlags = Ldate | Lansi
)

const (
	defPrefix = ">>>"
)

var (
	defColorPrefix = AnsiEscape(BOLD, GREEN, ">>>", OFF)
	// std is the default logger object
	std = New(os.Stderr, defColorPrefix, time.RubyDate, logFormat, WARNING, LstdFlags)
)

// A Logger represents an active logging object that generates lines of output
// to an io.Writer. Each logging operation makes a single call to the Writer's
// Write method. A Logger can be used simultaneously from multiple goroutines;
// it guarantees to serialize access to the Writer.
type Logger struct {
	mu          sync.Mutex // Ensures atomic writes
	buf         []byte     // For marshaling output to write
	colors      bool       // Enable/Disable colored output
	dateFormat  string     // time.RubyDate is the default format
	flags       int        // Properties of the output
	level       level      // The default level is warning
	logTemplate string     // The format order of the output
	prefix      string     // Inserted into every logging output
	stream      io.Writer  // Destination for output
}

// formatOutput is used by Output() to apply the desired output format using
// the logTemplate. Using this template, an output string is built containing
// the desired structure such as prefix, date, and file + line number.
func (l *Logger) formatOutput(buf *[]byte, t time.Time, file string,
	line int, text string) {
	l.buf = append(l.buf, t.Format(l.dateFormat)...)
	if len(text) > 0 && text[len(text)-1] != '\n' {
		l.buf = append(l.buf, '\n')
	}
}

// Output is used by all of the logging functions to send output to the output
// stream.
//
// calldepth is the number of stack frames to skip when getting the file
// name of original calling function for file name output.
//
// text is the string to append to the assembled log format output.
//
// stream will be used as the output stream the text will be written to. If
// stream is nil, the stream value contained in the logger object is used.
func (l *Logger) Output(calldepth int,
	text string, stream io.Writer) (n int, err error) {
	now := time.Now()
	var file string
	var line int
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.flags&(Lshortfile|Llongfile) != 0 {
		// release lock while getting caller info - it's expensive.
		l.mu.Unlock()
		var ok bool
		_, file, line, ok = runtime.Caller(calldepth)
		if !ok {
			file = "???"
			line = 0
		}
		l.mu.Lock()
	}
	l.buf = l.buf[:0]
	l.formatOutput(&l.buf, now, file, line, text)
	if stream == nil {
		n, err = l.stream.Write(l.buf)
	} else {
		n, err = stream.Write(l.buf)
	}
	return int(n), err
}

// New creates a new logger object.
func New(stream io.Writer, prefix string, dateFormat string,
	logTemplate string, level level, flags int) *Logger {
	return &Logger{stream: stream, prefix: prefix, dateFormat: dateFormat,
		logTemplate: logTemplate, level: level, flags: flags}
}

// Level returns the logging level of the current logger object
func Level() level {
	return std.level
}

// SetLevel sets the logging output level for the standard logger.
func SetLevel(level level) {
	std.level = level
}

// Stream gets the output stream for the standard logger object.
func Stream() io.Writer {
	return std.stream
}

// SetStream sets the output stream for the standard logger object.
func SetStream(stream io.Writer) {
	std.stream = stream
}

// Prefix returns the output prefix.
func Prefix() string {
	return std.prefix
}

// SetPrefix set the output prefix.
func SetPrefix(prefix string) {
	std.prefix = prefix
}

// DateFormat returns the date format as a string.
func DateFormat() string {
	return std.dateFormat
}

// SetDateFormat sets the date format. See the time package on how to create a
// date format.
func SetDateFormat(format string) {
	std.dateFormat = format
}

// Flags returns the flags of the standard logger.
func Flags() int {
	return std.flags
}

// SetFlags sets the flags of the standard logging object.
func SetFlags(flags int) {
	std.flags = flags
}

// Print sends output to the standard logger output stream regardless of
// logging level including the logger format properties and flags. Spaces are
// added between operands when neither is a string. It returns the number of
// bytes written and any write error encountered.
func Print(v ...interface{}) (n int, err error) {
	return std.Output(2, fmt.Sprint(v...), os.Stdout)
}

// Println formats using the default formats for its operands and writes to
// standard output. Spaces are always added between operands and a newline is
// appended. It returns the number of bytes written and any write error
// encountered.
func Println(v ...interface{}) (n int, err error) {
	return std.Output(2, fmt.Sprintln(v...), os.Stdout)
}

// Printf formats according to a format specifier and writes to standard
// output. It returns the number of bytes written and any write error
// encountered.
func Printf(format string, v ...interface{}) (n int, err error) {
	return std.Output(2, fmt.Sprintf(format, v...), os.Stdout)
}
