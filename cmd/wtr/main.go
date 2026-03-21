package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/byoungs/wtr/internal/wtr"
	tea "github.com/charmbracelet/bubbletea"
)

// resolveMainWorktree finds the main worktree root from any worktree or repo dir.
func resolveMainWorktree(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--git-common-dir")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return dir, nil // not a git repo, let it fail later
	}
	gitDir := strings.TrimSpace(string(out))
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(dir, gitDir)
	}
	// .git dir's parent is the main worktree root
	return filepath.Dir(gitDir), nil
}

func main() {
	repoDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if len(os.Args) > 1 {
		repoDir = os.Args[1]
	}

	repoDir, err = resolveMainWorktree(repoDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	app := wtr.NewApp(repoDir)
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
