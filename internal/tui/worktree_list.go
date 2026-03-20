package tui

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/byoungs/wtr/internal/git"
	"github.com/byoungs/wtr/internal/land"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (a App) updateWorktreeList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case worktreesLoadedMsg:
		a.worktrees = msg.worktrees
		return a, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Up):
			if a.selectedWorktree > 0 {
				a.selectedWorktree--
			}
		case key.Matches(msg, keys.Down):
			if a.selectedWorktree < len(a.worktrees)-1 {
				a.selectedWorktree++
			}
		case key.Matches(msg, keys.Enter):
			if len(a.worktrees) > 0 {
				a.screen = screenFileList
				return a, a.loadDiff()
			}
		case key.Matches(msg, keys.Test):
			if len(a.worktrees) > 0 {
				wt := a.worktrees[a.selectedWorktree]
				a.testStatus[wt.Branch] = 1 // running
				return a, func() tea.Msg {
					cmd := exec.Command("make", "test")
					cmd.Dir = wt.Path
					out, err := cmd.CombinedOutput()
					return testDoneMsg{
						branch: wt.Branch,
						passed: err == nil,
						output: string(out),
					}
				}
			}
		case key.Matches(msg, keys.Land):
			if len(a.worktrees) > 0 && !a.landing {
				wt := a.worktrees[a.selectedWorktree]
				a.landing = true
				a.landBranch = wt.Branch
				a.landStep = "starting..."
				return a, func() tea.Msg {
					_, err := land.Run(a.repoDir, wt.Branch, func(s land.Step) {
						// Can't easily send intermediate messages from here,
						// so we just track final result
					})
					return landDoneMsg{err: err}
				}
			}
		case key.Matches(msg, keys.Delete):
			if len(a.worktrees) > 0 {
				if a.confirmDelete {
					// Confirmed — do the delete
					wt := a.worktrees[a.selectedWorktree]
					a.confirmDelete = false
					return a, func() tea.Msg {
						exec.Command("git", "-C", a.repoDir, "worktree", "remove", wt.Path).Run()
						exec.Command("git", "-C", a.repoDir, "branch", "-D", wt.Branch).Run()
						return worktreeDeletedMsg{}
					}
				} else {
					a.confirmDelete = true
				}
			}
		case key.Matches(msg, keys.Back):
			if a.confirmDelete {
				a.confirmDelete = false
			}
		}
	}
	return a, nil
}

func (a App) loadDiff() tea.Cmd {
	wt := a.worktrees[a.selectedWorktree]
	return func() tea.Msg {
		files, err := git.GetDiff(wt.Path)
		if err != nil {
			return errMsg{err}
		}
		return diffLoadedMsg{files}
	}
}

func (a App) viewWorktreeList() string {
	var b strings.Builder

	title := styleTitle.Width(a.width).Render("Worktrees")
	b.WriteString(title + "\n\n")

	if a.err != nil {
		b.WriteString(lipgloss.NewStyle().Foreground(colorRed).Render(
			fmt.Sprintf("  Error: %v\n\n", a.err)))
	}

	if len(a.worktrees) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(colorSubtle).Render(
			"  No worktrees found.\n"))
		b.WriteString("\n")
		b.WriteString(styleHelp.Render("  q: quit"))
		return b.String()
	}

	for i, wt := range a.worktrees {
		cursor := "  "
		if i == a.selectedWorktree {
			cursor = "→ "
		}

		stats := fmt.Sprintf("%d files  +%d -%d",
			wt.FilesChanged, wt.Insertions, wt.Deletions)

		// Test status indicator
		var statusIcon string
		switch a.testStatus[wt.Branch] {
		case 1:
			statusIcon = styleRunning.Render(" ⟳")
		case 2:
			statusIcon = stylePass.Render(" ✓")
		case 3:
			statusIcon = styleFail.Render(" ✗")
		default:
			statusIcon = stylePending.Render(" —")
		}

		line := fmt.Sprintf("%s%-40s %s%s", cursor, wt.Branch, stats, statusIcon)
		if i == a.selectedWorktree {
			line = styleSelected.Width(a.width).Render(line)
		} else {
			line = styleNormal.Render(line)
		}
		b.WriteString(line + "\n")
	}

	b.WriteString("\n")

	if a.landing {
		b.WriteString(styleRunning.Render(fmt.Sprintf("  Landing %s... %s\n", a.landBranch, a.landStep)))
		b.WriteString("\n")
	}

	if a.confirmDelete && len(a.worktrees) > 0 {
		wt := a.worktrees[a.selectedWorktree]
		b.WriteString(styleFail.Render(fmt.Sprintf("  Delete %s? (d to confirm, esc to cancel)\n", wt.Branch)))
		b.WriteString("\n")
	}

	b.WriteString(styleHelp.Render("  enter: review  l: land  d: delete  t: run tests  q: quit"))

	return b.String()
}
