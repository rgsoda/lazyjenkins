package ui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// rowDelegate renders jobItem/runItem rows directly, bypassing
// list.DefaultDelegate so we can color status text without it being mangled
// by the default delegate's fuzzy-match highlighting (see items.go).
type rowDelegate struct {
	showDesc bool
}

func (d rowDelegate) Height() int {
	if d.showDesc {
		return 2
	}
	return 1
}

func (d rowDelegate) Spacing() int                        { return 0 }
func (d rowDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

func (d rowDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var title, desc string
	switch it := item.(type) {
	case jobItem:
		title = jobDot(it.job.Color) + " " + it.job.Name
	case runItem:
		status := lipgloss.NewStyle().Foreground(resultColor(it.run.Result, it.run.Status)).
			Render(fmt.Sprintf("%-9s", runResultLabel(it.run)))
		title = fmt.Sprintf("#%-5d %s", it.run.Number, status)
		desc = runDescription(it.run)
	default:
		return
	}

	indent := "  "
	titleStyle := lipgloss.NewStyle()
	if index == m.Index() {
		indent = lipgloss.NewStyle().Foreground(colorFocused).Render("▎") + " "
		titleStyle = titleStyle.Bold(true)
	}

	if d.showDesc {
		fmt.Fprintf(w, "%s%s\n  %s", indent, titleStyle.Render(title),
			lipgloss.NewStyle().Foreground(colorSubtle).Render(desc))
		return
	}
	fmt.Fprintf(w, "%s%s", indent, titleStyle.Render(title))
}
