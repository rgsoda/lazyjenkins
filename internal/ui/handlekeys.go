package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"lazyjenkins/internal/jenkins"
)

// formatParams renders a params map for the confirm prompt, sorted for
// deterministic output (map iteration order isn't).
func formatParams(params map[string]string) string {
	if len(params) == 0 {
		return "(no parameters)"
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	lines := make([]string, len(keys))
	for i, k := range keys {
		lines[i] = k + "=" + params[k]
	}
	return strings.Join(lines, "\n")
}

func isActive(r jenkins.Run) bool {
	return r.Result == ""
}

func (m Model) handleContextPickKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q", "esc":
		return m, tea.Quit
	case "up", "k":
		m.contextIdx = (m.contextIdx - 1 + len(m.contexts)) % len(m.contexts)
	case "down", "j":
		m.contextIdx = (m.contextIdx + 1) % len(m.contexts)
	case "enter":
		m.client.Context = m.contexts[m.contextIdx].Name
		m.pickingContext = false
		return m, m.startupCmds()
	}
	return m, nil
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

// resizeLeft adjusts the left column (Jobs/Runs) width by delta columns,
// clamped to [leftWidthMin, leftWidthMax]. Once used, it pins the width —
// the automatic 30%-of-terminal sizing no longer applies this session.
func (m *Model) resizeLeft(delta int) {
	cur := m.leftWidthOverride
	if cur == 0 {
		cur = m.leftWidth
	}
	cur += delta
	if cur < leftWidthMin {
		cur = leftWidthMin
	}
	if cur > leftWidthMax {
		cur = leftWidthMax
	}
	m.leftWidthOverride = cur
	m.layout()
	if m.mainMode == mainLog {
		m.refreshLogView()
	}
	if m.debugOn {
		m.refreshDebugView()
	}
}

func (m *Model) cycleFocus() {
	switch m.focus {
	case focusJobs:
		m.focus = focusRuns
	case focusRuns:
		if m.mainMode != mainEmpty {
			m.focus = focusMain
		} else if m.debugOn {
			m.focus = focusDebug
		} else {
			m.focus = focusJobs
		}
	case focusMain:
		if m.debugOn {
			m.focus = focusDebug
		} else {
			m.focus = focusJobs
		}
	case focusDebug:
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
			m.confirmPrompt = fmt.Sprintf("Start run for %q?\n\n%s", m.form.jobPath, formatParams(m.confirmParams))
		}
		return m, cmd
	}

	if m.search.typing {
		switch msg.String() {
		case "esc":
			m.search.cancelTyping()
			return m, nil
		case "enter":
			m.search.submit(m.logLines, m.log.YOffset)
			m.refreshLogView()
			m.scrollToCurrentMatch()
			return m, nil
		}
		cmd := m.search.updateTyping(msg)
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
	case "1":
		if !m.typingInFilter() {
			m.focus = focusJobs
			return m, nil
		}
	case "2":
		if !m.typingInFilter() {
			m.focus = focusRuns
			return m, nil
		}
	case "3":
		if !m.typingInFilter() {
			m.focus = focusMain
			return m, nil
		}
	case "4":
		if m.debugOn && !m.typingInFilter() {
			m.focus = focusDebug
			return m, nil
		}
	case "+", "=":
		if !m.typingInFilter() {
			m.resizeLeft(2)
			return m, nil
		}
	case "-", "_":
		if !m.typingInFilter() {
			m.resizeLeft(-2)
			return m, nil
		}
	}

	switch m.focus {
	case focusJobs:
		return m.handleJobsKey(msg)
	case focusRuns:
		return m.handleRunsKey(msg)
	case focusMain:
		return m.handleMainKey(msg)
	case focusDebug:
		return m.handleDebugKey(msg)
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
			if it.job.IsFolder() {
				return m.enterFolder(it.job.Name)
			}
			m.selectedJob = joinJobPath(m.jobFolder, it.job.Name)
			m.selectedRun = nil
			m.runs.ResetFilter() // clear any filter left from a previously viewed job
			m.loadingRuns = true
			m.focus = focusRuns
			return m, fetchRunsCmd(m.client, m.selectedJob)
		}
		return m, nil
	case "esc":
		if m.jobFolder != "" {
			return m.exitFolder()
		}
	case "r":
		m.loadingJobs = true
		return m, fetchJobsCmd(m.client, m.jobFolder)
	case "s":
		if it, ok := m.jobs.SelectedItem().(jobItem); ok && !it.job.IsFolder() {
			m.selectedJob = joinJobPath(m.jobFolder, it.job.Name)
			return m.beginStart()
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.jobs, cmd = m.jobs.Update(msg)
	return m, cmd
}

// joinJobPath builds a jk job path from a folder and a child name. jk
// already returns child names pre-encoded exactly as Jenkins expects them
// back (e.g. a branch "feat/x" is reported as "feat%2Fx"), so this is a
// plain join — no extra escaping needed.
func joinJobPath(folder, name string) string {
	if folder == "" {
		return name
	}
	return folder + "/" + name
}

func (m Model) enterFolder(name string) (tea.Model, tea.Cmd) {
	m.folderStack = append(m.folderStack, m.jobFolder)
	m.jobFolder = joinJobPath(m.jobFolder, name)
	m.selectedJob = ""
	m.selectedRun = nil
	// A filter typed for the parent listing (e.g. what you used to find
	// this folder) would otherwise silently apply to the child listing
	// too — bubbles' list.SetItems doesn't clear it on its own, and it can
	// hide every item (or leave one coincidental match) with no visible
	// indication why.
	m.jobs.ResetFilter()
	m.loadingJobs = true
	return m, fetchJobsCmd(m.client, m.jobFolder)
}

func (m Model) exitFolder() (tea.Model, tea.Cmd) {
	if len(m.folderStack) == 0 {
		return m, nil
	}
	m.jobFolder = m.folderStack[len(m.folderStack)-1]
	m.folderStack = m.folderStack[:len(m.folderStack)-1]
	m.jobs.ResetFilter()
	m.loadingJobs = true
	return m, fetchJobsCmd(m.client, m.jobFolder)
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
	case "y":
		if it, ok := m.runs.SelectedItem().(runItem); ok {
			return m.copyBuildLink(it.run.URL)
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
	case "/":
		if m.mainMode == mainLog {
			m.search.begin()
			return m, nil
		}
	case "n":
		if m.mainMode == mainLog && m.search.active {
			m.search.next()
			m.refreshLogView()
			m.scrollToCurrentMatch()
			return m, nil
		}
	case "N":
		if m.mainMode == mainLog && m.search.active {
			m.search.prev()
			m.refreshLogView()
			m.scrollToCurrentMatch()
			return m, nil
		}
	case "w":
		if m.mainMode == mainLog {
			m.wrapLogs = !m.wrapLogs
			m.refreshLogView()
			return m, nil
		}
	case "s":
		if m.mainMode == mainEmpty {
			return m.beginStart()
		}
	case "R":
		if m.selectedRun != nil {
			return m.beginRerun()
		}
	case "c":
		if m.selectedRun != nil && isActive(*m.selectedRun) {
			return m.beginCancel()
		}
	case "y":
		if m.mainMode == mainLog && m.selectedRun != nil {
			return m.copyBuildLink(m.selectedRun.URL)
		}
	}
	var cmd tea.Cmd
	m.log, cmd = m.log.Update(msg)
	return m, cmd
}

// copyBuildLink copies a run's Jenkins URL to the system clipboard,
// reporting success or failure the same way as other quick actions.
func (m Model) copyBuildLink(url string) (tea.Model, tea.Cmd) {
	if err := clipboard.WriteAll(url); err != nil {
		m.errStr = fmt.Sprintf("[clipboard] %v", err)
		return m, nil
	}
	m.errStr = ""
	m.status = "copied " + url + " to clipboard"
	return m, nil
}

func (m Model) handleDebugKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "c":
		m.debugLines = nil
		m.debugVP.SetContent("")
		return m, nil
	case "w":
		m.wrapLogs = !m.wrapLogs
		m.refreshDebugView()
		return m, nil
	}
	var cmd tea.Cmd
	m.debugVP, cmd = m.debugVP.Update(msg)
	return m, cmd
}

func (m Model) beginStart() (tea.Model, tea.Cmd) {
	if m.selectedJob == "" {
		return m, nil
	}
	m.busy = true
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
	m.search = newLogSearch()
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
	m.busy = true
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
	m.busy = false
	return m, nil
}
