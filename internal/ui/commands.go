package ui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"lazyjenkins/internal/jenkins"
)

type authLoadedMsg struct{ auth jenkins.AuthStatus }
type errMsg struct {
	scope string // "jobs", "runs", "params", "log", "action"
	err   error
}

type jobsLoadedMsg struct{ jobs []jenkins.Job }
type runsLoadedMsg struct {
	jobPath string
	runs    []jenkins.Run
}
type paramsLoadedMsg struct {
	jobPath string
	params  []jenkins.Param
}
type actionResultMsg struct {
	label  string
	output string
}

// Log messages carry a generation number tagging which openLog() call
// started them. A cancelled stream's goroutine can still deliver one more
// message after the user has already opened a different run's log; without
// this tag that stale message would corrupt the new log view.
type logLoadedMsg struct {
	gen     int
	content string
}
type logStreamStartedMsg struct {
	gen    int
	ch     <-chan string
	cancel context.CancelFunc
}
type logLineMsg struct {
	gen  int
	line string
	ch   <-chan string
}
type logStreamDoneMsg struct{ gen int }

type tickMsg time.Time

func fetchAuthCmd(c *jenkins.Client) tea.Cmd {
	return func() tea.Msg {
		st, err := c.AuthStatus(context.Background())
		if err != nil {
			return errMsg{scope: "auth", err: err}
		}
		return authLoadedMsg{auth: st}
	}
}

func fetchJobsCmd(c *jenkins.Client) tea.Cmd {
	return func() tea.Msg {
		jobs, err := c.JobLs(context.Background(), "")
		if err != nil {
			return errMsg{scope: "jobs", err: err}
		}
		return jobsLoadedMsg{jobs: jobs}
	}
}

func fetchRunsCmd(c *jenkins.Client, jobPath string) tea.Cmd {
	return func() tea.Msg {
		runs, err := c.RunLs(context.Background(), jobPath, 30)
		if err != nil {
			return errMsg{scope: "runs", err: err}
		}
		return runsLoadedMsg{jobPath: jobPath, runs: runs}
	}
}

func fetchParamsCmd(c *jenkins.Client, jobPath string) tea.Cmd {
	return func() tea.Msg {
		params, err := c.RunParams(context.Background(), jobPath)
		if err != nil {
			return errMsg{scope: "params", err: err}
		}
		return paramsLoadedMsg{jobPath: jobPath, params: params}
	}
}

func startRunCmd(c *jenkins.Client, jobPath string, params map[string]string) tea.Cmd {
	return func() tea.Msg {
		out, err := c.RunStart(context.Background(), jobPath, params)
		if err != nil {
			return errMsg{scope: "action", err: err}
		}
		return actionResultMsg{label: "started " + jobPath, output: out}
	}
}

func rerunCmd(c *jenkins.Client, jobPath string, number int) tea.Cmd {
	return func() tea.Msg {
		out, err := c.RunRerun(context.Background(), jobPath, number)
		if err != nil {
			return errMsg{scope: "action", err: err}
		}
		return actionResultMsg{label: "rerunning", output: out}
	}
}

func cancelRunCmd(c *jenkins.Client, jobPath string, number int) tea.Cmd {
	return func() tea.Msg {
		out, err := c.RunCancel(context.Background(), jobPath, number)
		if err != nil {
			return errMsg{scope: "action", err: err}
		}
		return actionResultMsg{label: "cancelled", output: out}
	}
}

func fetchLogCmd(c *jenkins.Client, jobPath string, number, gen int) tea.Cmd {
	return func() tea.Msg {
		content, err := c.LogFull(context.Background(), jobPath, number)
		if err != nil {
			return errMsg{scope: "log", err: err}
		}
		return logLoadedMsg{gen: gen, content: content}
	}
}

func startLogFollowCmd(c *jenkins.Client, jobPath string, number, gen int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		ch, err := c.LogFollow(ctx, jobPath, number)
		if err != nil {
			cancel()
			return errMsg{scope: "log", err: err}
		}
		return logStreamStartedMsg{gen: gen, ch: ch, cancel: cancel}
	}
}

func waitForLogLineCmd(gen int, ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return logStreamDoneMsg{gen: gen}
		}
		return logLineMsg{gen: gen, line: line, ch: ch}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(15*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
