package jenkins

import (
	"regexp"
	"strings"
)

// Jenkins' pipeline log emits fold markers as ANSI "conceal" (SGR 8) spans
// carrying base64 blobs that terminals are expected to hide. Bubbletea's
// viewport doesn't honor conceal, so drop those spans outright.
var concealSpan = regexp.MustCompile(`\x1b\[8m.*?\x1b\[0m`)

// CSI sequences whose final byte isn't 'm' are cursor movement / screen
// control (erase line, move cursor, etc.) rather than color — those would
// corrupt rendering inside a text buffer that isn't a real terminal, so
// strip them. SGR sequences (ending in 'm') are left alone: Jenkins' own
// ansiColor plugin uses them, and lipgloss renders them fine.
var nonSGRCsi = regexp.MustCompile(`\x1b\[[0-9;]*[^0-9;m]`)

// CleanLog strips fold-marker noise and unsafe ANSI codes from a full log
// blob, while preserving Jenkins' own SGR color codes.
func CleanLog(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = CleanLine(l)
	}
	return strings.Join(lines, "\n")
}

// CleanLine cleans a single log line: drops concealed fold-marker spans,
// strips unsafe (non-color) ANSI codes, and collapses carriage-return
// overwrites (progress-bar style output) down to their final state.
func CleanLine(l string) string {
	l = concealSpan.ReplaceAllString(l, "")
	l = nonSGRCsi.ReplaceAllString(l, "")
	// CRLF line endings leave a trailing \r on every line; strip that before
	// collapsing any real \r-driven progress-bar overwrites within the line.
	l = strings.TrimSuffix(l, "\r")
	if i := strings.LastIndexByte(l, '\r'); i >= 0 {
		l = l[i+1:]
	}
	return l
}
