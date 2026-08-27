package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if !m.ready {
		return "starting lazyjenkins..."
	}
	if m.pickingContext {
		return m.contextPickView()
	}
	if m.confirm != confirmNone {
		return m.confirmView()
	}

	header := m.headerView()

	jobsContent := m.jobs.View()
	if m.loadingJobs {
		jobsContent = "  " + m.spinner.View() + " loading jobs..."
	}
	runsContent := m.runs.View()
	if m.loadingRuns {
		runsContent = "  " + m.spinner.View() + " loading runs..."
	}
	jobsPanel := m.renderPanel(focusJobs, m.jobsTitle(), m.leftWidth, m.jobsHeight, jobsContent)
	runsPanel := m.renderPanel(focusRuns, m.runsTitle(), m.leftWidth, m.runsHeight, runsContent)
	left := lipgloss.JoinVertical(lipgloss.Left, jobsPanel, runsPanel)

	main := m.renderPanel(focusMain, m.mainTitle(), m.mainWidth, m.mainHeight, m.mainContent())
	right := main
	if m.debugOn {
		debugTitle := "Debug — jk calls"
		if !m.wrapLogs {
			debugTitle += " · nowrap"
		}
		debugPanel := m.renderPanel(focusDebug, debugTitle, m.mainWidth, m.debugHeight, m.debugVP.View())
		right = lipgloss.JoinVertical(lipgloss.Left, main, debugPanel)
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)

	return lipgloss.JoinVertical(lipgloss.Left, header, body, m.footerView())
}

func zoneNumber(zone focusZone) int {
	switch zone {
	case focusJobs:
		return 1
	case focusRuns:
		return 2
	case focusMain:
		return 3
	default:
		return 4
	}
}

func (m Model) renderPanel(zone focusZone, title string, width, height int, content string) string {
	return renderBoxedPanel(m.focus == zone, max(0, width-2), max(0, height-2), zoneNumber(zone), title, content)
}

func (m Model) jobsTitle() string {
	if m.jobFolder == "" {
		return "Jobs"
	}
	return "Jobs — " + m.jobFolder
}

func (m Model) runsTitle() string {
	if m.selectedJob == "" {
		return "Runs"
	}
	return "Runs — " + m.selectedJob
}

func (m Model) mainTitle() string {
	switch m.mainMode {
	case mainLog:
		state := "done"
		if m.following {
			state = m.spinner.View() + " following"
		}
		title := fmt.Sprintf("Log — %s #%d (%s, %d lines)", m.logRunPath, m.logRunNumber, state, len(m.logLines))
		if !m.wrapLogs {
			title += " · nowrap"
		}
		switch {
		case m.search.err != "":
			title += " · /" + m.search.raw + " " + m.search.err
		case m.search.active:
			title += fmt.Sprintf(" · /%s %d/%d", m.search.raw, m.search.idx+1, len(m.search.matches))
		}
		return title
	case mainParams:
		return "Start Run"
	default:
		return "Details"
	}
}

func (m Model) mainContent() string {
	switch m.mainMode {
	case mainLog:
		if m.loadingLog {
			return m.spinner.View() + " loading log..."
		}
		return m.log.View()
	case mainParams:
		return m.form.View()
	default:
		style := lipgloss.NewStyle().Foreground(colorSubtle)
		if m.selectedJob == "" {
			return style.Render("Select a job, then a run, to see details here.")
		}
		if m.loadingRuns {
			return style.Render("Job: " + m.selectedJob)
		}
		if len(m.runs.Items()) == 0 {
			return style.Render("Job: " + m.selectedJob + "\n\nNo runs found.\nPress s to start one.")
		}
		return style.Render("Job: " + m.selectedJob + "\n\nPress enter on a run to view its log.")
	}
}

func (m Model) headerView() string {
	left := lipgloss.NewStyle().Bold(true).Foreground(colorFocused).Render(" lazyjenkins ")
	right := ""
	if m.auth.Username != "" {
		right = fmt.Sprintf("%s @ %s ", m.auth.Username, m.auth.Context)
	}
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	return left + lipgloss.NewStyle().Foreground(colorSubtle).Render(fmt.Sprintf("%*s", gap, "")) + right
}

func (m Model) footerView() string {
	if m.focus == focusMain && m.search.typing {
		return keyHintStyle.Render(" " + m.search.input.View())
	}

	jumpKeys := "1/2/3"
	if m.debugOn {
		jumpKeys = "1/2/3/4"
	}
	hints := jumpKeys + " · tab jump · +/- resize · ↑/k ↓/j move · / filter · r refresh · q quit"
	switch m.focus {
	case focusJobs:
		if m.jobFolder != "" {
			hints = "enter open · esc back · s start run · " + hints
		} else {
			hints = "enter open · s start run · " + hints
		}
	case focusRuns:
		hints = "enter view log · s start · R rerun · c cancel · y copy link · " + hints
	case focusMain:
		switch {
		case m.mainMode == mainLog && m.search.active:
			hints = "esc back · / search · n/N next/prev match · w wrap · R rerun · c cancel · y copy link · ↑/k ↓/j scroll · q quit"
		case m.mainMode == mainLog:
			hints = "esc back · / search · w wrap · R rerun · c cancel · y copy link · ↑/k ↓/j scroll · q quit"
		case m.mainMode == mainEmpty && m.selectedJob != "":
			hints = "s start run · " + hints
		}
	case focusDebug:
		hints = "c clear · w wrap · ↑/k ↓/j scroll · " + hints
	}
	left := keyHintStyle.Render(" " + hints)
	right := ""
	if m.errStr != "" {
		right = errorBarStyle.Render(m.errStr)
	} else if m.busy {
		right = statusBarStyle.Render(m.spinner.View() + " " + m.status)
	} else if m.status != "" {
		right = statusBarStyle.Render(m.status)
	}
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	filler := statusBarStyle.Render(fmt.Sprintf("%*s", gap, ""))
	return left + filler + right
}

func (m Model) contextPickView() string {
	var b strings.Builder
	b.WriteString(modalTitleStyle.Render("Choose a Jenkins context") + "\n\n")
	for i, c := range m.contexts {
		cursor := "  "
		style := lipgloss.NewStyle()
		if i == m.contextIdx {
			cursor = lipgloss.NewStyle().Foreground(colorFocused).Render("▎") + " "
			style = style.Bold(true)
		}
		label := c.Name
		if c.Active {
			label += "  " + lipgloss.NewStyle().Foreground(colorSubtle).Render("(active)")
		}
		b.WriteString(cursor + style.Render(label) + "\n")
		b.WriteString("    " + lipgloss.NewStyle().Foreground(colorSubtle).Render(c.URL) + "\n")
	}
	b.WriteString("\n" + keyHintStyle.Render(" ↑/k ↓/j move · enter select · q quit "))
	box := modalStyle.Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m Model) confirmView() string {
	style := modalStyle
	if m.confirm == confirmCancel {
		style = dangerModalBorder
	}
	box := style.Render(
		modalTitleStyle.Render("Confirm") + "\n\n" +
			m.confirmPrompt + "\n\n" +
			keyHintStyle.Render(" y/enter confirm · n/esc abort "))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}
