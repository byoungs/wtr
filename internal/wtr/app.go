package wtr

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/byoungs/wtr/internal/git"
	"github.com/byoungs/wtr/internal/runner"
	"github.com/byoungs/wtr/internal/state"
	tea "github.com/charmbracelet/bubbletea"
)

// lastLogStep extracts the most recent step name from a land log.
// Log lines look like: "==> merge: git [merge --ff-only branch]"
func lastLogStep(logContent string) string {
	step := ""
	for _, line := range strings.Split(logContent, "\n") {
		if strings.HasPrefix(line, "==> ") {
			// Extract step name (e.g. "merge" from "==> merge: git ...")
			rest := strings.TrimPrefix(line, "==> ")
			if idx := strings.Index(rest, ":"); idx > 0 {
				step = rest[:idx]
			} else if idx := strings.Index(rest, " "); idx > 0 {
				step = rest[:idx]
			}
		}
	}
	return step
}

type screen int

const (
	screenWorktreeList screen = iota
	screenFileList
	screenDiffView
	screenAllDiffs
	screenHelp
	screenTestOutput
	screenGitStatus
	screenDirectLanding
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
	reviewed         map[string]bool   // key: "branch:filename"
	reviewedAt       map[string]string // branch -> commit hash when reviews were done
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
	landStarted time.Time

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
	confirmQuit    bool

	// Main branch info
	mainUncommitted int

	// File search
	searching   bool
	searchQuery string

	// Direct mode
	mode       string // "worktree" or "direct"
	branchInfo git.BranchInfo
}

// addTempMain appends a temporary "main" worktree entry for git status viewing,
// but only if one isn't already present.
func (a *App) addTempMain() {
	for _, wt := range a.worktrees {
		if wt.Branch == "main" {
			return
		}
	}
	a.worktrees = append(a.worktrees, git.Worktree{
		Path:        a.repoDir,
		Branch:      "main",
		CommitHash:  git.CurrentHash(a.repoDir),
		Uncommitted: a.mainUncommitted,
	})
}

// removeTempMain removes the temporary "main" entry from worktrees.
func (a *App) removeTempMain() {
	filtered := a.worktrees[:0]
	for _, wt := range a.worktrees {
		if wt.Branch != "main" {
			filtered = append(filtered, wt)
		}
	}
	a.worktrees = filtered
	if a.selectedWorktree >= len(a.worktrees) && len(a.worktrees) > 0 {
		a.selectedWorktree = len(a.worktrees) - 1
	}
}

func NewApp(repoDir string) App {
	s := state.Load(repoDir)
	reviewed := s.Reviewed
	if reviewed == nil {
		reviewed = make(map[string]bool)
	}
	reviewedAt := s.ReviewedAt
	if reviewedAt == nil {
		reviewedAt = make(map[string]string)
	}
	return App{
		repoDir:    repoDir,
		reviewed:   reviewed,
		reviewedAt: reviewedAt,
		sideBySide: false,
		testStatus: s.TestStatus,
		testedAt:   s.TestedAt,
	}
}

type autoRefreshMsg struct{}

func tickAutoRefresh() tea.Cmd {
	return tea.Tick(30*time.Second, func(time.Time) tea.Msg {
		return autoRefreshMsg{}
	})
}

func (a App) Init() tea.Cmd {
	return tea.Batch(a.loadWorktrees(), tickAutoRefresh())
}

func (a App) saveState() {
	state.Save(a.repoDir, state.State{
		TestStatus: a.testStatus,
		TestedAt:   a.testedAt,
		Reviewed:   a.reviewed,
		ReviewedAt: a.reviewedAt,
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
	case autoRefreshMsg:
		// Only refresh on landing screens, always re-schedule
		if a.screen == screenWorktreeList {
			return a, tea.Batch(a.loadWorktrees(), tickAutoRefresh())
		}
		if a.screen == screenDirectLanding {
			return a, tea.Batch(a.loadBranchInfo(), tickAutoRefresh())
		}
		return a, tickAutoRefresh()
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
					debugLog(a.repoDir, "[tick] branch=%s finished, disk_status=%q resolved=%d", branch, diskStatus, runner.StatusToInt(diskStatus))
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
	case landTickMsg:
		if !a.landing {
			return a, nil
		}
		// Read current step from log file
		logContent := runner.ReadLog(a.repoDir, a.landBranch)
		if logContent != "" {
			a.landStep = lastLogStep(logContent)
		}
		return a, tickLandStatus()
	case branchInfoMsg:
		a.branchInfo = msg.info
		// In direct mode, populate synthetic worktree so shared screens work
		if a.mode == "direct" {
			a.worktrees = []git.Worktree{{
				Path:        a.repoDir,
				Branch:      msg.info.Name,
				CommitHash:  msg.info.CommitHash,
				Uncommitted: msg.info.Uncommitted,
			}}
			a.selectedWorktree = 0
			// Sync test status from disk (restores state after restart)
			cmd := a.syncTestStatus()
			return a, cmd
		}
		return a, nil
	case landDoneMsg:
		a.landing = false
		if msg.err != nil {
			a.err = msg.err
		}
		if a.mode == "direct" {
			return a, a.loadBranchInfo()
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
		// Handle quit confirmation prompt
		if a.confirmQuit {
			if msg.String() == "y" || msg.String() == "Y" {
				return a, tea.Quit
			}
			a.confirmQuit = false
			return a, nil
		}
		if msg.Type == tea.KeyCtrlC {
			if a.landing {
				a.confirmQuit = true
				return a, nil
			}
			return a, tea.Quit
		}
		if a.err != nil {
			a.err = nil
		}
		// q always quits — except during force-delete typing or search
		if msg.String() == "q" && a.deleteState != 2 && !a.searching {
			if a.landing {
				a.confirmQuit = true
				return a, nil
			}
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
		if !a.searching && (msg.String() == "h" || msg.String() == "?") {
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
	case screenDirectLanding:
		return a.updateDirectLanding(msg)
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
	case screenDirectLanding:
		return a.viewDirectLanding()
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
type landTickMsg struct{}
type landDoneMsg struct{ err error }

type worktreeDeletedMsg struct{}
type deleteFailedMsg struct{ output string }
type flashClearMsg struct{}
type squashDoneMsg struct{ err error }
type rebaseDoneMsg struct{ err error }
type branchInfoMsg struct{ info git.BranchInfo }

func flashAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return flashClearMsg{}
	})
}

// fmtElapsed formats a duration as "Xs" or "Xm Ys".
func fmtElapsed(d time.Duration) string {
	d = d.Truncate(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", m, s)
}

// tickTestStatus schedules a check on running tests after 2 seconds.
func tickTestStatus() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return testTickMsg{}
	})
}

// tickLandStatus schedules a check on the landing log after 1 second.
func tickLandStatus() tea.Cmd {
	return tea.Tick(1*time.Second, func(time.Time) tea.Msg {
		return landTickMsg{}
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

func (a App) loadBranchInfo() tea.Cmd {
	return func() tea.Msg {
		info, err := git.GetBranchInfo(a.repoDir)
		if err != nil {
			return errMsg{err}
		}
		return branchInfoMsg{info}
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
				debugLog(a.repoDir, "[sync] branch=%s still running", wt.Branch)
			} else {
				// Process is gone — re-read status in case it finished between our two reads
				finalStatus := runner.ReadStatus(a.repoDir, wt.Branch)
				if finalStatus != runner.StatusRunning && finalStatus != "" {
					a.testStatus[wt.Branch] = runner.StatusToInt(finalStatus)
					debugLog(a.repoDir, "[sync] branch=%s finished (race resolved), status=%q", wt.Branch, finalStatus)
				} else {
					// Process truly died without writing status
					a.testStatus[wt.Branch] = 3
					debugLog(a.repoDir, "[sync] branch=%s status=running but process dead, marking failed", wt.Branch)
				}
			}
		} else {
			a.testStatus[wt.Branch] = runner.StatusToInt(diskStatus)
			debugLog(a.repoDir, "[sync] branch=%s disk_status=%q", wt.Branch, diskStatus)
		}
	}
	if anyRunning {
		return tickTestStatus()
	}
	return nil
}

// debugLog appends a timestamped line to .git/wtr/debug.log.
func debugLog(repoDir, format string, args ...any) {
	dir := filepath.Join(repoDir, ".git", "wtr")
	os.MkdirAll(dir, 0755)
	f, err := os.OpenFile(filepath.Join(dir, "debug.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(f, "%s %s\n", time.Now().Format("15:04:05.000"), msg)
}
