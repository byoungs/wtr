package main

import (
	"fmt"
	"os"

	"github.com/byoungs/wtr/internal/wtr"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	repoDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if len(os.Args) > 1 {
		repoDir = os.Args[1]
	}

	app := wtr.NewApp(repoDir)
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
