package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

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

	binPath := resolveJKBin(*jkBin)

	// Any positional argument means "run this jk subcommand", not "launch
	// the TUI" — e.g. `lazyjenkins auth login <url> --token X` sets up
	// jk's own context (URL, token in the OS keychain) without needing a
	// separate jk install. Every jk subcommand works this way, not just
	// auth: we're just forwarding to whichever jk we'd otherwise exec
	// internally, stdio connected straight through.
	if args := flag.Args(); len(args) > 0 {
		os.Exit(runJKPassthrough(binPath, args))
	}

	// A context the user already pinned — via flag or jk's own env var —
	// means skip the picker; jk resolves JK_CONTEXT itself from our
	// subprocess's inherited environment, so we don't need to thread it
	// through client.Context ourselves.
	explicitContext := *context
	if explicitContext == "" {
		explicitContext = os.Getenv("JK_CONTEXT")
	}

	client := jenkins.New(*context)
	client.Bin = binPath
	m := ui.New(client, *debug, explicitContext)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "lazyjenkins:", err)
		os.Exit(1)
	}
}

// runJKPassthrough execs binPath with args, stdio connected straight
// through, and returns the exit code to propagate.
func runJKPassthrough(binPath string, args []string) int {
	cmd := exec.Command(binPath, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode()
		}
		fmt.Fprintln(os.Stderr, "lazyjenkins:", err)
		return 1
	}
	return 0
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
