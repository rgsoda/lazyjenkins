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
	flag.Parse()

	client := jenkins.New(*context)
	m := ui.New(client)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "lazyjenkins:", err)
		os.Exit(1)
	}
}
