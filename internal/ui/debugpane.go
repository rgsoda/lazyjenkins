package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"lazyjenkins/internal/jenkins"
)

// maxDebugLines caps how much history the debug pane keeps, so a long
// session (or a job someone starts polling aggressively) can't grow this
// unbounded.
const maxDebugLines = 1000

// formatDebugEntry renders one jk invocation as a couple of log lines: the
// exact command line run (so "why don't I see my build" is answerable at a
// glance — wrong folder, wrong limit, wrong job path, etc.), then its
// timing and either an error or a preview of what came back.
func formatDebugEntry(e jenkins.DebugEntry) string {
	ts := e.Time.Format("15:04:05.000")
	line := fmt.Sprintf("[%s] %s", ts, strings.Join(e.Args, " "))
	switch {
	case e.Err != nil:
		line += fmt.Sprintf("\n  ✗ %s — %s", e.Duration.Round(time.Millisecond), e.Err)
	case e.Preview != "":
		line += fmt.Sprintf("\n  ✓ %s — %s", e.Duration.Round(time.Millisecond), e.Preview)
	default:
		line += fmt.Sprintf("\n  ✓ %s", e.Duration.Round(time.Millisecond))
	}
	return line
}

// refreshDebugView re-renders the debug viewport's content, word-wrapping
// entries (a full jk command line can easily run past the panel width) to
// the pane's current width. lipgloss's wrap preserves the "\n" already in
// each entry as hard breaks, so the command line and its result line stay
// visually distinct.
func (m *Model) refreshDebugView() {
	content := strings.Join(m.debugLines, "\n")
	if m.wrapLogs && m.debugVP.Width > 0 {
		content = lipgloss.NewStyle().Width(m.debugVP.Width).Render(content)
	}
	m.debugVP.SetContent(content)
}
