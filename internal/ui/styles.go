package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Palette loosely follows lazygit's default theme: green for the focused
// panel, cyan-ish accents for selection, and semantic colors for build
// results.
var (
	colorFocused   = lipgloss.Color("42")  // green
	colorBlurred   = lipgloss.Color("240") // gray
	colorSelected  = lipgloss.Color("86")  // cyan
	colorSubtle    = lipgloss.Color("245")
	colorSuccess   = lipgloss.Color("42")  // green
	colorFailure   = lipgloss.Color("196") // red
	colorUnstable  = lipgloss.Color("214") // orange
	colorAborted   = lipgloss.Color("244") // gray
	colorBuilding  = lipgloss.Color("39")  // blue
	colorWarn      = lipgloss.Color("214")
	colorHeaderBg  = lipgloss.Color("57")
	colorHeaderFg  = lipgloss.Color("255")
	colorErrorText = lipgloss.Color("196")
)

// renderBoxedPanel draws a lazygit-style panel: the "[N] Title" label is
// spliced directly into the top border rather than taking a content row.
func renderBoxedPanel(focused bool, width, height, number int, title, content string) string {
	c := colorBlurred
	if focused {
		c = colorFocused
	}
	border := lipgloss.RoundedBorder()
	borderStyle := lipgloss.NewStyle().Foreground(c)

	prefix, suffix := fmt.Sprintf("─[%d] ", number), "─"
	avail := max(0, width-lipgloss.Width(prefix)-lipgloss.Width(suffix))
	head := prefix + ansi.Truncate(title, avail, ellipsis) + suffix
	dashes := max(0, width-lipgloss.Width(head))
	top := border.TopLeft + head + strings.Repeat(border.Top, dashes) + border.TopRight

	box := lipgloss.NewStyle().
		Border(border, false, true, true, true).
		BorderForeground(c).
		Width(width).
		Height(height)

	return borderStyle.Render(top) + "\n" + box.Render(content)
}

var (
	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("250")).
			Padding(0, 1)

	keyHintStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("108"))

	errorBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("52")).
			Foreground(lipgloss.Color("231")).
			Padding(0, 1)

	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorFocused).
			Padding(1, 2)

	modalTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(colorFocused)

	dangerModalBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorFailure).
				Padding(1, 2)
)

func resultColor(result, status string) lipgloss.Color {
	switch result {
	case "SUCCESS":
		return colorSuccess
	case "FAILURE":
		return colorFailure
	case "UNSTABLE":
		return colorUnstable
	case "ABORTED":
		return colorAborted
	}
	if status == "building" || status == "queued" {
		return colorBuilding
	}
	return colorSubtle
}

// jobDot renders a small colored bullet based on Jenkins' job "color" field
// (e.g. "blue", "red", "yellow", "notbuilt", "blue_anime").
func jobDot(color string) string {
	base := color
	building := false
	if rest, ok := cutSuffix(color, "_anime"); ok {
		base = rest
		building = true
	}
	var c lipgloss.Color
	switch base {
	case "blue":
		c = colorSuccess
	case "red":
		c = colorFailure
	case "yellow":
		c = colorUnstable
	case "grey", "gray", "disabled", "notbuilt", "aborted":
		c = colorAborted
	default:
		c = colorSubtle
	}
	glyph := "●"
	if building {
		glyph = "◐"
	}
	return lipgloss.NewStyle().Foreground(c).Render(glyph)
}

func cutSuffix(s, suf string) (string, bool) {
	if len(s) >= len(suf) && s[len(s)-len(suf):] == suf {
		return s[:len(s)-len(suf)], true
	}
	return s, false
}
