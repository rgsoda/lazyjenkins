package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if !m.ready {
		return "starting lazyjenkins..."
	}
	if m.confirm != confirmNone {
		return m.confirmView()
	}

	header := m.headerView()
	jobsPanel := m.renderPanel(focusJobs, "Jobs", m.leftWidth, m.jobsHeight, m.jobs.View())
	runsPanel := m.renderPanel(focusRuns, m.runsTitle(), m.leftWidth, m.runsHeight, m.runs.View())
	left := lipgloss.JoinVertical(lipgloss.Left, jobsPanel, runsPanel)

	main := m.renderPanel(focusMain, m.mainTitle(), m.mainWidth, m.bodyHeight, m.mainContent())

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, " ", main)

	return lipgloss.JoinVertical(lipgloss.Left, header, body, m.footerView())
}

func (m Model) renderPanel(zone focusZone, title string, width, height int, content string) string {
	box := panelStyle(m.focus == zone, max(0, width-2), max(0, height-2))
	return box.Render(panelTitle(m.focus == zone, title) + "\n" + content)
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
			state = "following"
		}
		return fmt.Sprintf("Log — %s #%d (%s)", m.logRunPath, m.logRunNumber, state)
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
		if m.selectedJob == "" {
			return lipgloss.NewStyle().Foreground(colorSubtle).Render("Select a job, then a run, to see details here.")
		}
		return lipgloss.NewStyle().Foreground(colorSubtle).Render(
			"Job: " + m.selectedJob + "\n\nPress enter on a run to view its log.")
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
	hints := "tab switch panel · ↑/k ↓/j move · / filter · r refresh · q quit"
	switch m.focus {
	case focusJobs:
		hints = "enter open job · s start run · " + hints
	case focusRuns:
		hints = "enter view log · s start · R rerun · c cancel · " + hints
	case focusMain:
		if m.mainMode == mainLog {
			hints = "esc back · R rerun · c cancel · ↑/k ↓/j scroll · q quit"
		}
	}
	left := keyHintStyle.Render(" " + hints)
	right := ""
	if m.errStr != "" {
		right = errorBarStyle.Render(m.errStr)
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
