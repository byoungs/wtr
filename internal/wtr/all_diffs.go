package wtr

import (
	"fmt"
	"strings"

	"github.com/byoungs/wtr/internal/git"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var allDiffsScrollY int

func (a App) updateAllDiffs(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Up):
			if allDiffsScrollY > 0 {
				allDiffsScrollY--
			}
		case key.Matches(msg, keys.Down):
			allDiffsScrollY++
		case key.Matches(msg, keys.PageDown):
			viewHeight := a.height - 3
			if viewHeight < 1 {
				viewHeight = 1
			}
			allDiffsScrollY += viewHeight
		case key.Matches(msg, keys.PageUp):
			viewHeight := a.height - 3
			if viewHeight < 1 {
				viewHeight = 1
			}
			allDiffsScrollY -= viewHeight
			if allDiffsScrollY < 0 {
				allDiffsScrollY = 0
			}
		case key.Matches(msg, keys.Toggle):
			a.sideBySide = !a.sideBySide
			allDiffsScrollY = 0
		case key.Matches(msg, keys.Back):
			a.screen = screenFileList
			allDiffsScrollY = 0
		}
	}
	return a, nil
}

func (a App) viewAllDiffs() string {
	wt := a.worktrees[a.selectedWorktree]

	var b strings.Builder

	title := styleTitle.Width(a.width).Render(
		fmt.Sprintf("All diffs — %s  (%d files)", wt.Branch, len(a.files)))
	b.WriteString(title + "\n")

	wrap := a.width >= minWrapWidth
	contentWidth := a.width - unifiedGutterWidth

	// Build all rendered lines across all files
	var allLines []string

	fileHeaderStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(colorYellow).
		Background(lipgloss.Color("#1e1e2e")).
		Width(a.width)

	for _, f := range a.files {
		name := f.NewName
		if name == "" || name == "/dev/null" {
			name = f.OldName + " (deleted)"
		}
		if f.OldName == "/dev/null" {
			name = f.NewName + " (new)"
		}

		// Count stats
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

		// File header separator
		header := fileHeaderStyle.Render(
			fmt.Sprintf("━━ %s  +%d -%d", name, added, removed))
		allLines = append(allLines, header)

		if f.Binary {
			allLines = append(allLines, "  Binary file")
			allLines = append(allLines, "")
			continue
		}

		// Render each hunk in unified format
		for _, hunk := range f.Hunks {
			allLines = append(allLines, styleHunkHeader.Render(hunk.Header))
			oldNum := hunk.OldStart
			newNum := hunk.NewStart

			for _, line := range hunk.Lines {
				var prefix, numStr string
				switch line.Type {
				case git.LineContext:
					prefix = " "
					numStr = fmt.Sprintf("%4d %4d", oldNum, newNum)
					oldNum++
					newNum++
				case git.LineRemoved:
					prefix = "-"
					numStr = fmt.Sprintf("%4d     ", oldNum)
					oldNum++
				case git.LineAdded:
					prefix = "+"
					numStr = fmt.Sprintf("     %4d", newNum)
					newNum++
				}

				allLines = append(allLines, renderUnifiedDiffLine(numStr, prefix, line.Content, line.Type, contentWidth, wrap)...)
			}
		}

		// Blank line between files
		allLines = append(allLines, "")
	}

	// Apply scroll and viewport
	viewHeight := a.height - 3
	if viewHeight < 1 {
		viewHeight = 1
	}
	maxScroll := len(allLines) - viewHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if allDiffsScrollY > maxScroll {
		allDiffsScrollY = maxScroll
	}
	end := allDiffsScrollY + viewHeight
	if end > len(allLines) {
		end = len(allLines)
	}

	for _, line := range allLines[allDiffsScrollY:end] {
		b.WriteString(line + "\n")
	}

	padToBottom(&b, a.height, strings.Count(b.String(), "\n"))
	b.WriteString(styleHelp.Render("  q:quit  ←back  space/b:page  v:toggle"))

	return b.String()
}
