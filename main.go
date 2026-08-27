package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"lazyjenkins/internal/jenkins"
	"lazyjenkins/internal/ui"
)

func main() {
	context := flag.String("context", "", "Jenkins context to use (defaults to jk's active context)")
	debug := flag.Bool("debug", false, "show a [4] Debug panel logging every jk invocation (args, timing, errors)")
	flag.Parse()

	client := jenkins.New(*context)
	m := ui.New(client, *debug)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "lazyjenkins:", err)
		os.Exit(1)
	}
}
