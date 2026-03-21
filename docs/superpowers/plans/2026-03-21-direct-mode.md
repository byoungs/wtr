# Direct Mode Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a "direct mode" landing screen for repos without worktrees — review uncommitted/unpushed changes on main, run tests, and push to origin.

**Architecture:** Auto-detect mode based on worktree count (>1 = worktree mode, 1 = direct mode). Direct mode shows a single-branch review screen with commits ahead/behind origin, test status, uncommitted changes count. Drill-in reuses existing file list, diff view, and git status screens. "Land" means `git push` instead of merge+push.

**Tech Stack:** Go + Bubble Tea (existing). No new dependencies.

---

## File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/git/diff.go` | Modify | Add `baseRef` param to `GetDiff` |
| `internal/git/branch.go` | Create | Branch info: ahead/behind origin, tracking ref detection |
| `internal/land/land.go` | Modify | Add `DirectSteps()` for push-only workflow |
| `internal/land/land_test.go` | Modify | Test `DirectSteps()` |
| `internal/wtr/app.go` | Modify | Add `mode` field, route to direct landing screen |
| `internal/wtr/direct_landing.go` | Create | Direct mode landing screen (update + view) |
| `internal/wtr/direct_landing_test.go` | Create | Tests for direct landing |
| `internal/wtr/help.go` | Modify | Add direct mode help section |
| `internal/wtr/worktree_list.go` | Modify | Update `loadDiff` to pass base ref |
| `cmd/wtr/main.go` | No change | Mode detection happens in `App.Init()` |

---

### Task 1: Parameterize GetDiff base ref

**Files:**
- Modify: `internal/git/diff.go:143-151`
- Modify: `internal/wtr/worktree_list.go:210`

- [ ] **Step 1: Update GetDiff signature to accept a base ref**

In `internal/git/diff.go`, change `GetDiff` to:

```go
// GetDiff runs `git -C <path> diff <baseRef>..HEAD` and parses the output.
func GetDiff(repoPath string, baseRef string) ([]FileDiff, error) {
	cmd := exec.Command("git", "-C", repoPath, "diff", baseRef+"..HEAD")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}
	return ParseDiff(string(out))
}
```

- [ ] **Step 2: Update the one call site in worktree_list.go**

In `internal/wtr/worktree_list.go`, update `loadDiff`:

```go
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
```

- [ ] **Step 3: Run tests to verify nothing broke**

Run: `make test`
Expected: All tests pass (GetDiff is not directly tested, only ParseDiff is).

- [ ] **Step 4: Commit**

```
refactor: parameterize GetDiff base ref for direct mode support
```

---

### Task 2: Add branch info utilities

**Files:**
- Create: `internal/git/branch.go`

- [ ] **Step 1: Implement branch.go**

Create `internal/git/branch.go`:

```go
package git

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// BranchInfo holds the current branch's relationship to its upstream.
type BranchInfo struct {
	Name         string // e.g. "main"
	CommitHash   string // HEAD short hash
	AheadOrigin  int    // commits ahead of origin
	BehindOrigin int    // commits behind origin
	HasUpstream  bool   // whether origin/<branch> exists
	Uncommitted  int    // uncommitted changes count
}

// GetBranchInfo returns info about the current branch relative to its origin.
func GetBranchInfo(repoDir string) (BranchInfo, error) {
	var info BranchInfo

	// Current branch name
	out, err := exec.Command("git", "-C", repoDir, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return info, fmt.Errorf("get branch: %w", err)
	}
	info.Name = strings.TrimSpace(string(out))

	// HEAD hash
	out, err = exec.Command("git", "-C", repoDir, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return info, fmt.Errorf("get hash: %w", err)
	}
	info.CommitHash = strings.TrimSpace(string(out))

	// Check if upstream exists
	upstream := "origin/" + info.Name
	err = exec.Command("git", "-C", repoDir, "rev-parse", "--verify", upstream).Run()
	if err != nil {
		// No upstream — everything local is "ahead"
		info.HasUpstream = false
		out, _ := exec.Command("git", "-C", repoDir, "rev-list", "--count", "HEAD").Output()
		info.AheadOrigin, _ = strconv.Atoi(strings.TrimSpace(string(out)))
		info.Uncommitted = UncommittedCount(repoDir)
		return info, nil
	}
	info.HasUpstream = true

	// Ahead/behind
	out, err = exec.Command("git", "-C", repoDir, "rev-list", "--count", upstream+"..HEAD").Output()
	if err == nil {
		info.AheadOrigin, _ = strconv.Atoi(strings.TrimSpace(string(out)))
	}
	out, err = exec.Command("git", "-C", repoDir, "rev-list", "--count", "HEAD.."+upstream).Output()
	if err == nil {
		info.BehindOrigin, _ = strconv.Atoi(strings.TrimSpace(string(out)))
	}

	info.Uncommitted = UncommittedCount(repoDir)
	return info, nil
}

// UpstreamRef returns "origin/<branch>" for the current branch.
// Returns empty string if no upstream is configured.
func UpstreamRef(repoDir string) string {
	out, err := exec.Command("git", "-C", repoDir, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return ""
	}
	branch := strings.TrimSpace(string(out))
	ref := "origin/" + branch
	if exec.Command("git", "-C", repoDir, "rev-parse", "--verify", ref).Run() != nil {
		return ""
	}
	return ref
}
```

- [ ] **Step 2: Run tests to verify build succeeds**

Run: `make test`
Expected: All pass. `GetBranchInfo` and `UpstreamRef` require a real git repo so are tested via integration in later tasks.

- [ ] **Step 3: Commit**

```
feat: add git branch info utilities for direct mode
```

---

### Task 3: Add DirectSteps to land package

**Files:**
- Modify: `internal/land/land.go`
- Modify: `internal/land/land_test.go`

- [ ] **Step 1: Write test for DirectSteps**

Add to `internal/land/land_test.go`:

```go
func TestDirectSteps(t *testing.T) {
	steps := DirectSteps()
	if len(steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(steps))
	}

	expected := []struct {
		name    string
		command string
	}{
		{"test", "make"},
		{"validate", "make"},
		{"push", "git"},
	}

	for i, exp := range expected {
		if steps[i].Name != exp.name {
			t.Errorf("step %d name = %q, want %q", i, steps[i].Name, exp.name)
		}
		if steps[i].Command != exp.command {
			t.Errorf("step %d command = %q, want %q", i, steps[i].Command, exp.command)
		}
	}

	// Verify push uses "push"
	if steps[2].Args[0] != "push" {
		t.Errorf("push args[0] = %q, want %q", steps[2].Args[0], "push")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `make test`
Expected: FAIL — `DirectSteps` undefined.

- [ ] **Step 3: Implement DirectSteps**

Add to `internal/land/land.go`:

```go
// DirectSteps returns the steps for pushing the current branch to origin.
// No merge step — we're already on the branch.
func DirectSteps() []Step {
	return []Step{
		{"test", "make", []string{"test"}},
		{"validate", "make", []string{"validate"}},
		{"push", "git", []string{"push"}},
	}
}
```

- [ ] **Step 4: Update Run to accept steps as parameter**

Currently `Run` has signature `Run(repoDir string, branch string, logPath string, onStep func(Step))` and calls `Steps(branch)` internally. Change the second parameter from `branch string` to `steps []Step`:

```go
// Run executes land steps sequentially from repoDir.
func Run(repoDir string, steps []Step, logPath string, onStep func(Step)) ([]StepResult, error) {
```

Update the loop from `for _, step := range Steps(branch)` to `for _, step := range steps`.

Update the one call site in `worktree_list.go:154`:
```go
// Before: land.Run(a.repoDir, wt.Branch, logFile, func(s land.Step) {})
// After:
land.Run(a.repoDir, land.Steps(wt.Branch), logFile, func(s land.Step) {})
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `make test`
Expected: All pass.

- [ ] **Step 6: Commit**

```
feat: add DirectSteps and parameterize land.Run
```

---

### Task 4: Add mode detection and direct landing screen

**Files:**
- Modify: `internal/wtr/app.go`
- Create: `internal/wtr/direct_landing.go`

- [ ] **Step 1: Add mode and Push key to app.go and keys.go**

In `internal/wtr/app.go`, add to the `screen` const block:

```go
screenDirectLanding
```

Add to `App` struct:

```go
// Direct mode
mode       string // "worktree" or "direct"
branchInfo git.BranchInfo
```

No new key binding needed — reuse `keys.Land` (`l`) in the direct landing handler. Same key, different action depending on mode.

- [ ] **Step 2: Add mode detection in Init**

Modify `App.Init()` in `app.go`:

```go
func (a App) Init() tea.Cmd {
	return a.loadWorktrees()
}
```

The mode detection happens when worktrees load. In `updateWorktreeList`, after receiving `worktreesLoadedMsg`, if `len(worktrees) == 0`, switch to direct mode. Update the handler in `app.go`:

In the `worktreesLoadedMsg` handler in `updateWorktreeList`, add at the end (after `syncTestStatus`):

```go
// Auto-detect mode: no worktrees → direct mode
if len(a.worktrees) == 0 {
	a.mode = "direct"
	a.screen = screenDirectLanding
	// Populate a synthetic worktree so shared screens (file list, git status,
	// test output, diff view) can use a.worktrees[a.selectedWorktree] without
	// special-casing direct mode.
	return a, a.loadBranchInfo()
}
a.mode = "worktree"
```

- [ ] **Step 3: Add loadBranchInfo command**

Add to `app.go`:

```go
type branchInfoMsg struct{ info git.BranchInfo }

func (a App) loadBranchInfo() tea.Cmd {
	return func() tea.Msg {
		info, err := git.GetBranchInfo(a.repoDir)
		if err != nil {
			return errMsg{err}
		}
		return branchInfoMsg{info}
	}
}
```

- [ ] **Step 4: Create direct_landing.go**

Create `internal/wtr/direct_landing.go`:

```go
package wtr

import (
	"fmt"
	"strings"

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
				a.landStep = "(v: view output)"
				logFile := runner.LogPath(a.repoDir, a.branchInfo.Name)
				return a, func() tea.Msg {
					_, err := land.Run(a.repoDir, land.DirectSteps(), logFile, func(s land.Step) {})
					return landDoneMsg{err: err}
				}
			}
		case key.Matches(msg, keys.GitStatus):
			a.statusFiles = loadGitStatus(a.repoDir)
			if len(a.statusFiles) > 0 {
				a.statusCursor = 0
				a.confirmRevert = false
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

	title := styleTitle.Width(a.width).Render("Review")
	b.WriteString(title + "\n\n")

	if a.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(colorRed).Width(a.width - 4)
		b.WriteString(errStyle.Render(fmt.Sprintf("  Error: %v", a.err)) + "\n")
		b.WriteString(styleHelp.Render("  (v: view full output  any key: dismiss)") + "\n\n")
	}

	// Branch name and hash
	branchLabel := lipgloss.NewStyle().Bold(true).Foreground(colorBlue)
	b.WriteString("  " + branchLabel.Render(a.branchInfo.Name))
	if a.branchInfo.CommitHash != "" {
		b.WriteString("  " + lipgloss.NewStyle().Foreground(colorSubtle).Render(a.branchInfo.CommitHash))
	}
	b.WriteString("\n\n")

	// Ahead/behind origin
	if !a.branchInfo.HasUpstream {
		b.WriteString("  " + styleRunning.Render("No remote tracking branch") + "\n")
	} else if a.branchInfo.AheadOrigin == 0 && a.branchInfo.BehindOrigin == 0 {
		b.WriteString("  " + stylePass.Render("Up to date with origin") + "\n")
	} else {
		if a.branchInfo.AheadOrigin > 0 {
			b.WriteString("  " + styleRunning.Render(fmt.Sprintf("↑%d unpushed commit(s)", a.branchInfo.AheadOrigin)) + "\n")
		}
		if a.branchInfo.BehindOrigin > 0 {
			b.WriteString("  " + styleFail.Render(fmt.Sprintf("↓%d behind origin", a.branchInfo.BehindOrigin)) + "\n")
		}
	}

	// Test status
	testStatus := a.testStatus[a.branchInfo.Name]
	switch testStatus {
	case 1:
		b.WriteString("  " + styleRunning.Render("⟳ Tests running...") + "\n")
	case 2:
		b.WriteString("  " + stylePass.Render("✓ Tests passed") + "\n")
	case 3:
		b.WriteString("  " + styleFail.Render("✗ Tests failed") + "\n")
	}

	// Uncommitted changes
	if a.branchInfo.Uncommitted > 0 {
		b.WriteString("  " + styleRunning.Render(fmt.Sprintf("△ %d uncommitted change(s)", a.branchInfo.Uncommitted)) + "\n")
	}

	b.WriteString("\n")

	// Landing status
	if a.landing {
		b.WriteString(styleRunning.Render(fmt.Sprintf("  Pushing %s... %s\n", a.landBranch, a.landStep)))
		b.WriteString("\n")
	}

	if a.flashMsg != "" {
		b.WriteString(styleRunning.Render("  "+a.flashMsg) + "\n")
		b.WriteString("\n")
	}

	padToBottom(&b, a.height, strings.Count(b.String(), "\n"))
	b.WriteString(styleHelp.Render("  q:quit  h:help  →review  g:status  t:test  v:output  l:push  u:refresh"))

	return b.String()
}
```

- [ ] **Step 5: Wire up routing in app.go**

Add `screenDirectLanding` to the `Update` switch:

```go
case screenDirectLanding:
    return a.updateDirectLanding(msg)
```

Add to `View` switch:

```go
case screenDirectLanding:
    return a.viewDirectLanding()
```

Handle `branchInfoMsg` in the main Update (before the screen switch). This is where the synthetic worktree gets populated — it must happen here since the screen-specific handler won't be reached after this returns:

```go
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
    }
    return a, nil
```

Remove the duplicate `branchInfoMsg` handler from `updateDirectLanding` — the main Update handler handles it.

Handle `landDoneMsg` — already handled globally, but need to reload branch info instead of worktrees when in direct mode:

In the `landDoneMsg` handler, change:
```go
case landDoneMsg:
    a.landing = false
    if msg.err != nil {
        a.err = msg.err
    }
    if a.mode == "direct" {
        return a, a.loadBranchInfo()
    }
    return a, a.loadWorktrees()
```

- [ ] **Step 6: Wire up file list back navigation for direct mode**

In `internal/wtr/file_list.go`, update the Back handler:

```go
case key.Matches(msg, keys.Back):
    if a.mode == "direct" {
        a.screen = screenDirectLanding
    } else {
        a.screen = screenWorktreeList
    }
```

Also in `internal/wtr/git_status.go`, update Back handler:

```go
case key.Matches(msg, keys.Back):
    if a.mode == "direct" && a.prevScreen == screenDirectLanding {
        a.screen = screenDirectLanding
    } else {
        a.screen = screenFileList
    }
    statusScrollY = 0
```

- [ ] **Step 7: Add missing import for time in direct_landing.go**

The `direct_landing.go` file uses `time.Second` — make sure `"time"` is in the imports.

- [ ] **Step 8: Run tests and build**

Run: `make test`
Run: `make build`
Expected: All pass, binary builds.

- [ ] **Step 9: Commit**

```
feat: add direct mode landing screen for single-branch repos
```

---

### Task 5: Update help screen for direct mode

**Files:**
- Modify: `internal/wtr/help.go`

- [ ] **Step 1: Add direct mode section to help**

Add after the "Worktree List" section:

```go
if a.mode == "direct" {
    b.WriteString(heading.Render("  Review (Direct Mode)") + "\n\n")
    directKeys := [][2]string{
        {"→ / enter", "Review changed files"},
        {"g", "Git status (uncommitted changes)"},
        {"t", "Run make validate (background)"},
        {"v", "View test/validate output"},
        {"l", "Push to origin (test + validate + push)"},
        {"u", "Refresh"},
        {"h / ?", "This help screen"},
        {"q", "Quit"},
    }
    for _, kv := range directKeys {
        b.WriteString("  " + label.Render(kv[0]) + desc.Render(kv[1]) + "\n")
    }
} else {
    // existing worktree list help
}
```

- [ ] **Step 2: Run tests**

Run: `make test`
Expected: All pass. The navbar test for worktree list is unchanged.

- [ ] **Step 3: Commit**

```
feat: add direct mode help section
```

---

### Task 6: Add direct landing tests

**Files:**
- Create: `internal/wtr/direct_landing_test.go`

- [ ] **Step 1: Write tests**

Create `internal/wtr/direct_landing_test.go`:

```go
package wtr

import (
	"strings"
	"testing"

	"github.com/byoungs/wtr/internal/git"
)

func TestDirectLandingView_ShowsAheadCount(t *testing.T) {
	a := NewApp("/tmp/fake")
	a.mode = "direct"
	a.screen = screenDirectLanding
	a.width = 80
	a.height = 24
	a.branchInfo = git.BranchInfo{
		Name:        "main",
		CommitHash:  "abc1234",
		AheadOrigin: 3,
		HasUpstream: true,
	}

	view := a.viewDirectLanding()
	if !strings.Contains(view, "3 unpushed") {
		t.Error("should show unpushed commit count")
	}
}

func TestDirectLandingView_ShowsUpToDate(t *testing.T) {
	a := NewApp("/tmp/fake")
	a.mode = "direct"
	a.screen = screenDirectLanding
	a.width = 80
	a.height = 24
	a.branchInfo = git.BranchInfo{
		Name:        "main",
		HasUpstream: true,
	}

	view := a.viewDirectLanding()
	if !strings.Contains(view, "Up to date") {
		t.Error("should show up to date message")
	}
}

func TestDirectLandingView_ShowsTestStatus(t *testing.T) {
	a := NewApp("/tmp/fake")
	a.mode = "direct"
	a.screen = screenDirectLanding
	a.width = 80
	a.height = 24
	a.branchInfo = git.BranchInfo{Name: "main", HasUpstream: true}
	a.testStatus["main"] = 2

	view := a.viewDirectLanding()
	if !strings.Contains(view, "passed") {
		t.Error("should show test passed status")
	}
}

func TestDirectLandingNavbar(t *testing.T) {
	navbar := "  q:quit  h:help  →review  g:status  t:test  v:output  l:push  u:refresh"
	expected := []string{"q:", "h:", "→review", "g:", "t:", "v:", "l:", "u:"}
	for _, e := range expected {
		if !strings.Contains(navbar, e) {
			t.Errorf("direct landing navbar missing %q", e)
		}
	}
}
```

- [ ] **Step 2: Run tests**

Run: `make test`
Expected: All pass.

- [ ] **Step 3: Commit**

```
test: add direct mode landing screen tests
```

---

### Task 7: Update CLAUDE.md

**Files:**
- Modify: `CLAUDE.md`

- [ ] **Step 1: Add direct mode documentation**

Add after the "Screens" section:

```markdown
## Modes

wtr auto-detects which mode to use:

- **Worktree mode** — when git worktrees exist (besides main). Landing screen shows worktree list.
- **Direct mode** — when only main branch exists (no worktrees). Landing screen shows branch review with unpushed commits, test status, and push action.

### Direct Mode Keys
- `→`/`enter`: review changed files (diff vs origin)
- `g`: git status (uncommitted changes)
- `t`: run make validate (background)
- `v`: view test/validate output
- `l`: push to origin (test + validate + push)
- `u`: refresh
- `h`/`?`: help
- `q`: quit
```

Update the Project Structure section to include new files:

```
    internal/git/branch.go         Branch info (ahead/behind origin)
    internal/wtr/direct_landing.go Direct mode landing screen
```

- [ ] **Step 2: Commit**

```
docs: add direct mode to CLAUDE.md
```

---

### Task 8: Manual integration test

- [ ] **Step 1: Build and run in this repo (which has no worktrees)**

Run: `make build`
Run: `bin/wtr`
Expected: Should show direct mode landing screen with branch info for main.

- [ ] **Step 2: Verify drill-in works**

Press `→` to review files. Verify file list shows diff vs origin. Press `←` to go back to direct landing.

- [ ] **Step 3: Verify git status works**

Press `g` to view uncommitted changes. Press `←` to go back.

- [ ] **Step 4: Verify test runner works**

Press `t` to run tests. Press `v` to view output. Press `←` to go back.

- [ ] **Step 5: Verify help shows direct mode keys**

Press `h` to view help. Verify direct mode section appears.
