package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lazyjenkins/internal/jenkins"
)

// extraCustomRows is how many blank "key=value" rows are always appended
// after a job's known parameters, so params jk couldn't discover (e.g. a
// job with no build history yet, or a Jenkinsfile-declared parameter jk
// hasn't indexed) can still be supplied by hand.
const extraCustomRows = 4

// paramsForm is a tiny sequential form: one textinput per known job
// parameter, plus a few blank "key=value" rows for custom ones.
type paramsForm struct {
	jobPath string
	defs    []jenkins.Param // len(defs) of inputs are named params; the rest are custom key=value rows
	inputs  []textinput.Model
	active  int
}

func newParamsForm(jobPath string, defs []jenkins.Param) paramsForm {
	f := paramsForm{jobPath: jobPath, defs: defs}
	f.inputs = make([]textinput.Model, len(defs)+extraCustomRows)
	for i, d := range defs {
		ti := textinput.New()
		ti.Prompt = ""
		ti.SetValue(d.Default)
		if len(d.SampleValues) > 0 {
			ti.Placeholder = strings.Join(d.SampleValues, ", ")
		}
		if d.IsSecret {
			ti.EchoMode = textinput.EchoPassword
		}
		f.inputs[i] = ti
	}
	for i := len(defs); i < len(f.inputs); i++ {
		ti := textinput.New()
		ti.Prompt = ""
		ti.Placeholder = "KEY=VALUE"
		f.inputs[i] = ti
	}
	f.inputs[0].Focus()
	return f
}

func (f *paramsForm) focus(i int) {
	for j := range f.inputs {
		if j == i {
			f.inputs[j].Focus()
		} else {
			f.inputs[j].Blur()
		}
	}
	f.active = i
}

// Update returns true when the user submitted the form (enter on the last
// field) and false otherwise. Tab/shift+tab/up/down move between fields.
func (f *paramsForm) Update(msg tea.Msg) (submitted bool, cmd tea.Cmd) {
	switch m := msg.(type) {
	case tea.KeyMsg:
		switch m.String() {
		case "tab", "down":
			f.focus((f.active + 1) % len(f.inputs))
			return false, nil
		case "shift+tab", "up":
			f.focus((f.active - 1 + len(f.inputs)) % len(f.inputs))
			return false, nil
		case "enter":
			if f.active == len(f.inputs)-1 {
				return true, nil
			}
			f.focus(f.active + 1)
			return false, nil
		}
	}
	var c tea.Cmd
	f.inputs[f.active], c = f.inputs[f.active].Update(msg)
	return false, c
}

func (f paramsForm) values() map[string]string {
	out := map[string]string{}
	for i, d := range f.defs {
		v := f.inputs[i].Value()
		if v != "" {
			out[d.Name] = v
		}
	}
	for i := len(f.defs); i < len(f.inputs); i++ {
		key, val, ok := strings.Cut(f.inputs[i].Value(), "=")
		key = strings.TrimSpace(key)
		if ok && key != "" {
			out[key] = strings.TrimSpace(val)
		}
	}
	return out
}

var (
	formLabelStyle  = lipgloss.NewStyle().Foreground(colorSubtle).Width(24)
	formActiveLabel = lipgloss.NewStyle().Foreground(colorFocused).Bold(true).Width(24)
)

func (f paramsForm) View() string {
	var b strings.Builder
	b.WriteString(modalTitleStyle.Render("Start run: "+f.jobPath) + "\n\n")
	for i, d := range f.defs {
		label := d.Name
		if d.Type != "" {
			label += " (" + d.Type + ")"
		}
		style := formLabelStyle
		if i == f.active {
			style = formActiveLabel
		}
		b.WriteString(style.Render(label) + " " + f.inputs[i].View() + "\n")
	}
	if len(f.defs) > 0 {
		b.WriteString("\n")
	}
	for i := len(f.defs); i < len(f.inputs); i++ {
		style := formLabelStyle
		if i == f.active {
			style = formActiveLabel
		}
		b.WriteString(style.Render("+ custom param") + " " + f.inputs[i].View() + "\n")
	}
	b.WriteString("\n" + keyHintStyle.Render(" tab/↑↓ move · enter next/submit · esc cancel "))
	return b.String()
}
