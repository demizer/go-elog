// Copyright 2013 The go-logger Authors. All rights reserved.
// This code is MIT licensed. See the LICENSE file for more info.

package logger

import (
	"regexp"
)

// stripAnsi removes all ansi escapes from a string and returns the clean
// string.
func stripAnsi(text string) string {
	reg := regexp.MustCompile("\x1b\\[\\d+m")
	return reg.ReplaceAllString(text, "")
}