package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"lazyjenkins/internal/jenkins"
)

type focusZone int

const (
	focusJobs focusZone = iota
	focusRuns
	focusMain
)

type mainMode int

const (
	mainEmpty mainMode = iota
	mainLog
	mainParams
)

type confirmKind int

const (
	confirmNone confirmKind = iota
	confirmStart
	confirmRerun
	confirmCancel
)

type Model struct {
	client *jenkins.Client
	auth   jenkins.AuthStatus

	width, height int
	ready         bool

	leftWidth, mainWidth               int
	jobsHeight, runsHeight, bodyHeight int

	jobs list.Model
	runs list.Model

	selectedJob string
	selectedRun *jenkins.Run

	focus    focusZone
	mainMode mainMode

	log          viewport.Model
	logLines     []string
	following    bool
	logCancel    context.CancelFunc
	logRunPath   string
	logRunNumber int
	logGen       int

	form paramsForm

	confirm       confirmKind
	confirmPrompt string
	confirmParams map[string]string

	spinner     spinner.Model
	loadingJobs bool
	loadingRuns bool
	loadingLog  bool
	busy        bool // a start/rerun/cancel/params fetch is in flight

	status string
	errStr string
}

func New(client *jenkins.Client) Model {
	jobsList := list.New(nil, rowDelegate{showDesc: false}, 0, 0)
	jobsList.SetShowTitle(false)
	jobsList.SetShowStatusBar(false)
	jobsList.SetShowHelp(false)
	jobsList.DisableQuitKeybindings()

	runsList := list.New(nil, rowDelegate{showDesc: true}, 0, 0)
	runsList.SetShowTitle(false)
	runsList.SetShowStatusBar(false)
	runsList.SetShowHelp(false)
	runsList.DisableQuitKeybindings()

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	vp := viewport.New(0, 0)

	return Model{
		client:      client,
		jobs:        jobsList,
		runs:        runsList,
		spinner:     sp,
		log:         vp,
		focus:       focusJobs,
		loadingJobs: true,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(fetchAuthCmd(m.client), fetchJobsCmd(m.client), m.spinner.Tick, tickCmd())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ready = true
		m.layout()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case authLoadedMsg:
		m.auth = msg.auth
		return m, nil

	case jobsLoadedMsg:
		m.loadingJobs = false
		m.errStr = ""
		items := make([]list.Item, len(msg.jobs))
		for i, j := range msg.jobs {
			items[i] = jobItem{job: j}
		}
		cmd := m.jobs.SetItems(items)
		return m, cmd

	case runsLoadedMsg:
		m.loadingRuns = false
		m.errStr = ""
		if msg.jobPath != m.selectedJob {
			return m, nil
		}
		items := make([]list.Item, len(msg.runs))
		for i, r := range msg.runs {
			items[i] = runItem{run: r}
		}
		cmd := m.runs.SetItems(items)
		return m, cmd

	case paramsLoadedMsg:
		m.busy = false
		m.status = ""
		m.errStr = ""
		if len(msg.params) == 0 {
			m.confirm = confirmStart
			m.confirmPrompt = fmt.Sprintf("Start run for %q with no parameters?", msg.jobPath)
			m.confirmParams = map[string]string{}
			return m, nil
		}
		m.form = newParamsForm(msg.jobPath, msg.params)
		m.mainMode = mainParams
		m.focus = focusMain
		return m, nil

	case actionResultMsg:
		m.busy = false
		m.status = msg.label
		m.errStr = ""
		var cmds []tea.Cmd
		if m.selectedJob != "" {
			cmds = append(cmds, fetchRunsCmd(m.client, m.selectedJob))
		}
		cmds = append(cmds, fetchJobsCmd(m.client))
		return m, tea.Batch(cmds...)

	case errMsg:
		m.loadingJobs, m.loadingRuns, m.loadingLog, m.busy = false, false, false, false
		m.errStr = fmt.Sprintf("[%s] %v", msg.scope, msg.err)
		return m, nil

	case logLoadedMsg:
		if msg.gen != m.logGen {
			return m, nil
		}
		m.loadingLog = false
		m.errStr = ""
		m.logLines = strings.Split(msg.content, "\n")
		m.log.SetContent(msg.content)
		m.log.GotoBottom()
		return m, nil

	case logStreamStartedMsg:
		if msg.gen != m.logGen {
			msg.cancel()
			return m, nil
		}
		m.loadingLog = false
		m.errStr = ""
		m.following = true
		m.logCancel = msg.cancel
		return m, waitForLogLineCmd(msg.gen, msg.ch)

	case logLineMsg:
		if msg.gen != m.logGen {
			return m, nil
		}
		m.logLines = append(m.logLines, msg.line)
		m.log.SetContent(strings.Join(m.logLines, "\n"))
		m.log.GotoBottom()
		return m, waitForLogLineCmd(msg.gen, msg.ch)

	case logStreamDoneMsg:
		if msg.gen != m.logGen {
			return m, nil
		}
		m.following = false
		m.logLines = append(m.logLines, "", "-- stream ended --")
		m.log.SetContent(strings.Join(m.logLines, "\n"))
		m.log.GotoBottom()
		if m.selectedJob != "" {
			return m, fetchRunsCmd(m.client, m.selectedJob)
		}
		return m, nil

	case tickMsg:
		var cmds []tea.Cmd
		if m.confirm == confirmNone && m.mainMode != mainParams {
			cmds = append(cmds, fetchJobsCmd(m.client))
			if m.selectedJob != "" {
				cmds = append(cmds, fetchRunsCmd(m.client, m.selectedJob))
			}
		}
		cmds = append(cmds, tickCmd())
		return m, tea.Batch(cmds...)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	// Anything we don't special-case (e.g. list.FilterMatchesMsg, produced
	// asynchronously by the lists' own fuzzy-filter pipeline) still needs to
	// reach the lists, or filtering silently never applies.
	var jobsCmd, runsCmd tea.Cmd
	m.jobs, jobsCmd = m.jobs.Update(msg)
	m.runs, runsCmd = m.runs.Update(msg)
	return m, tea.Batch(jobsCmd, runsCmd)
}

func (m *Model) layout() {
	if !m.ready {
		return
	}
	m.leftWidth = m.width * 3 / 10
	if m.leftWidth < 24 {
		m.leftWidth = 24
	}
	if m.leftWidth > 35 {
		m.leftWidth = 35
	}
	m.mainWidth = m.width - m.leftWidth - 1
	m.bodyHeight = m.height - 2 // header + status bar

	m.jobsHeight = m.bodyHeight / 2
	m.runsHeight = m.bodyHeight - m.jobsHeight

	leftContentW := max(0, m.leftWidth-2)
	mainContentW := max(0, m.mainWidth-2)

	// -2 for the border; the panel title lives in the border itself now.
	m.jobs.SetSize(leftContentW, max(0, m.jobsHeight-2))
	m.runs.SetSize(leftContentW, max(0, m.runsHeight-2))
	m.log.Width = mainContentW
	m.log.Height = max(0, m.bodyHeight-2)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
