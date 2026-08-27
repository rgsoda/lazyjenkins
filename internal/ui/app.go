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
	focusDebug
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
	mainHeight, debugHeight            int

	debugOn    bool
	debugCh    <-chan jenkins.DebugEntry
	debugLines []string
	debugVP    viewport.Model

	jobs list.Model
	runs list.Model

	selectedJob string
	selectedRun *jenkins.Run

	focus    focusZone
	mainMode mainMode

	log           viewport.Model
	logLines      []string
	logRowOffsets []int // logical line -> visual row, once word-wrapped
	wrapLogs      bool
	following     bool
	logCancel     context.CancelFunc
	logRunPath    string
	logRunNumber  int
	logGen        int
	search        logSearch

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

	// Context picker, shown before anything else loads when the user
	// didn't pin a context (via --context/JK_CONTEXT) and jk has 2+
	// configured. See contextsLoadedMsg and handleContextPickKey.
	explicitContext string
	pickingContext  bool
	contexts        []jenkins.Context
	contextIdx      int
}

func New(client *jenkins.Client, debug bool, explicitContext string) Model {
	var debugCh <-chan jenkins.DebugEntry
	if debug {
		ch := make(chan jenkins.DebugEntry, 256)
		debugCh = ch
		client.Debug = func(e jenkins.DebugEntry) {
			select {
			case ch <- e:
			default: // pane can't keep up; drop rather than block jk calls
			}
		}
	}

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
	debugVP := viewport.New(0, 0)
	// bubbles' horizontal scroll step defaults to 0 (a silent no-op for
	// h/l/left/right) unless set explicitly — needed for nowrap mode.
	vp.SetHorizontalStep(10)
	debugVP.SetHorizontalStep(10)

	return Model{
		client:          client,
		jobs:            jobsList,
		runs:            runsList,
		spinner:         sp,
		log:             vp,
		wrapLogs:        true,
		search:          newLogSearch(),
		focus:           focusJobs,
		loadingJobs:     true,
		debugOn:         debug,
		debugCh:         debugCh,
		debugVP:         debugVP,
		explicitContext: explicitContext,
	}
}

func (m Model) Init() tea.Cmd {
	if m.explicitContext != "" {
		return m.startupCmds()
	}
	// No context pinned via --context/JK_CONTEXT: find out if there's
	// even a choice to make before loading anything else.
	return fetchContextsCmd(m.client)
}

// startupCmds is the normal "go load everything" batch — fired immediately
// when a context is already pinned, or once the user picks one.
func (m Model) startupCmds() tea.Cmd {
	cmds := []tea.Cmd{fetchAuthCmd(m.client), fetchJobsCmd(m.client), m.spinner.Tick, tickCmd()}
	if m.debugOn {
		cmds = append(cmds, waitForDebugCmd(m.debugCh))
	}
	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ready = true
		m.layout()
		if m.mainMode == mainLog {
			// Wrapping is width-dependent; re-flow already-loaded content
			// against the new width instead of leaving it wrapped stale.
			m.refreshLogView()
		}
		if m.debugOn {
			m.refreshDebugView()
		}
		return m, nil

	case tea.KeyMsg:
		if m.pickingContext {
			return m.handleContextPickKey(msg)
		}
		return m.handleKey(msg)

	case contextsLoadedMsg:
		if len(msg.contexts) <= 1 {
			return m, m.startupCmds()
		}
		m.contexts = msg.contexts
		for i, c := range msg.contexts {
			if c.Active {
				m.contextIdx = i
			}
		}
		m.pickingContext = true
		return m, nil

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
		// Sitting in an empty runs panel is a dead end — bounce to the main
		// panel, which now shows a "press s to start one" prompt. Only when
		// we're actually parked in the runs panel, so a quiet background
		// refresh never yanks focus away from wherever the user is.
		if len(items) == 0 && m.focus == focusRuns {
			m.focus = focusMain
		}
		return m, cmd

	case paramsLoadedMsg:
		m.busy = false
		m.status = ""
		m.errStr = ""
		// Always open the form, even with zero discovered params: jk can't
		// discover Jenkinsfile-declared parameters for a job with no prior
		// runs, but the job may still take some — let the user add them by
		// hand via the form's custom key=value rows.
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
		if msg.scope == "contexts" {
			// Failing to list contexts shouldn't brick startup — just
			// proceed with whatever jk's own active context is.
			return m, m.startupCmds()
		}
		return m, nil

	case logLoadedMsg:
		if msg.gen != m.logGen {
			return m, nil
		}
		m.loadingLog = false
		m.errStr = ""
		m.logLines = strings.Split(msg.content, "\n")
		m.refreshLogView()
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
		m.search.noteNewLine(len(m.logLines)-1, msg.line)
		m.refreshLogView()
		if !m.search.active {
			m.log.GotoBottom()
		}
		return m, waitForLogLineCmd(msg.gen, msg.ch)

	case logStreamDoneMsg:
		if msg.gen != m.logGen {
			return m, nil
		}
		m.following = false
		m.logLines = append(m.logLines, "", "-- stream ended --")
		m.refreshLogView()
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

	case debugEntryMsg:
		m.debugLines = append(m.debugLines, formatDebugEntry(jenkins.DebugEntry(msg)))
		if len(m.debugLines) > maxDebugLines {
			m.debugLines = m.debugLines[len(m.debugLines)-maxDebugLines:]
		}
		m.refreshDebugView()
		m.debugVP.GotoBottom()
		return m, waitForDebugCmd(m.debugCh)
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

	if m.debugOn {
		m.debugHeight = max(5, m.bodyHeight/3)
		m.mainHeight = m.bodyHeight - m.debugHeight - 1 // -1 for the gap between them
		m.debugVP.Width = mainContentW
		m.debugVP.Height = max(0, m.debugHeight-2)
	} else {
		m.mainHeight = m.bodyHeight
	}
	m.log.Width = mainContentW
	m.log.Height = max(0, m.mainHeight-2)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
