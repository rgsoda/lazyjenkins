package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"lazyjenkins/internal/embedjk"
	"lazyjenkins/internal/jenkins"
	"lazyjenkins/internal/ui"
)

func main() {
	context := flag.String("context", "", "Jenkins context to use (defaults to jk's active context)")
	debug := flag.Bool("debug", false, "show a [4] Debug panel logging every jk invocation (args, timing, errors)")
	jkBin := flag.String("jk-bin", "", "path to a jk binary to use (defaults to the bundled copy, falling back to jk on PATH)")
	flag.Parse()

	client := jenkins.New(*context)
	client.Bin = resolveJKBin(*jkBin)
	m := ui.New(client, *debug)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "lazyjenkins:", err)
		os.Exit(1)
	}
}

// resolveJKBin picks which jk binary to exec: an explicit --jk-bin always
// wins, then the bundled copy (release builds only — see internal/embedjk),
// falling back to whatever "jk" resolves to on PATH.
func resolveJKBin(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if path, err := embedjk.Extract(); err == nil {
		return path
	}
	return "jk"
}
