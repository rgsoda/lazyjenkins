package ui

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// highlightLogLine adds light structural coloring to plain log lines.
// Jenkins' own ansiColor-plugin output already carries real ANSI codes
// (preserved by jenkins.CleanLine) — we leave those completely alone and
// only touch lines with no escape codes of their own, so we never fight
// with or override Jenkins' own colors.
func highlightLogLine(line string) string {
	if strings.ContainsRune(line, '\x1b') {
		return line
	}
	if m := finishedRe.FindStringSubmatch(line); m != nil {
		return lipgloss.NewStyle().Bold(true).Foreground(resultColor(m[1], "")).Render(line)
	}
	if pipelineRe.MatchString(line) {
		return lipgloss.NewStyle().Foreground(colorBuilding).Render(line)
	}
	if loc := timestampRe.FindStringIndex(line); loc != nil {
		stamp := lipgloss.NewStyle().Foreground(colorSubtle).Render(line[loc[0]:loc[1]])
		return stamp + line[loc[1]:]
	}
	return line
}

var (
	timestampRe = regexp.MustCompile(`^\[\d{4}-\d{2}-\d{2}T[0-9:.]+Z\]`)
	pipelineRe  = regexp.MustCompile(`^\[Pipeline\]`)
	finishedRe  = regexp.MustCompile(`^Finished: (\w+)$`)
)
