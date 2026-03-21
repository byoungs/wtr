package wtr

import (
	"time"

	"github.com/byoungs/wtr/internal/git"
	"github.com/byoungs/wtr/internal/runner"
	"github.com/byoungs/wtr/internal/state"
	tea "github.com/charmbracelet/bubbletea"
)

type screen int

const (
	screenWorktreeList screen = iota
	screenFileList
	screenDiffView
	screenAllDiffs
	screenHelp
	screenTestOutput
	screenGitStatus
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
	testedAt   map[string]string // branch -> commit hash when tested

	// Land state
	landing     bool
	landBranch  string
	landStep    string
	landResults []string

	// Delete: 0=idle, 1=trying, 2=force prompt (typing "force")
	deleteState int
	deleteError string
	forceInput  string // accumulates typed chars for "force"

	// Flash message (auto-clears after 2s)
	flashMsg string

	// Help/output screen return
	prevScreen screen

	// Git status screen
	statusFiles    []statusEntry
	statusCursor   int
	confirmRevert  bool

	// Main branch info
	mainUncommitted int
}

func NewApp(repoDir string) App {
	s := state.Load(repoDir)
	reviewed := s.Reviewed
	if reviewed == nil {
		reviewed = make(map[string]bool)
	}
	return App{
		repoDir:    repoDir,
		reviewed:   reviewed,
		sideBySide: false,
		testStatus: s.TestStatus,
		testedAt:   s.TestedAt,
	}
}

func (a App) Init() tea.Cmd {
	return a.loadWorktrees()
}

func (a App) saveState() {
	state.Save(a.repoDir, state.State{
		TestStatus: a.testStatus,
		TestedAt:   a.testedAt,
		Reviewed:   a.reviewed,
	})
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
	case outputTickMsg:
		if a.screen == screenTestOutput {
			return a.updateTestOutput(msg)
		}
		return a, nil
	case testDoneMsg:
		// Background process finished — read status from disk
		status := runner.ReadStatus(a.repoDir, msg.branch)
		a.testStatus[msg.branch] = runner.StatusToInt(status)
		a.testedAt[msg.branch] = msg.hash
		a.saveState()
		return a, nil
	case testTickMsg:
		// Periodic check for running tests
		var cmds []tea.Cmd
		anyRunning := false
		for branch, status := range a.testStatus {
			if status == 1 {
				// Check if still running
				if runner.IsRunning(a.repoDir, branch) {
					anyRunning = true
				} else {
					// Finished — read result
					diskStatus := runner.ReadStatus(a.repoDir, branch)
					a.testStatus[branch] = runner.StatusToInt(diskStatus)
					// Find the commit hash for this branch
					for _, wt := range a.worktrees {
						if wt.Branch == branch {
							a.testedAt[branch] = wt.CommitHash
							break
						}
					}
					a.saveState()
				}
			}
		}
		if anyRunning {
			cmds = append(cmds, tickTestStatus())
		}
		return a, tea.Batch(cmds...)
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
		a.deleteState = 0
		a.deleteError = ""
		return a, a.loadWorktrees()
	case deleteFailedMsg:
		a.deleteState = 2 // offer force
		a.deleteError = msg.output
		return a, nil
	case flashClearMsg:
		a.flashMsg = ""
		return a, nil
	case squashDoneMsg:
		if msg.err != nil {
			a.err = msg.err
		} else {
			a.flashMsg = "Squashed to 1 commit on main"
		}
		return a, tea.Batch(a.loadWorktrees(), flashAfter(3*time.Second))
	case rebaseDoneMsg:
		if msg.err != nil {
			a.err = msg.err
		} else {
			a.flashMsg = "Rebased on main"
		}
		return a, tea.Batch(a.loadWorktrees(), flashAfter(3*time.Second))
	case tea.KeyMsg:
		if a.err != nil {
			a.err = nil
		}
		// q always quits — except during force-delete typing
		if msg.String() == "q" && a.deleteState != 2 {
			return a, tea.Quit
		}
		if a.screen == screenHelp {
			a.screen = a.prevScreen
			return a, nil
		}
		if a.screen == screenTestOutput {
			return a.updateTestOutput(msg)
		}
		if a.screen == screenGitStatus {
			return a.updateGitStatus(msg)
		}
		if msg.String() == "h" || msg.String() == "?" {
			a.prevScreen = a.screen
			a.screen = screenHelp
			return a, nil
		}
	}

	switch a.screen {
	case screenWorktreeList:
		return a.updateWorktreeList(msg)
	case screenFileList:
		return a.updateFileList(msg)
	case screenDiffView:
		return a.updateDiffView(msg)
	case screenAllDiffs:
		return a.updateAllDiffs(msg)
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
	case screenAllDiffs:
		return a.viewAllDiffs()
	case screenHelp:
		return a.viewHelp()
	case screenTestOutput:
		return a.viewTestOutput()
	case screenGitStatus:
		return a.viewGitStatus()
	}
	return ""
}

// Messages
type worktreesLoadedMsg struct {
	worktrees       []git.Worktree
	mainUncommitted int
}
type diffLoadedMsg struct{ files []git.FileDiff }
type errMsg struct{ err error }

type testDoneMsg struct {
	branch string
	hash   string
}
type testTickMsg struct{}

type landStepMsg struct{ step string }
type landDoneMsg struct{ err error }

type worktreeDeletedMsg struct{}
type deleteFailedMsg struct{ output string }
type flashClearMsg struct{}
type squashDoneMsg struct{ err error }
type rebaseDoneMsg struct{ err error }

func flashAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return flashClearMsg{}
	})
}

// tickTestStatus schedules a check on running tests after 2 seconds.
func tickTestStatus() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return testTickMsg{}
	})
}

func (a App) loadWorktrees() tea.Cmd {
	return func() tea.Msg {
		wts, err := git.ListWorktrees(a.repoDir)
		if err != nil {
			return errMsg{err}
		}
		mainDirty := git.UncommittedCount(a.repoDir)
		return worktreesLoadedMsg{worktrees: wts, mainUncommitted: mainDirty}
	}
}

// syncTestStatus checks disk for any running or completed background tests.
// Called on startup and refresh.
func (a *App) syncTestStatus() tea.Cmd {
	anyRunning := false
	for _, wt := range a.worktrees {
		diskStatus := runner.ReadStatus(a.repoDir, wt.Branch)
		if diskStatus == "" {
			continue
		}
		if diskStatus == runner.StatusRunning {
			if runner.IsRunning(a.repoDir, wt.Branch) {
				a.testStatus[wt.Branch] = 1
				anyRunning = true
			} else {
				// Process died without writing status — mark failed
				a.testStatus[wt.Branch] = 3
			}
		} else {
			a.testStatus[wt.Branch] = runner.StatusToInt(diskStatus)
		}
	}
	if anyRunning {
		return tickTestStatus()
	}
	return nil
}
