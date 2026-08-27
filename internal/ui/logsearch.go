package ui

import (
	"regexp"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// logSearch implements vim-style "/" search over the log viewport: type a
// regex, jump to the first match at or after the current scroll position,
// then cycle through matches with n/N.
type logSearch struct {
	input   textinput.Model
	typing  bool
	active  bool
	pattern *regexp.Regexp
	raw     string
	matches []int // matching line indices into Model.logLines, ascending
	idx     int
	err     string
}

func newLogSearch() logSearch {
	ti := textinput.New()
	ti.Prompt = "/"
	return logSearch{input: ti}
}

func (s *logSearch) begin() {
	*s = newLogSearch()
	s.typing = true
	s.input.Focus()
}

func (s *logSearch) cancelTyping() {
	s.typing = false
	s.input.Blur()
}

func (s *logSearch) updateTyping(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return cmd
}

// submit compiles the typed pattern, finds all matching lines, and jumps to
// the first one at or after currentLine (wrapping to the top if none).
func (s *logSearch) submit(lines []string, currentLine int) {
	s.typing = false
	s.input.Blur()
	val := s.input.Value()
	s.raw = val
	if val == "" {
		s.active, s.matches, s.err = false, nil, ""
		return
	}
	pattern, err := regexp.Compile(val)
	if err != nil {
		s.active, s.matches, s.err = false, nil, err.Error()
		return
	}
	s.pattern, s.err = pattern, ""
	matches := make([]int, 0, len(s.matches))
	for i, l := range lines {
		if pattern.MatchString(l) {
			matches = append(matches, i)
		}
	}
	s.matches = matches
	if len(s.matches) == 0 {
		s.active, s.err = false, "no matches"
		return
	}
	s.active = true
	s.idx = sort.Search(len(s.matches), func(i int) bool { return s.matches[i] > currentLine })
	if s.idx == len(s.matches) {
		s.idx = 0
	}
}

// noteNewLine checks a freshly-appended line (at index lineIdx) against the
// active pattern, extending matches so a live-following search keeps
// finding new occurrences without the user having to re-run it.
func (s *logSearch) noteNewLine(lineIdx int, line string) {
	if s.pattern == nil || !s.pattern.MatchString(line) {
		return
	}
	s.matches = append(s.matches, lineIdx)
	s.active = true
}

func (s *logSearch) next() {
	if len(s.matches) > 0 {
		s.idx = (s.idx + 1) % len(s.matches)
	}
}

func (s *logSearch) prev() {
	if len(s.matches) > 0 {
		s.idx = (s.idx - 1 + len(s.matches)) % len(s.matches)
	}
}

var (
	searchMatchStyle   = lipgloss.NewStyle().Background(lipgloss.Color("58"))
	searchCurrentStyle = lipgloss.NewStyle().Background(lipgloss.Color("226")).Foreground(lipgloss.Color("16")).Bold(true)
)

// renderLogContent applies structural highlighting to each (plain) log line
// and, on top of that, the search highlight for any matching line. Matching
// itself always happens against the plain lines (m.logLines) — never
// against this rendered-and-colored output — so anchored patterns like
// "^ERROR" aren't defeated by our own embedded ANSI codes.
//
// It also returns, per logical log line, the visual row at which that line
// starts once word-wrapping has (maybe) split it across several rows —
// scrollToCurrentMatch needs that to land the viewport in the right place.
func (m Model) renderLogContent() (content string, rowOffsets []int) {
	matches := m.search.matches
	current := -1
	if len(matches) > 0 {
		current = matches[m.search.idx]
	}
	out := make([]string, len(m.logLines))
	rowOffsets = make([]int, len(m.logLines))
	row, mi := 0, 0
	for i, raw := range m.logLines {
		var line string
		if mi < len(matches) && matches[mi] == i {
			// Highlight straight from the raw (uncolored) line rather than
			// wrapping highlightLogLine's output: that output can contain
			// its own full-reset codes partway through (e.g. after a
			// styled timestamp), which would cut our background off after
			// just the first few characters instead of the whole line.
			if i == current {
				line = searchCurrentStyle.Render(raw)
			} else {
				line = searchMatchStyle.Render(raw)
			}
			mi++
		} else {
			line = highlightLogLine(raw)
		}
		rowOffsets[i] = row
		if m.wrapLogs && m.log.Width > 0 {
			line = lipgloss.NewStyle().Width(m.log.Width).Render(line)
		}
		out[i] = line
		row += strings.Count(line, "\n") + 1
	}
	return strings.Join(out, "\n"), rowOffsets
}

// refreshLogView re-renders the log viewport's content from m.logLines,
// applying the current highlighting, search state, and wrap setting. Call
// this any time one of those inputs changes.
func (m *Model) refreshLogView() {
	content, offsets := m.renderLogContent()
	m.log.SetContent(content)
	m.logRowOffsets = offsets
}

// scrollToCurrentMatch centers the viewport on the active search match.
func (m *Model) scrollToCurrentMatch() {
	if len(m.search.matches) == 0 {
		return
	}
	line := m.search.matches[m.search.idx]
	row := line
	if line < len(m.logRowOffsets) {
		row = m.logRowOffsets[line]
	}
	target := row - m.log.Height/2
	if target < 0 {
		target = 0
	}
	m.log.SetYOffset(target)
}
