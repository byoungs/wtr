package wtr

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/byoungs/wtr/internal/git"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var fileListScrollY int

// totalFileItems returns the number of navigable items: files + optional status row
func (a App) totalFileItems() int {
	n := len(a.files)
	if len(a.statusFiles) > 0 {
		n++
	}
	return n
}

// onStatusRow returns true if cursor is on the uncommitted changes row
func (a App) onStatusRow() bool {
	return len(a.statusFiles) > 0 && a.selectedFile == len(a.files)
}

func (a App) updateFileList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case diffLoadedMsg:
		a.files = msg.files
		a.selectedFile = 0
		fileListScrollY = 0
		// Load git status for this worktree
		wt := a.worktrees[a.selectedWorktree]
		a.statusFiles = loadGitStatus(wt.Path)
		return a, nil

	case tea.KeyMsg:
		// Search mode: capture typed characters
		if a.searching {
			switch msg.String() {
			case "esc":
				a.searching = false
				a.searchQuery = ""
				a.selectedFile = 0
				fileListScrollY = 0
			case "enter":
				a.searching = false
			case "backspace":
				if len(a.searchQuery) > 0 {
					a.searchQuery = a.searchQuery[:len(a.searchQuery)-1]
					a.selectedFile = 0
					fileListScrollY = 0
				}
			default:
				if len(msg.String()) == 1 {
					a.searchQuery += msg.String()
					a.selectedFile = 0
					fileListScrollY = 0
				}
			}
			return a, nil
		}

		switch {
		case key.Matches(msg, keys.Search):
			a.searching = true
			a.searchQuery = ""
			a.selectedFile = 0
			fileListScrollY = 0
			return a, nil
		case key.Matches(msg, keys.Up):
			if a.selectedFile > 0 {
				a.selectedFile--
			}
		case key.Matches(msg, keys.Down):
			if a.selectedFile < a.totalFileItems()-1 {
				a.selectedFile++
			}
		case key.Matches(msg, keys.Enter), key.Matches(msg, keys.Right):
			if a.onStatusRow() {
				a.statusCursor = 0
				a.confirmRevert = false
				a.screen = screenGitStatus
				return a, nil
			}
			if len(a.files) > 0 && a.selectedFile < len(a.files) {
				a.screen = screenDiffView
			}
		case key.Matches(msg, keys.MarkReviewed):
			if len(a.files) > 0 && a.selectedFile < len(a.files) {
				k := a.reviewKey(a.selectedFile)
				a.reviewed[k] = !a.reviewed[k]
				a.saveState()
			}
		case key.Matches(msg, keys.Open):
			if len(a.files) > 0 && a.selectedFile < len(a.files) {
				wt := a.worktrees[a.selectedWorktree]
				f := a.files[a.selectedFile]
				name := f.NewName
				if name == "" || name == "/dev/null" {
					name = f.OldName
				}
				filePath := filepath.Join(wt.Path, name)
				exec.Command("code", "--goto", filePath).Start()
			}
		case key.Matches(msg, keys.AllDiffs):
			if len(a.files) > 0 {
				a.screen = screenAllDiffs
			}
		case key.Matches(msg, keys.GitStatus):
			if len(a.statusFiles) > 0 {
				a.statusCursor = 0
				a.confirmRevert = false
				a.screen = screenGitStatus
				return a, nil
			}
		case key.Matches(msg, keys.Back):
			a.searching = false
			a.searchQuery = ""
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
	titleText := fmt.Sprintf("Files — %s", wt.Branch)
	if a.searchQuery != "" {
		titleText += fmt.Sprintf("  [search: %s]", a.searchQuery)
	}
	title := styleTitle.Width(a.width).Render(titleText)
	b.WriteString(title + "\n")

	if len(a.files) == 0 && len(a.statusFiles) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(colorSubtle).Render(
			"  No changes found.\n"))
		b.WriteString("\n")
		b.WriteString(styleHelp.Render("  ←/esc: back"))
		return b.String()
	}

	// Filter files by search query
	visibleFiles := a.files
	var visibleIndices []int
	if a.searchQuery != "" {
		var names []string
		for _, f := range a.files {
			name := f.NewName
			if name == "" || name == "/dev/null" {
				name = f.OldName
			}
			names = append(names, name)
		}
		filtered := filterFiles(names, a.searchQuery)
		filteredSet := make(map[string]bool, len(filtered))
		for _, n := range filtered {
			filteredSet[n] = true
		}
		visibleFiles = nil
		for i, f := range a.files {
			name := f.NewName
			if name == "" || name == "/dev/null" {
				name = f.OldName
			}
			if filteredSet[name] {
				visibleFiles = append(visibleFiles, f)
				visibleIndices = append(visibleIndices, i)
			}
		}
	} else {
		for i := range a.files {
			visibleIndices = append(visibleIndices, i)
		}
	}

	// Build all lines
	var lines []string
	for vi, f := range visibleFiles {
		origIdx := visibleIndices[vi]
		cursor := "  "
		if vi == a.selectedFile {
			cursor = "→ "
		}

		check := "○"
		if a.reviewed[a.reviewKey(origIdx)] {
			check = styleReviewed.Render("✓")
		}

		name := f.NewName
		if name == "" || name == "/dev/null" {
			name = f.OldName + " (deleted)"
		}
		if f.OldName == "/dev/null" {
			name = f.NewName + " (new)"
		}

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

		dir := filepath.Dir(name)
		base := filepath.Base(name)
		var displayName string
		if dir == "." {
			displayName = base
		} else {
			displayName = lipgloss.NewStyle().Foreground(colorSubtle).Render(dir+"/") + base
		}

		line := fmt.Sprintf("%s%s %s  %s", cursor, check, displayName, stats)
		lines = append(lines, line)
	}

	// Add uncommitted changes as a navigable row
	if len(a.statusFiles) > 0 {
		lines = append(lines, "") // blank separator
		cursor := "  "
		if a.onStatusRow() {
			cursor = "→ "
		}
		statusLine := styleRunning.Render(
			fmt.Sprintf("%s△ %d uncommitted changes", cursor, len(a.statusFiles))) +
			styleHelp.Render("  (g or →/enter)")
		lines = append(lines, statusLine)
	}

	// Viewport
	overhead := 4 // title+border(2) + help(2)
	viewHeight := a.height - overhead
	if viewHeight < 1 {
		viewHeight = 1
	}

	// Keep cursor visible — use actual cursor position mapped to line index
	cursorLine := a.selectedFile
	if a.onStatusRow() {
		cursorLine = len(lines) - 1 // last line
	}
	if cursorLine < fileListScrollY {
		fileListScrollY = cursorLine
	}
	if cursorLine >= fileListScrollY+viewHeight {
		fileListScrollY = cursorLine - viewHeight + 1
	}
	if fileListScrollY > len(lines)-viewHeight {
		fileListScrollY = max(0, len(lines)-viewHeight)
	}

	end := fileListScrollY + viewHeight
	if end > len(lines) {
		end = len(lines)
	}
	for _, line := range lines[fileListScrollY:end] {
		b.WriteString(line + "\n")
	}

	padToBottom(&b, a.height, strings.Count(b.String(), "\n"))
	if a.searching {
		b.WriteString(styleHelp.Render(fmt.Sprintf("  search: %s█  (enter: confirm  esc: cancel)", a.searchQuery)))
	} else {
		b.WriteString(styleHelp.Render("  q:quit  ←back  →view  o:open  a:all diffs  x:reviewed  /:search  g:status"))
	}

	return b.String()
}
