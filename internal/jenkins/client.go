package jenkins

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Client shells out to the `jk` binary. All calls are blocking; callers
// (bubbletea commands) run them off the UI goroutine.
type Client struct {
	Bin     string
	Context string
}

func New(context string) *Client {
	return &Client{Bin: "jk", Context: context}
}

func (c *Client) args(a ...string) []string {
	out := []string{}
	if c.Context != "" {
		out = append(out, "-c", c.Context)
	}
	out = append(out, a...)
	return out
}

func (c *Client) run(ctx context.Context, a ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, c.Bin, c.args(a...)...)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return out, fmt.Errorf("%s: %s", err, strings.TrimSpace(string(ee.Stderr)))
		}
		return out, err
	}
	return out, nil
}

func (c *Client) AuthStatus(ctx context.Context) (AuthStatus, error) {
	out, err := c.run(ctx, "auth", "status")
	var st AuthStatus
	if err != nil {
		return st, err
	}
	for _, line := range strings.Split(string(out), "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		switch strings.TrimSpace(k) {
		case "Active context":
			st.Context = v
		case "URL":
			st.URL = v
		case "Username":
			st.Username = v
		}
	}
	return st, nil
}

func (c *Client) JobLs(ctx context.Context, folder string) ([]Job, error) {
	a := []string{"job", "ls", "--json"}
	if folder != "" {
		a = append(a, "--folder", folder)
	}
	out, err := c.run(ctx, a...)
	if err != nil {
		return nil, err
	}
	var jobs []Job
	if err := json.Unmarshal(out, &jobs); err != nil {
		return nil, fmt.Errorf("parsing job ls output: %w", err)
	}
	return jobs, nil
}

func (c *Client) RunLs(ctx context.Context, jobPath string, limit int) ([]Run, error) {
	out, err := c.run(ctx, "run", "ls", jobPath, "--json", "--limit", fmt.Sprint(limit), "--include-queued")
	if err != nil {
		return nil, err
	}
	var list RunList
	if err := json.Unmarshal(out, &list); err != nil {
		return nil, fmt.Errorf("parsing run ls output: %w", err)
	}
	return list.Items, nil
}

func (c *Client) RunParams(ctx context.Context, jobPath string) ([]Param, error) {
	out, err := c.run(ctx, "run", "params", jobPath, "--json")
	if err != nil {
		return nil, err
	}
	var list ParamList
	if err := json.Unmarshal(out, &list); err != nil {
		return nil, fmt.Errorf("parsing run params output: %w", err)
	}
	return list.Parameters, nil
}

func (c *Client) RunStart(ctx context.Context, jobPath string, params map[string]string) (string, error) {
	a := []string{"run", "start", jobPath, "--non-interactive"}
	for k, v := range params {
		a = append(a, "-p", k+"="+v)
	}
	out, err := c.run(ctx, a...)
	return string(out), err
}

func (c *Client) RunRerun(ctx context.Context, jobPath string, buildNumber int) (string, error) {
	out, err := c.run(ctx, "run", "rerun", jobPath, fmt.Sprint(buildNumber))
	return string(out), err
}

func (c *Client) RunCancel(ctx context.Context, jobPath string, buildNumber int) (string, error) {
	out, err := c.run(ctx, "run", "cancel", jobPath, fmt.Sprint(buildNumber))
	return string(out), err
}

// LogFull fetches the complete console log for a finished (or in-progress)
// run in one shot, with ANSI/fold noise cleaned up for display.
func (c *Client) LogFull(ctx context.Context, jobPath string, buildNumber int) (string, error) {
	out, err := c.run(ctx, "log", jobPath, fmt.Sprint(buildNumber), "--plain")
	if err != nil {
		return "", err
	}
	return CleanLog(string(out)), nil
}

// LogFollow streams a run's console log line by line until the run finishes
// or ctx is cancelled. Cleaned lines are sent on the returned channel, which
// is closed when the underlying process exits.
func (c *Client) LogFollow(ctx context.Context, jobPath string, buildNumber int) (<-chan string, error) {
	cmd := exec.CommandContext(ctx, c.Bin, c.args("log", jobPath, fmt.Sprint(buildNumber), "--plain", "--follow")...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	lines := make(chan string, 256)
	go func() {
		defer close(lines)
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		for scanner.Scan() {
			lines <- CleanLine(scanner.Text())
		}
		_ = cmd.Wait()
	}()
	return lines, nil
}

func (c *Client) QueueLs(ctx context.Context) ([]byte, error) {
	return c.run(ctx, "queue", "ls", "--json")
}

func (c *Client) QueueCancel(ctx context.Context, id string) (string, error) {
	out, err := c.run(ctx, "queue", "cancel", id)
	return string(out), err
}
