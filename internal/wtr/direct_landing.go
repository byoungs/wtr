package wtr

import (
	"fmt"
	"strings"
	"time"

	"github.com/byoungs/wtr/internal/git"
	"github.com/byoungs/wtr/internal/land"
	"github.com/byoungs/wtr/internal/runner"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (a App) updateDirectLanding(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Note: branchInfoMsg is handled in the main Update method (app.go),
	// including synthetic worktree population for direct mode.
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Enter), key.Matches(msg, keys.Right):
			// Drill into file list showing diff vs origin
			if a.branchInfo.AheadOrigin > 0 || !a.branchInfo.HasUpstream {
				a.screen = screenFileList
				return a, a.loadDirectDiff()
			}
		case key.Matches(msg, keys.Test):
			if err := runner.Start(a.repoDir, a.repoDir, a.branchInfo.Name); err != nil {
				a.err = err
				return a, nil
			}
			a.testStatus[a.branchInfo.Name] = 1
			a.flashMsg = "Running make validate..."
			return a, tea.Batch(flashAfter(2*time.Second), tickTestStatus())
		case key.Matches(msg, keys.ViewOutput):
			output := runner.ReadLog(a.repoDir, a.branchInfo.Name)
			if output != "" || a.testStatus[a.branchInfo.Name] == 1 {
				a.prevScreen = a.screen
				a.screen = screenTestOutput
				testOutputFollow = true
				if a.testStatus[a.branchInfo.Name] == 1 {
					return a, tickOutput()
				}
				return a, nil
			}
		case key.Matches(msg, keys.Land):
			if a.branchInfo.AheadOrigin > 0 && !a.landing {
				a.landing = true
				a.landBranch = a.branchInfo.Name
				a.landStep = ""
				logFile := runner.LogPath(a.repoDir, a.branchInfo.Name)
				return a, tea.Batch(func() tea.Msg {
					_, err := land.Run(a.repoDir, land.DirectSteps(), logFile, func(s land.Step) {})
					return landDoneMsg{err: err}
				}, tickLandStatus())
			}
		case key.Matches(msg, keys.GitStatus):
			a.statusFiles = loadGitStatus(a.repoDir)
			if len(a.statusFiles) > 0 {
				a.statusCursor = 0
				a.confirmRevert = false
				a.prevScreen = screenDirectLanding
				a.screen = screenGitStatus
				return a, nil
			}
		case key.Matches(msg, keys.Refresh):
			a.flashMsg = "Refreshing..."
			return a, tea.Batch(a.loadBranchInfo(), flashAfter(2*time.Second))
		}
	}
	return a, nil
}

func (a App) loadDirectDiff() tea.Cmd {
	return func() tea.Msg {
		baseRef := git.UpstreamRef(a.repoDir)
		if baseRef == "" {
			// No upstream — show all commits (diff against empty tree)
			baseRef = "4b825dc642cb6eb9a060e54bf899d69f82cf7871" // git empty tree hash
		}
		files, err := git.GetDiff(a.repoDir, baseRef)
		if err != nil {
			return errMsg{err}
		}
		return diffLoadedMsg{files}
	}
}

func (a App) viewDirectLanding() string {
	var b strings.Builder

	title := styleTitle.Width(a.width).Render("Development on main")
	b.WriteString(title + "\n\n")

	if a.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(colorRed).Width(a.width - 4)
		b.WriteString(errStyle.Render(fmt.Sprintf("  Error: %v", a.err)) + "\n")
		b.WriteString(styleHelp.Render("  (o:output  any key: dismiss)") + "\n\n")
	}

	// Branch summary line — matches worktree list style:
	//   main                                     12 files  +473 -33 ↑1 ✓ △1
	info := a.branchInfo

	stats := fmt.Sprintf("%d files  +%d -%d", info.FilesChanged, info.Insertions, info.Deletions)

	// Ahead/behind
	var branchState string
	if !info.HasUpstream {
		branchState = styleRunning.Render(" (no upstream)")
	} else if info.AheadOrigin == 0 {
		branchState = stylePass.Render(" (up to date)")
	} else {
		ahead := fmt.Sprintf(" ↑%d", info.AheadOrigin)
		if info.BehindOrigin > 0 {
			branchState = styleRunning.Render(ahead) + styleRunning.Render(fmt.Sprintf(" ↓%d", info.BehindOrigin))
		} else {
			branchState = stylePass.Render(ahead)
		}
	}

	// Test status icon
	var testIcon string
	switch a.testStatus[info.Name] {
	case 1:
		testIcon = styleRunning.Render(" ⟳")
	case 2:
		testIcon = stylePass.Render(" ✓")
	case 3:
		testIcon = styleFail.Render(" ✗")
	}

	// Uncommitted indicator
	var dirtyIcon string
	if info.Uncommitted > 0 {
		dirtyIcon = styleRunning.Render(fmt.Sprintf(" △%d", info.Uncommitted))
	}

	line := fmt.Sprintf("  %-40s %s%s%s%s", info.Name, stats, branchState, testIcon, dirtyIcon)
	b.WriteString(styleSelected.Width(a.width).Render(line) + "\n")

	if info.HasUpstream && info.AheadOrigin == 0 && len(info.Commits) == 0 {
		b.WriteString("\n")
		b.WriteString("  " + stylePass.Render("Nothing to push.") + "\n")
	}

	b.WriteString("\n")

	// Landing status
	if a.landing {
		stepInfo := ""
		if a.landStep != "" {
			stepInfo = " " + a.landStep
		}
		b.WriteString(styleRunning.Render(fmt.Sprintf("  Pushing %s...%s", a.landBranch, stepInfo)) +
			" " + styleHelp.Render("(o:output)") + "\n")
		b.WriteString("\n")
	}

	if a.flashMsg != "" {
		b.WriteString(styleRunning.Render("  "+a.flashMsg) + "\n")
		b.WriteString("\n")
	}

	// Unpushed commits — fills remaining space
	if len(info.Commits) > 0 {
		linesUsed := strings.Count(b.String(), "\n")
		remaining := a.height - linesUsed - 1 // reserve 1 for help bar
		if remaining > 0 {
			hashStyle := lipgloss.NewStyle().Foreground(colorSubtle)
			msgStyle := lipgloss.NewStyle().Foreground(colorSubtle)
			linesWritten := 0
			for _, c := range info.Commits {
				if linesWritten >= remaining {
					break
				}
				b.WriteString("    " + hashStyle.Render(c.Hash) + "  " + msgStyle.Render(c.Subject) + "\n")
				linesWritten++
				if c.Body != "" {
					for _, bodyLine := range strings.Split(c.Body, "\n") {
						if linesWritten >= remaining {
							break
						}
						b.WriteString("             " + msgStyle.Render(bodyLine) + "\n")
						linesWritten++
					}
				}
			}
		}
	}

	padToBottom(&b, a.height, strings.Count(b.String(), "\n"))
	b.WriteString(styleHelp.Render("  q:quit  h:help  →review  g:status  t:test  o:output  l:push  u:refresh"))

	return b.String()
}
