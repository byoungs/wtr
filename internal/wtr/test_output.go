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
		switch {
		case isFailLine(line):
			b.WriteString(failLine.Render(line) + "\n")
		case isPassLine(line):
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

// isFailLine returns true if the line indicates an actual failure or error,
// not just a test name that happens to contain "error" or "fail".
func isFailLine(line string) bool {
	trimmed := strings.TrimSpace(line)

	// Go test failure markers
	if strings.HasPrefix(trimmed, "--- FAIL:") {
		return true
	}
	if strings.HasPrefix(trimmed, "FAIL\t") || trimmed == "FAIL" {
		return true
	}

	// Panic
	if strings.HasPrefix(trimmed, "panic:") {
		return true
	}

	// Go compiler/test errors: file.go:line:col: message
	// Also matches t.Errorf output (file_test.go:42: msg) which is intentional
	if isCompilerError(trimmed) {
		return true
	}

	// Testify assertion failures
	if strings.HasPrefix(trimmed, "Error Trace:") ||
		strings.HasPrefix(trimmed, "Error:") ||
		strings.HasPrefix(trimmed, "Expected:") ||
		strings.HasPrefix(trimmed, "Actual:") {
		return true
	}

	// Race detector
	if strings.HasPrefix(trimmed, "WARNING: DATA RACE") {
		return true
	}

	// Make errors
	if strings.HasPrefix(trimmed, "make:") && strings.Contains(trimmed, "Error") {
		return true
	}
	if strings.HasPrefix(trimmed, "make[") && strings.Contains(trimmed, "Error") {
		return true
	}

	// exit status
	if strings.Contains(trimmed, "exit status") {
		return true
	}

	return false
}

// isCompilerError detects lines like "file.go:42:5: undefined: foo"
func isCompilerError(line string) bool {
	// Must contain .go: followed by a digit (line number)
	idx := strings.Index(line, ".go:")
	if idx < 0 || idx+4 >= len(line) {
		return false
	}
	after := line[idx+4:]
	return len(after) > 0 && after[0] >= '0' && after[0] <= '9'
}

// isPassLine returns true if the line indicates a passing test or package.
func isPassLine(line string) bool {
	trimmed := strings.TrimSpace(line)

	// Go test pass markers
	if strings.HasPrefix(trimmed, "--- PASS:") {
		return true
	}
	if strings.HasPrefix(trimmed, "ok  \t") || strings.HasPrefix(trimmed, "ok \t") {
		return true
	}
	if trimmed == "PASS" {
		return true
	}

	return false
}
