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

var (
	defPrefix      = ">>>"
	defColorPrefix = AnsiEscape(BOLD, GREEN, ">>>", OFF)
	// std is the default logger object
	log = New(os.Stderr, WARNING)
)

// A Logger represents an active logging object that generates lines of output
// to an io.Writer. Each logging operation makes a single call to the Writer's
// Write method. A Logger can be used simultaneously from multiple goroutines;
// it guarantees to serialize access to the Writer.
type Logger struct {
	mu         sync.Mutex         // Ensures atomic writes
	buf        []byte             // For marshaling output to write
	Colors     bool               // Enable/Disable colored output
	DateFormat string             // time.RubyDate is the default format
	Flags      int                // Properties of the output
	Level      level              // The default level is warning
	Template   *template.Template // The format order of the output
	Prefix     string             // Inserted into every logging output
	Stream     io.Writer          // Destination for output
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
func (l *Logger) Fprint(calldepth int,
	text string, stream io.Writer) (n int, err error) {
	now := time.Now()
	var file string
	var line int
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.Flags&(Lshortfile|Llongfile) != 0 {
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
		n, err = l.Stream.Write(l.buf)
	} else {
		n, err = stream.Write(l.buf)
	}
	return int(n), err
}

// Print sends output to the standard logger output stream regardless of
// logging level including the logger format properties and flags. Spaces are
// added between operands when neither is a string. It returns the number of
// bytes written and any write error encountered.
func (l *Logger) Print(v ...interface{}) (n int, err error) {
	return l.Fprint(2, fmt.Sprint(v...), os.Stdout)
}

// Println formats using the default formats for its operands and writes to
// standard output. Spaces are always added between operands and a newline is
// appended. It returns the number of bytes written and any write error
// encountered.
func (l *Logger) Println(v ...interface{}) (n int, err error) {
	return l.Fprint(2, fmt.Sprintln(v...), os.Stdout)
}

// Printf formats according to a format specifier and writes to standard
// output. It returns the number of bytes written and any write error
// encountered.
func (l *Logger) Printf(format string, v ...interface{}) (n int, err error) {
	return l.Fprint(2, fmt.Sprintf(format, v...), os.Stdout)
}

// New creates a new logger object and returns it.
func New(stream io.Writer, level level) (obj *Logger) {
	tmpl := template.Must(template.New("std").Funcs(funcMap).Parse(logFmt))
	obj = &Logger{Stream: stream, Colors: true, DateFormat: time.RubyDate,
		Flags: LstdFlags, Level: level, Template: tmpl,
		Prefix: defColorPrefix}
	return
}
