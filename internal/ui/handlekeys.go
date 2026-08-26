package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"lazyjenkins/internal/jenkins"
)

func isActive(r jenkins.Run) bool {
	return r.Result == ""
}

func (m Model) typingInFilter() bool {
	return m.jobs.FilterState() == list.Filtering || m.runs.FilterState() == list.Filtering
}

// forwardToList reports whether a key should be routed straight to the list
// widget instead of being interpreted as one of our own bindings: while the
// user is actively typing a filter query, or clearing an applied filter.
func forwardToList(l list.Model, msg tea.KeyMsg) bool {
	switch l.FilterState() {
	case list.Filtering:
		return true
	case list.FilterApplied:
		return msg.String() == "esc"
	}
	return false
}

func (m *Model) cycleFocus() {
	switch m.focus {
	case focusJobs:
		m.focus = focusRuns
	case focusRuns:
		if m.mainMode != mainEmpty {
			m.focus = focusMain
		} else {
			m.focus = focusJobs
		}
	case focusMain:
		m.focus = focusJobs
	}
}

func (m *Model) stopFollowing() {
	if m.logCancel != nil {
		m.logCancel()
		m.logCancel = nil
	}
	m.following = false
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.confirm != confirmNone {
		switch msg.String() {
		case "y", "enter":
			return m.runConfirmed()
		case "n", "esc":
			m.confirm = confirmNone
			m.confirmParams = nil
			return m, nil
		}
		return m, nil
	}

	if m.mainMode == mainParams {
		if msg.String() == "esc" {
			m.mainMode = mainEmpty
			m.focus = focusRuns
			m.status = ""
			return m, nil
		}
		submitted, cmd := m.form.Update(msg)
		if submitted {
			m.confirm = confirmStart
			m.confirmParams = m.form.values()
			m.confirmPrompt = fmt.Sprintf("Start run for %q with these parameters?", m.form.jobPath)
		}
		return m, cmd
	}

	switch msg.String() {
	case "ctrl+c":
		m.stopFollowing()
		return m, tea.Quit
	case "q":
		if !m.typingInFilter() {
			m.stopFollowing()
			return m, tea.Quit
		}
	case "tab":
		m.cycleFocus()
		return m, nil
	}

	switch m.focus {
	case focusJobs:
		return m.handleJobsKey(msg)
	case focusRuns:
		return m.handleRunsKey(msg)
	case focusMain:
		return m.handleMainKey(msg)
	}
	return m, nil
}

func (m Model) handleJobsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if forwardToList(m.jobs, msg) {
		var cmd tea.Cmd
		m.jobs, cmd = m.jobs.Update(msg)
		return m, cmd
	}
	switch msg.String() {
	case "enter":
		if it, ok := m.jobs.SelectedItem().(jobItem); ok {
			m.selectedJob = it.job.Name
			m.selectedRun = nil
			m.loadingRuns = true
			m.focus = focusRuns
			return m, fetchRunsCmd(m.client, m.selectedJob)
		}
		return m, nil
	case "r":
		m.loadingJobs = true
		return m, fetchJobsCmd(m.client)
	case "s":
		if it, ok := m.jobs.SelectedItem().(jobItem); ok {
			m.selectedJob = it.job.Name
		}
		return m.beginStart()
	}
	var cmd tea.Cmd
	m.jobs, cmd = m.jobs.Update(msg)
	return m, cmd
}

func (m Model) handleRunsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if forwardToList(m.runs, msg) {
		var cmd tea.Cmd
		m.runs, cmd = m.runs.Update(msg)
		return m, cmd
	}
	switch msg.String() {
	case "enter":
		if it, ok := m.runs.SelectedItem().(runItem); ok {
			run := it.run
			m.selectedRun = &run
			return m.openLog(run)
		}
		return m, nil
	case "esc":
		m.focus = focusJobs
		return m, nil
	case "r":
		if m.selectedJob != "" {
			m.loadingRuns = true
			return m, fetchRunsCmd(m.client, m.selectedJob)
		}
	case "s":
		return m.beginStart()
	case "R":
		if it, ok := m.runs.SelectedItem().(runItem); ok {
			run := it.run
			m.selectedRun = &run
			return m.beginRerun()
		}
	case "c":
		if it, ok := m.runs.SelectedItem().(runItem); ok && isActive(it.run) {
			run := it.run
			m.selectedRun = &run
			return m.beginCancel()
		}
	}
	var cmd tea.Cmd
	m.runs, cmd = m.runs.Update(msg)
	return m, cmd
}

func (m Model) handleMainKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.stopFollowing()
		m.logGen++
		m.mainMode = mainEmpty
		m.focus = focusRuns
		return m, nil
	case "R":
		if m.selectedRun != nil {
			return m.beginRerun()
		}
	case "c":
		if m.selectedRun != nil && isActive(*m.selectedRun) {
			return m.beginCancel()
		}
	}
	var cmd tea.Cmd
	m.log, cmd = m.log.Update(msg)
	return m, cmd
}

func (m Model) beginStart() (tea.Model, tea.Cmd) {
	if m.selectedJob == "" {
		return m, nil
	}
	m.status = "loading parameters for " + m.selectedJob + "..."
	return m, fetchParamsCmd(m.client, m.selectedJob)
}

func (m Model) beginRerun() (tea.Model, tea.Cmd) {
	m.confirm = confirmRerun
	m.confirmPrompt = fmt.Sprintf("Rerun %s #%d with its previous parameters?", m.selectedJob, m.selectedRun.Number)
	return m, nil
}

func (m Model) beginCancel() (tea.Model, tea.Cmd) {
	m.confirm = confirmCancel
	m.confirmPrompt = fmt.Sprintf("Cancel %s #%d?", m.selectedJob, m.selectedRun.Number)
	return m, nil
}

func (m Model) openLog(run jenkins.Run) (tea.Model, tea.Cmd) {
	m.stopFollowing()
	m.logGen++
	gen := m.logGen
	m.mainMode = mainLog
	m.focus = focusMain
	m.logLines = nil
	m.log.SetContent("")
	m.loadingLog = true
	m.logRunPath = m.selectedJob
	m.logRunNumber = run.Number
	if isActive(run) {
		return m, startLogFollowCmd(m.client, m.selectedJob, run.Number, gen)
	}
	return m, fetchLogCmd(m.client, m.selectedJob, run.Number, gen)
}

func (m Model) runConfirmed() (tea.Model, tea.Cmd) {
	kind := m.confirm
	m.confirm = confirmNone
	switch kind {
	case confirmStart:
		params := m.confirmParams
		jobPath := m.selectedJob
		m.confirmParams = nil
		m.mainMode = mainEmpty
		m.status = "starting " + jobPath + "..."
		return m, startRunCmd(m.client, jobPath, params)
	case confirmRerun:
		m.status = fmt.Sprintf("rerunning %s #%d...", m.selectedJob, m.selectedRun.Number)
		return m, rerunCmd(m.client, m.selectedJob, m.selectedRun.Number)
	case confirmCancel:
		m.status = fmt.Sprintf("cancelling %s #%d...", m.selectedJob, m.selectedRun.Number)
		return m, cancelRunCmd(m.client, m.selectedJob, m.selectedRun.Number)
	}
	return m, nil
}
