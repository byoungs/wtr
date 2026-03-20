package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/byoungs/wtr/internal/git"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (a App) updateFileList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case diffLoadedMsg:
		a.files = msg.files
		a.selectedFile = 0
		return a, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Up):
			if a.selectedFile > 0 {
				a.selectedFile--
			}
		case key.Matches(msg, keys.Down):
			if a.selectedFile < len(a.files)-1 {
				a.selectedFile++
			}
		case key.Matches(msg, keys.Enter):
			if len(a.files) > 0 {
				a.screen = screenDiffView
			}
		case key.Matches(msg, keys.Space):
			if len(a.files) > 0 {
				k := a.reviewKey(a.selectedFile)
				a.reviewed[k] = !a.reviewed[k]
			}
		case key.Matches(msg, keys.Back):
			a.screen = screenWorktreeList
		}
	}
	return a, nil
}

func (a App) reviewKey(fileIdx int) string {
	wt := a.worktrees[a.selectedWorktree]
	f := a.files[fileIdx]
	name := f.NewName
	if name == "" || name == "/dev/null" {
		name = f.OldName
	}
	return wt.Branch + ":" + name
}

func (a App) viewFileList() string {
	var b strings.Builder

	wt := a.worktrees[a.selectedWorktree]
	title := styleTitle.Width(a.width).Render(fmt.Sprintf("Files — %s", wt.Branch))
	b.WriteString(title + "\n\n")

	// Count reviewed
	reviewedCount := 0
	for i := range a.files {
		if a.reviewed[a.reviewKey(i)] {
			reviewedCount++
		}
	}
	b.WriteString(fmt.Sprintf("  %d/%d reviewed\n\n", reviewedCount, len(a.files)))

	if len(a.files) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(colorSubtle).Render(
			"  No changes found.\n"))
		b.WriteString("\n")
		b.WriteString(styleHelp.Render("  esc: back"))
		return b.String()
	}

	for i, f := range a.files {
		cursor := "  "
		if i == a.selectedFile {
			cursor = "→ "
		}

		check := "○"
		if a.reviewed[a.reviewKey(i)] {
			check = styleReviewed.Render("✓")
		}

		name := f.NewName
		if name == "" || name == "/dev/null" {
			name = f.OldName + " (deleted)"
		}
		if f.OldName == "/dev/null" {
			name = f.NewName + " (new)"
		}

		// Count added/removed lines
		var added, removed int
		for _, h := range f.Hunks {
			for _, l := range h.Lines {
				switch l.Type {
				case git.LineAdded:
					added++
				case git.LineRemoved:
					removed++
				}
			}
		}
		stats := fmt.Sprintf("+%d -%d", added, removed)

		// Shorten path: dim the directory part
		dir := filepath.Dir(name)
		base := filepath.Base(name)
		var displayName string
		if dir == "." {
			displayName = base
		} else {
			displayName = lipgloss.NewStyle().Foreground(colorSubtle).Render(dir+"/") + base
		}

		line := fmt.Sprintf("%s%s %s  %s", cursor, check, displayName, stats)
		b.WriteString(line + "\n")
	}

	b.WriteString("\n")
	b.WriteString(styleHelp.Render("  enter: view diff  space: mark reviewed  esc: back"))

	return b.String()
}
