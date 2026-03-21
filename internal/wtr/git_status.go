package wtr

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/byoungs/wtr/internal/git"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type statusEntry struct {
	Status string // e.g. "??", " M", "A ", "MM"
	Path   string // relative file path
}

func (e statusEntry) IsUntracked() bool {
	return e.Status == "??"
}

func (e statusEntry) IsModified() bool {
	return strings.Contains(e.Status, "M") || strings.Contains(e.Status, "D")
}

func (e statusEntry) Label() string {
	switch {
	case e.Status == "??":
		return "untracked"
	case strings.Contains(e.Status, "D"):
		return "deleted"
	case strings.Contains(e.Status, "A"):
		return "added"
	default:
		return "modified"
	}
}

func loadGitStatus(worktreePath string) []statusEntry {
	out, err := exec.Command("git", "-C", worktreePath, "status", "--short").Output()
	if err != nil {
		return nil
	}
	var entries []statusEntry
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if len(line) < 4 {
			continue
		}
		entries = append(entries, statusEntry{
			Status: line[:2],
			Path:   strings.TrimSpace(line[2:]),
		})
	}
	return entries
}

var statusScrollY int

func (a App) updateGitStatus(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if a.confirmRevert {
			if msg.String() == "y" {
				a.confirmRevert = false
				wt := a.worktrees[a.selectedWorktree]
				entry := a.statusFiles[a.statusCursor]
				fullPath := filepath.Join(wt.Path, entry.Path)

				if entry.IsUntracked() {
					if err := os.RemoveAll(fullPath); err != nil {
						a.err = fmt.Errorf("delete %s: %v", entry.Path, err)
						a.confirmRevert = false
						return a, nil
					}
				} else {
					out, err := exec.Command("git", "-C", wt.Path, "checkout", "--", entry.Path).CombinedOutput()
					if err != nil {
						a.err = fmt.Errorf("revert %s: %s", entry.Path, strings.TrimSpace(string(out)))
						a.confirmRevert = false
						return a, nil
					}
				}

				// Reload status
				a.statusFiles = loadGitStatus(wt.Path)
				if a.statusCursor >= len(a.statusFiles) {
					a.statusCursor = max(0, len(a.statusFiles)-1)
				}
				return a, nil
			}
			a.confirmRevert = false
			return a, nil
		}

		switch {
		case key.Matches(msg, keys.Up):
			if a.statusCursor > 0 {
				a.statusCursor--
			}
		case key.Matches(msg, keys.Down):
			if a.statusCursor < len(a.statusFiles)-1 {
				a.statusCursor++
			}
		case key.Matches(msg, keys.Revert):
			if len(a.statusFiles) > 0 {
				a.confirmRevert = true
			}
		case key.Matches(msg, keys.Right), key.Matches(msg, keys.Enter):
			if len(a.statusFiles) > 0 {
				wt := a.worktrees[a.selectedWorktree]
				// Load diffs for ALL status files so ]/[ navigation works
				var allFiles []git.FileDiff
				for _, entry := range a.statusFiles {
					diffs, err := git.GetWorkingDiff(wt.Path, entry.Path)
					if err != nil {
						continue
					}
					allFiles = append(allFiles, diffs...)
				}
				a.files = allFiles
				a.selectedFile = a.statusCursor
				if a.selectedFile >= len(a.files) {
					a.selectedFile = 0
				}
				a.prevScreen = screenGitStatus
				a.screen = screenDiffView
				diffScrollY = 0
				return a, nil
			}
		case key.Matches(msg, keys.Open):
			if len(a.statusFiles) > 0 {
				wt := a.worktrees[a.selectedWorktree]
				entry := a.statusFiles[a.statusCursor]
				fullPath := filepath.Join(wt.Path, entry.Path)
				exec.Command("code", "--goto", fullPath).Start()
			}
		case key.Matches(msg, keys.Back):
			a.screen = screenFileList
			statusScrollY = 0
		}
	}
	return a, nil
}

func (a App) viewGitStatus() string {
	var b strings.Builder

	wt := a.worktrees[a.selectedWorktree]
	title := styleTitle.Width(a.width).Render(fmt.Sprintf("Git Status — %s", wt.Branch))
	b.WriteString(title + "\n")

	if len(a.statusFiles) == 0 {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(colorSubtle).Render("  Working tree clean.\n"))
		b.WriteString("\n")
		b.WriteString(styleHelp.Render("  esc: back"))
		return b.String()
	}

	var lines []string
	for i, entry := range a.statusFiles {
		cursor := "  "
		if i == a.statusCursor {
			cursor = "→ "
		}

		var statusStyle lipgloss.Style
		switch {
		case entry.IsUntracked():
			statusStyle = lipgloss.NewStyle().Foreground(colorYellow)
		case strings.Contains(entry.Status, "D"):
			statusStyle = lipgloss.NewStyle().Foreground(colorRed)
		default:
			statusStyle = lipgloss.NewStyle().Foreground(colorGreen)
		}

		label := lipgloss.NewStyle().Foreground(colorSubtle).Render(fmt.Sprintf("%-10s", entry.Label()))
		line := fmt.Sprintf("%s%s %s %s", cursor, statusStyle.Render(entry.Status), label, entry.Path)
		lines = append(lines, line)
	}

	// Viewport
	viewHeight := a.height - 5
	if viewHeight < 1 {
		viewHeight = 1
	}

	if a.statusCursor < statusScrollY {
		statusScrollY = a.statusCursor
	}
	if a.statusCursor >= statusScrollY+viewHeight {
		statusScrollY = a.statusCursor - viewHeight + 1
	}
	if statusScrollY > len(lines)-viewHeight {
		statusScrollY = max(0, len(lines)-viewHeight)
	}

	end := statusScrollY + viewHeight
	if end > len(lines) {
		end = len(lines)
	}
	for _, line := range lines[statusScrollY:end] {
		b.WriteString(line + "\n")
	}

	b.WriteString("\n")

	if a.confirmRevert {
		entry := a.statusFiles[a.statusCursor]
		action := "revert"
		if entry.IsUntracked() {
			action = "delete"
		}
		b.WriteString(styleFail.Render(fmt.Sprintf("  %s %s? (y/n)", action, entry.Path)) + "\n")
	}

	b.WriteString(styleHelp.Render("  q:quit  ←back  →view  del:revert  o:open"))

	return b.String()
}
