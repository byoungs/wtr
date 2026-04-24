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

var devOutputScrollY int
var devOutputFollow bool = true

type devTickMsg struct{}

func tickDev() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
		return devTickMsg{}
	})
}

func (a App) updateDevOutput(msg tea.Msg) (tea.Model, tea.Cmd) {
	wt := a.worktrees[a.selectedWorktree]

	switch msg := msg.(type) {
	case devTickMsg:
		if runner.DevIsRunning(a.repoDir, wt.Branch) {
			return a, tickDev()
		}
		return a, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Up):
			devOutputFollow = false
			if devOutputScrollY > 0 {
				devOutputScrollY--
			}
		case key.Matches(msg, keys.Down):
			devOutputFollow = false
			devOutputScrollY++
		case key.Matches(msg, keys.PageDown):
			devOutputFollow = false
			viewHeight := a.height - 5
			if viewHeight < 1 {
				viewHeight = 1
			}
			devOutputScrollY += viewHeight
		case key.Matches(msg, keys.PageUp):
			devOutputFollow = false
			viewHeight := a.height - 5
			if viewHeight < 1 {
				viewHeight = 1
			}
			devOutputScrollY -= viewHeight
			if devOutputScrollY < 0 {
				devOutputScrollY = 0
			}
		case key.Matches(msg, keys.Back):
			runner.KillDev(a.repoDir, wt.Branch)
			a.screen = a.prevScreen
			devOutputScrollY = 0
			devOutputFollow = true
			a.flashMsg = fmt.Sprintf("Stopped make dev in %s", wt.Branch)
			return a, flashAfter(2 * time.Second)
		}
	}
	return a, nil
}

func (a App) viewDevOutput() string {
	var b strings.Builder

	wt := a.worktrees[a.selectedWorktree]

	var statusStr string
	if runner.DevIsRunning(a.repoDir, wt.Branch) {
		statusStr = styleRunning.Render(" RUNNING")
	} else {
		statusStr = styleFail.Render(" STOPPED")
	}

	title := styleTitle.Width(a.width).Render(
		fmt.Sprintf("make dev — %s%s", wt.Branch, statusStr))
	b.WriteString(title + "\n")

	output := runner.ReadDevLog(a.repoDir, wt.Branch)
	if output == "" {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(colorSubtle).Render("  Waiting for output...\n"))
		b.WriteString("\n")
		b.WriteString(styleHelp.Render("  esc: kill & back"))
		return b.String()
	}

	allLines := strings.Split(output, "\n")

	viewHeight := a.height - 5
	if viewHeight < 1 {
		viewHeight = 1
	}

	if devOutputFollow {
		devOutputScrollY = len(allLines) - viewHeight
		if devOutputScrollY < 0 {
			devOutputScrollY = 0
		}
	}

	maxScroll := len(allLines) - viewHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if devOutputScrollY > maxScroll {
		devOutputScrollY = maxScroll
	}
	end := devOutputScrollY + viewHeight
	if end > len(allLines) {
		end = len(allLines)
	}

	outputStyle := lipgloss.NewStyle().Foreground(colorText)
	failLine := lipgloss.NewStyle().Foreground(colorRed)
	passLine := lipgloss.NewStyle().Foreground(colorGreen)

	for _, line := range allLines[devOutputScrollY:end] {
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
	helpText := fmt.Sprintf("  space/b: page dn/up  esc: kill & back  (%d/%d lines)",
		devOutputScrollY+1, len(allLines))
	if runner.DevIsRunning(a.repoDir, wt.Branch) {
		helpText += "  (live)"
	}
	b.WriteString(styleHelp.Render(helpText))

	return b.String()
}
