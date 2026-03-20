package tui

import (
	"github.com/byoungs/wtr/internal/git"
	tea "github.com/charmbracelet/bubbletea"
)

type screen int

const (
	screenWorktreeList screen = iota
	screenFileList
	screenDiffView
)

type App struct {
	screen           screen
	repoDir          string
	width            int
	height           int

	// State
	worktrees        []git.Worktree
	selectedWorktree int
	files            []git.FileDiff
	selectedFile     int
	reviewed         map[string]bool // key: "branch:filename"
	sideBySide       bool
	err              error

	// Test status per worktree
	testStatus map[string]int    // 0=none, 1=running, 2=passed, 3=failed
	testOutput map[string]string // branch -> combined output

	// Land state
	landing     bool
	landBranch  string
	landStep    string
	landResults []string

	// Delete confirmation
	confirmDelete bool
}

func NewApp(repoDir string) App {
	return App{
		repoDir:    repoDir,
		reviewed:   make(map[string]bool),
		sideBySide: true,
		testStatus: make(map[string]int),
		testOutput: make(map[string]string),
	}
}

func (a App) Init() tea.Cmd {
	return a.loadWorktrees()
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil
	case errMsg:
		a.err = msg.err
		return a, nil
	case testDoneMsg:
		if msg.passed {
			a.testStatus[msg.branch] = 2
		} else {
			a.testStatus[msg.branch] = 3
		}
		a.testOutput[msg.branch] = msg.output
		return a, nil
	case landStepMsg:
		a.landStep = msg.step
		return a, nil
	case landDoneMsg:
		a.landing = false
		if msg.err != nil {
			a.err = msg.err
		}
		return a, a.loadWorktrees()
	case worktreeDeletedMsg:
		a.confirmDelete = false
		return a, a.loadWorktrees()
	case tea.KeyMsg:
		// Clear errors on any keypress
		if a.err != nil {
			a.err = nil
		}
		if msg.String() == "q" && a.screen == screenWorktreeList {
			return a, tea.Quit
		}
	}

	switch a.screen {
	case screenWorktreeList:
		return a.updateWorktreeList(msg)
	case screenFileList:
		return a.updateFileList(msg)
	case screenDiffView:
		return a.updateDiffView(msg)
	}
	return a, nil
}

func (a App) View() string {
	if a.width == 0 {
		return "Loading..."
	}
	switch a.screen {
	case screenWorktreeList:
		return a.viewWorktreeList()
	case screenFileList:
		return a.viewFileList()
	case screenDiffView:
		return a.viewDiffView()
	}
	return ""
}

// Messages
type worktreesLoadedMsg struct{ worktrees []git.Worktree }
type diffLoadedMsg struct{ files []git.FileDiff }
type errMsg struct{ err error }

type testDoneMsg struct {
	branch string
	passed bool
	output string
}

type landStepMsg struct{ step string }
type landDoneMsg struct{ err error }

type worktreeDeletedMsg struct{}

func (a App) loadWorktrees() tea.Cmd {
	return func() tea.Msg {
		wts, err := git.ListWorktrees(a.repoDir)
		if err != nil {
			return errMsg{err}
		}
		return worktreesLoadedMsg{wts}
	}
}
