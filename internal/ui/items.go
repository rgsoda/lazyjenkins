package ui

import (
	"fmt"
	"strings"
	"time"

	"lazyjenkins/internal/jenkins"
)

// jobItem and runItem are deliberately thin: list.Item only requires
// FilterValue(). Display rendering (which needs embedded ANSI color) happens
// in rowDelegate.Render, not here — bubbles/list's DefaultDelegate expects
// plain-text Title()/Description() and mangles embedded escape codes when it
// fuzzy-highlights matches during filtering, so we don't use it.
type jobItem struct {
	job jenkins.Job
}

func (i jobItem) FilterValue() string { return i.job.Name }

type runItem struct {
	run jenkins.Run
}

func (i runItem) FilterValue() string {
	return fmt.Sprintf("%d %s %s", i.run.Number, i.run.Result, i.run.Branch)
}

func runResultLabel(r jenkins.Run) string {
	if r.Result != "" {
		return r.Result
	}
	return strings.ToUpper(r.Status)
}

func runDescription(r jenkins.Run) string {
	parts := []string{}
	if r.Branch != "" {
		parts = append(parts, r.Branch)
	}
	if t, err := time.Parse(time.RFC3339, r.StartTime); err == nil {
		parts = append(parts, humanAgo(t))
	}
	if r.DurationMs > 0 {
		parts = append(parts, (time.Duration(r.DurationMs) * time.Millisecond).Round(time.Second).String())
	}
	return strings.Join(parts, " · ")
}

func humanAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
