package wtr

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/byoungs/wtr/internal/git"
	"github.com/byoungs/wtr/internal/land"
	"github.com/byoungs/wtr/internal/runner"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var wtListScrollY int

// checkFresh verifies the worktree hasn't changed since we last loaded it.
func (a *App) checkFresh(wt git.Worktree) bool {
	currentHash := git.CurrentHash(wt.Path)
	if currentHash != "" && currentHash != wt.CommitHash {
		a.flashMsg = fmt.Sprintf("⚠ %s changed (was %.7s, now %.7s) — press u to refresh",
			wt.Branch, wt.CommitHash, currentHash)
		return false
	}
	return true
}

func (a App) updateWorktreeList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case worktreesLoadedMsg:
		a.worktrees = msg.worktrees
		a.mainUncommitted = msg.mainUncommitted
		// Invalidate test status if code changed since last test
		for _, wt := range a.worktrees {
			if testedHash, ok := a.testedAt[wt.Branch]; ok {
				if wt.CommitHash != testedHash {
					delete(a.testStatus, wt.Branch)
					delete(a.testedAt, wt.Branch)
					runner.Clean(a.repoDir, wt.Branch)
				}
			}
		}
		// Invalidate reviews only for files that actually changed since last review
		for _, wt := range a.worktrees {
			if reviewedHash, ok := a.reviewedAt[wt.Branch]; ok {
				if wt.CommitHash != reviewedHash {
					changedFiles := git.ChangedFilesBetween(wt.Path, reviewedHash, wt.CommitHash)
					if len(changedFiles) == 0 {
						// Can't determine diff (e.g. hash no longer exists) — clear all
						prefix := wt.Branch + ":"
						for k := range a.reviewed {
							if strings.HasPrefix(k, prefix) {
								delete(a.reviewed, k)
							}
						}
					} else {
						for _, f := range changedFiles {
							delete(a.reviewed, wt.Branch+":"+f)
						}
					}
					// Update to current hash so next refresh compares from here
					a.reviewedAt[wt.Branch] = wt.CommitHash
				}
			}
		}
		// Sync with any background processes from disk
		cmd := a.syncTestStatus()

		// Auto-detect mode: no worktrees → direct mode
		if len(a.worktrees) == 0 {
			a.mode = "direct"
			a.screen = screenDirectLanding
			return a, a.loadBranchInfo()
		}
		a.mode = "worktree"
		return a, cmd

	case tea.KeyMsg:
		// Force delete typing mode
		if a.deleteState == 2 {
			if key.Matches(msg, keys.Back) {
				a.deleteState = 0
				a.deleteError = ""
				a.forceInput = ""
				return a, nil
			}
			ch := msg.String()
			if len(ch) == 1 {
				a.forceInput += ch
				if a.forceInput == "force" {
					wt := a.worktrees[a.selectedWorktree]
					a.deleteState = 0
					a.deleteError = ""
					a.forceInput = ""
					return a, func() tea.Msg {
						runner.Clean(a.repoDir, wt.Branch)
						out, err := exec.Command("git", "-C", a.repoDir, "worktree", "remove", "--force", wt.Path).CombinedOutput()
						if err != nil {
							return errMsg{fmt.Errorf("force remove failed: %s", strings.TrimSpace(string(out)))}
						}
						exec.Command("git", "-C", a.repoDir, "branch", "-D", wt.Branch).Run()
						return worktreeDeletedMsg{}
					}
				}
				// If what they've typed so far isn't a prefix of "force", reset
				if !strings.HasPrefix("force", a.forceInput) {
					a.forceInput = ""
				}
			}
			return a, nil
		}

		switch {
		case key.Matches(msg, keys.Up):
			if a.selectedWorktree > 0 {
				a.selectedWorktree--
			}
		case key.Matches(msg, keys.Down):
			maxIdx := len(a.worktrees) - 1
			if a.mainUncommitted > 0 {
				maxIdx = len(a.worktrees) // main row is one past the last worktree
			}
			if a.selectedWorktree < maxIdx {
				a.selectedWorktree++
			}
		case key.Matches(msg, keys.Refresh):
			a.flashMsg = "Refreshing..."
			return a, tea.Batch(a.loadWorktrees(), flashAfter(2*time.Second))
		case key.Matches(msg, keys.Enter), key.Matches(msg, keys.Right):
			if a.onMainRow() {
				a.statusFiles = loadGitStatus(a.repoDir)
				if len(a.statusFiles) > 0 {
					a.worktrees = append(a.worktrees, git.Worktree{
					Path:        a.repoDir,
					Branch:      "main",
					CommitHash:  git.CurrentHash(a.repoDir),
					Uncommitted: a.mainUncommitted,
				})
				a.selectedWorktree = len(a.worktrees) - 1
					a.statusCursor = 0
					a.confirmRevert = false
					a.prevScreen = screenWorktreeList
					a.screen = screenGitStatus
				}
				return a, nil
			}
			if len(a.worktrees) > 0 {
				a.screen = screenFileList
				return a, a.loadDiff()
			}
		case key.Matches(msg, keys.GitStatus):
			a.statusFiles = loadGitStatus(a.repoDir)
			if len(a.statusFiles) > 0 {
				a.worktrees = append(a.worktrees, git.Worktree{
					Path:        a.repoDir,
					Branch:      "main",
					CommitHash:  git.CurrentHash(a.repoDir),
					Uncommitted: a.mainUncommitted,
				})
				a.selectedWorktree = len(a.worktrees) - 1
				a.statusCursor = 0
				a.confirmRevert = false
				a.prevScreen = screenWorktreeList
				a.screen = screenGitStatus
			}
			return a, nil
		case key.Matches(msg, keys.Open):
			if len(a.worktrees) > 0 {
				wt := a.worktrees[a.selectedWorktree]
				exec.Command("code", wt.Path).Start()
			}
		case key.Matches(msg, keys.Test):
			if len(a.worktrees) > 0 {
				wt := a.worktrees[a.selectedWorktree]
				if !a.checkFresh(wt) {
					return a, flashAfter(3 * time.Second)
				}
				// Start background process
				if err := runner.Start(a.repoDir, wt.Path, wt.Branch); err != nil {
					a.err = err
					return a, nil
				}
				a.testStatus[wt.Branch] = 1
				a.flashMsg = fmt.Sprintf("Running make validate in %s...", wt.Branch)
				return a, tea.Batch(flashAfter(2*time.Second), tickTestStatus())
			}
		case key.Matches(msg, keys.ViewOutput):
			if len(a.worktrees) > 0 {
				wt := a.worktrees[a.selectedWorktree]
				// Allow viewing if there's a log file (running or completed)
				output := runner.ReadLog(a.repoDir, wt.Branch)
				if output != "" || a.testStatus[wt.Branch] == 1 {
					a.prevScreen = a.screen
					a.screen = screenTestOutput
					testOutputFollow = true
					if a.testStatus[wt.Branch] == 1 {
						return a, tickOutput()
					}
					return a, nil
				}
			}
		case key.Matches(msg, keys.Land):
			if len(a.worktrees) > 0 && !a.landing {
				wt := a.worktrees[a.selectedWorktree]
				if !a.checkFresh(wt) {
					return a, flashAfter(3 * time.Second)
				}
				a.landing = true
				a.landBranch = wt.Branch
				a.landStep = "(o: view output)"
				logFile := runner.LogPath(a.repoDir, wt.Branch)
				wtPath := wt.Path
				return a, func() tea.Msg {
					_, err := land.Run(a.repoDir, land.Steps(wt.Branch), logFile, func(s land.Step) {})
					if err == nil {
						// Fast-forward worktree branch to match main
						exec.Command("git", "-C", wtPath, "rebase", "main").Run()
					}
					return landDoneMsg{err: err}
				}
			}
		case key.Matches(msg, keys.Squash):
			if len(a.worktrees) > 0 {
				wt := a.worktrees[a.selectedWorktree]
				if !a.checkFresh(wt) {
					return a, flashAfter(3 * time.Second)
				}
				a.flashMsg = fmt.Sprintf("Squashing %s onto main...", wt.Branch)
				return a, func() tea.Msg {
					_, err := git.SquashOntoMain(wt.Path)
					return squashDoneMsg{err: err}
				}
			}
		case key.Matches(msg, keys.Rebase):
			if len(a.worktrees) > 0 {
				wt := a.worktrees[a.selectedWorktree]
				if !a.checkFresh(wt) {
					return a, flashAfter(3 * time.Second)
				}
				a.flashMsg = fmt.Sprintf("Rebasing %s on main...", wt.Branch)
				return a, func() tea.Msg {
					_, err := git.RebaseOnMain(wt.Path)
					return rebaseDoneMsg{err: err}
				}
			}
		case key.Matches(msg, keys.Delete):
			if len(a.worktrees) > 0 && a.deleteState == 0 {
				wt := a.worktrees[a.selectedWorktree]
				a.deleteState = 1
				return a, func() tea.Msg {
					runner.Clean(a.repoDir, wt.Branch)
					out, err := exec.Command("git", "-C", a.repoDir, "worktree", "remove", wt.Path).CombinedOutput()
					if err != nil {
						return deleteFailedMsg{output: strings.TrimSpace(string(out))}
					}
					exec.Command("git", "-C", a.repoDir, "branch", "-D", wt.Branch).Run()
					return worktreeDeletedMsg{}
				}
			}
		case key.Matches(msg, keys.Back):
			if a.deleteState > 0 {
				a.deleteState = 0
				a.deleteError = ""
				a.forceInput = ""
			}
		}
	}
	return a, nil
}

// onMainRow returns true if the cursor is on the main branch row.
func (a App) onMainRow() bool {
	return a.mainUncommitted > 0 && a.selectedWorktree == len(a.worktrees)
}

// currentWorktreePath returns the path for the currently selected worktree,
// or repoDir if on the main row.
func (a App) currentWorktreePath() string {
	if a.selectedWorktree >= len(a.worktrees) {
		return a.repoDir
	}
	return a.worktrees[a.selectedWorktree].Path
}


func (a App) loadDiff() tea.Cmd {
	wt := a.worktrees[a.selectedWorktree]
	return func() tea.Msg {
		files, err := git.GetDiff(wt.Path, "main")
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
		errStyle := lipgloss.NewStyle().Foreground(colorRed).Width(a.width - 4)
		b.WriteString(errStyle.Render(fmt.Sprintf("  Error: %v", a.err)) + "\n")
		b.WriteString(styleHelp.Render("  (o: view full output  any key: dismiss)") + "\n\n")
	}

	if len(a.worktrees) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(colorSubtle).Render(
			"  No worktrees found.\n"))
		b.WriteString("\n")
		b.WriteString(styleHelp.Render("  q/esc: quit"))
		return b.String()
	}

	var lines []string
	for i, wt := range a.worktrees {
		cursor := "  "
		if i == a.selectedWorktree {
			cursor = "→ "
		}

		stats := fmt.Sprintf("%d files  +%d -%d",
			wt.FilesChanged, wt.Insertions, wt.Deletions)

		// Branch state
		var branchState string
		ahead := fmt.Sprintf(" ↑%d", wt.CommitsAhead)
		if wt.CommitsAhead == 0 && wt.CommitsBehind == 0 {
			branchState = stylePending.Render(" (no commits)")
		} else if wt.CommitsAhead == 0 && wt.CommitsBehind > 0 {
			branchState = styleRunning.Render(fmt.Sprintf(" ↓%d", wt.CommitsBehind)) + stylePending.Render(" (ff)")
		} else if wt.CommitsBehind > 0 {
			branchState = styleRunning.Render(ahead) + styleRunning.Render(fmt.Sprintf(" ↓%d", wt.CommitsBehind))
		} else {
			branchState = stylePass.Render(ahead)
		}

		// Test status
		var testIcon string
		switch a.testStatus[wt.Branch] {
		case 1:
			testIcon = styleRunning.Render(" ⟳")
		case 2:
			testIcon = stylePass.Render(" ✓")
		case 3:
			testIcon = styleFail.Render(" ✗")
		default:
			testIcon = ""
		}

		// Uncommitted changes indicator
		var dirtyIcon string
		if wt.Uncommitted > 0 {
			dirtyIcon = styleRunning.Render(fmt.Sprintf(" △%d", wt.Uncommitted))
		}

		line := fmt.Sprintf("%s%-40s %s%s%s%s", cursor, wt.Branch, stats, branchState, testIcon, dirtyIcon)
		if i == a.selectedWorktree {
			line = styleSelected.Width(a.width).Render(line)
		} else {
			line = styleNormal.Render(line)
		}
		lines = append(lines, line)
	}

	overhead := 7 // title(2) + blank(1) + main footer(1) + padToBottom + help(1) + extra
	if a.err != nil {
		overhead += 3 // error + hint + blank
	}
	if a.landing {
		overhead += 2
	}
	if a.flashMsg != "" {
		overhead += 2
	}
	if a.deleteState > 0 {
		overhead += 2
		if a.deleteState == 2 {
			overhead += 2 // extra line for error message
		}
	}
	viewHeight := a.height - overhead
	if viewHeight < 1 {
		viewHeight = 1
	}

	if a.selectedWorktree < wtListScrollY {
		wtListScrollY = a.selectedWorktree
	}
	if a.selectedWorktree >= wtListScrollY+viewHeight {
		wtListScrollY = a.selectedWorktree - viewHeight + 1
	}
	if wtListScrollY > len(lines)-viewHeight {
		wtListScrollY = max(0, len(lines)-viewHeight)
	}

	end := wtListScrollY + viewHeight
	if end > len(lines) {
		end = len(lines)
	}
	for _, line := range lines[wtListScrollY:end] {
		b.WriteString(line + "\n")
	}

	// Commit preview for selected worktree
	if a.selectedWorktree < len(a.worktrees) {
		wt := a.worktrees[a.selectedWorktree]
		if len(wt.Commits) > 0 {
			b.WriteString("\n")
			hashStyle := lipgloss.NewStyle().Foreground(colorSubtle)
			for _, c := range wt.Commits {
				b.WriteString("    " + hashStyle.Render(c.Hash) + "  " + c.Subject + "\n")
			}
		}
	}

	b.WriteString("\n")

	if a.landing {
		b.WriteString(styleRunning.Render(fmt.Sprintf("  Landing %s... %s\n", a.landBranch, a.landStep)))
		b.WriteString("\n")
	}

	if a.flashMsg != "" {
		b.WriteString(styleRunning.Render("  "+a.flashMsg) + "\n")
		b.WriteString("\n")
	}

	if a.deleteState == 1 && len(a.worktrees) > 0 {
		b.WriteString(styleRunning.Render("  Deleting...") + "\n")
		b.WriteString("\n")
	}
	if a.deleteState == 2 && len(a.worktrees) > 0 {
		errStyle := lipgloss.NewStyle().Foreground(colorRed).Width(a.width - 4)
		b.WriteString(errStyle.Render(fmt.Sprintf("  Delete failed: %s", a.deleteError)) + "\n")
		typed := a.forceInput
		remaining := "force"[len(typed):]
		b.WriteString(styleFail.Render("  Type 'force' to force delete: ") +
			stylePass.Render(typed) + stylePending.Render(remaining) +
			styleFail.Render("  (esc to cancel)") + "\n")
		b.WriteString("\n")
	}

	// Main branch footer
	mainCursor := "  "
	if a.onMainRow() {
		mainCursor = "→ "
	}
	var mainStatus string
	if a.mainUncommitted > 0 {
		mainStatus = styleRunning.Render(fmt.Sprintf(" △%d uncommitted", a.mainUncommitted))
	} else {
		mainStatus = stylePass.Render(" clean")
	}
	mainLine := fmt.Sprintf("%s%-40s%s", mainCursor, "main", mainStatus)
	if a.onMainRow() {
		mainLine = styleSelected.Width(a.width).Render(mainLine)
	} else {
		mainLine = styleNormal.Render(mainLine)
	}
	b.WriteString(mainLine + "\n")

	padToBottom(&b, a.height, strings.Count(b.String(), "\n"))
	b.WriteString(styleHelp.Render("  q:quit  h:help  →review  e:edit  t:test  o:output  r:rebase  l:land  del:delete  g:status  u:refresh"))

	return b.String()
}
