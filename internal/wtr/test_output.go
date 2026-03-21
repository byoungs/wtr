package wtr

import (
	"fmt"
	"strings"
	"time"

	"github.com/byoungs/wtr/internal/runner"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var testOutputScrollY int
var testOutputFollow bool = true // auto-scroll to bottom while running

type outputTickMsg struct{}

func tickOutput() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
		return outputTickMsg{}
	})
}

func (a App) updateTestOutput(msg tea.Msg) (tea.Model, tea.Cmd) {
	wt := a.worktrees[a.selectedWorktree]

	switch msg := msg.(type) {
	case outputTickMsg:
		// Keep ticking if still running
		if a.testStatus[wt.Branch] == 1 {
			// Check if finished
			if !runner.IsRunning(a.repoDir, wt.Branch) {
				diskStatus := runner.ReadStatus(a.repoDir, wt.Branch)
				a.testStatus[wt.Branch] = runner.StatusToInt(diskStatus)
				a.testedAt[wt.Branch] = wt.CommitHash
			}
			return a, tickOutput()
		}
		return a, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Up):
			testOutputFollow = false
			if testOutputScrollY > 0 {
				testOutputScrollY--
			}
		case key.Matches(msg, keys.Down):
			testOutputFollow = false
			testOutputScrollY++
		case key.Matches(msg, keys.PageDown):
			testOutputFollow = false
			viewHeight := a.height - 5
			if viewHeight < 1 {
				viewHeight = 1
			}
			testOutputScrollY += viewHeight
		case key.Matches(msg, keys.PageUp):
			testOutputFollow = false
			viewHeight := a.height - 5
			if viewHeight < 1 {
				viewHeight = 1
			}
			testOutputScrollY -= viewHeight
			if testOutputScrollY < 0 {
				testOutputScrollY = 0
			}
		case key.Matches(msg, keys.Back):
			a.screen = a.prevScreen
			testOutputScrollY = 0
			testOutputFollow = true
		}
	}
	return a, nil
}

func (a App) viewTestOutput() string {
	var b strings.Builder

	wt := a.worktrees[a.selectedWorktree]

	var statusStr string
	switch a.testStatus[wt.Branch] {
	case 1:
		statusStr = styleRunning.Render(" RUNNING")
	case 2:
		statusStr = stylePass.Render(" PASSED")
	case 3:
		statusStr = styleFail.Render(" FAILED")
	default:
		statusStr = ""
	}

	title := styleTitle.Width(a.width).Render(
		fmt.Sprintf("Validate Output — %s%s", wt.Branch, statusStr))
	b.WriteString(title + "\n")

	// Read output from disk (live)
	output := runner.ReadLog(a.repoDir, wt.Branch)
	if output == "" {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(colorSubtle).Render("  Waiting for output...\n"))
		b.WriteString("\n")
		b.WriteString(styleHelp.Render("  esc: back"))
		return b.String()
	}

	allLines := strings.Split(output, "\n")

	viewHeight := a.height - 5
	if viewHeight < 1 {
		viewHeight = 1
	}

	// Auto-follow: scroll to bottom while running
	if testOutputFollow {
		testOutputScrollY = len(allLines) - viewHeight
		if testOutputScrollY < 0 {
			testOutputScrollY = 0
		}
	}

	maxScroll := len(allLines) - viewHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if testOutputScrollY > maxScroll {
		testOutputScrollY = maxScroll
	}
	end := testOutputScrollY + viewHeight
	if end > len(allLines) {
		end = len(allLines)
	}

	outputStyle := lipgloss.NewStyle().Foreground(colorText)
	failLine := lipgloss.NewStyle().Foreground(colorRed)
	passLine := lipgloss.NewStyle().Foreground(colorGreen)

	for _, line := range allLines[testOutputScrollY:end] {
		lower := strings.ToLower(line)
		switch {
		case strings.Contains(lower, "fail") || strings.Contains(lower, "error"):
			b.WriteString(failLine.Render(line) + "\n")
		case strings.Contains(lower, "pass") || strings.Contains(lower, "ok "):
			b.WriteString(passLine.Render(line) + "\n")
		default:
			b.WriteString(outputStyle.Render(line) + "\n")
		}
	}

	b.WriteString("\n")
	helpText := fmt.Sprintf("  space/b: page dn/up  ←/esc: back  (%d/%d lines)",
		testOutputScrollY+1, len(allLines))
	if a.testStatus[wt.Branch] == 1 {
		helpText += "  (live)"
	}
	b.WriteString(styleHelp.Render(helpText))

	return b.String()
}
