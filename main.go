package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "claude-hook" {
		// A hook error must not get in the way of Claude Code, so it is silent.
		_ = runClaudeHook()
		return
	}

	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cfg := LoadConfig()
	m := initialModel(cfg, root)

	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running sidegit: %v\n", err)
		os.Exit(1)
	}
}
