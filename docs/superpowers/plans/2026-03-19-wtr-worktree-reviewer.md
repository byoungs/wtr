# `wtr` — Worktree Review TUI

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A terminal UI for reviewing git worktree diffs file-by-file, marking files as reviewed, running tests in background, and landing branches on main via ff-only merge + test + validate + push.

**Architecture:** Go TUI built with Bubble Tea (charmbracelet). Three-screen navigation: worktree list → file list → diff view. Git operations via `os/exec` calling git CLI. No database — state is the git worktrees themselves plus in-memory "reviewed" checkmarks. Runs from repo root on main branch.

**Tech Stack:** Go 1.26, Bubble Tea (bubbletea), Lip Gloss (lipgloss), Bubbles (bubbles), git CLI

---

## File Structure

```
wtr/
├── cmd/wtr/main.go                  # Entry point: parse args, launch TUI
├── internal/
│   ├── git/
│   │   ├── worktree.go             # Discover worktrees via `git worktree list`
│   │   ├── worktree_test.go        # Tests for worktree discovery
│   │   ├── diff.go                 # Parse `git diff` output into structured types
│   │   └── diff_test.go            # Tests for diff parsing
│   ├── tui/
│   │   ├── app.go                  # Root Bubble Tea model, screen routing
│   │   ├── worktree_list.go        # Screen 1: list worktrees with stats
│   │   ├── file_list.go            # Screen 2: list changed files in a worktree
│   │   ├── diff_view.go            # Screen 3: render diff (side-by-side + unified)
│   │   ├── styles.go               # Lip Gloss style definitions
│   │   └── keys.go                 # Key binding definitions
│   └── land/
│       ├── land.go                 # Land workflow: merge + test + validate + push
│       └── land_test.go            # Tests for land workflow
├── Makefile                        # Build, test, install targets
├── CLAUDE.md                       # Agent instructions
├── go.mod
└── go.sum
```

**Design notes:**
- `internal/git/` is a pure library — no TUI dependencies, fully testable with fixture data
- `internal/tui/` owns all Bubble Tea models and views — one file per screen
- `internal/land/` isolates the merge+test+validate+push workflow so it can be tested independently
- `cmd/wtr/main.go` is just argument parsing and wiring

---

### Task 1: Project Scaffold

**Files:**
- Create: `cmd/wtr/main.go`
- Create: `Makefile`
- Create: `CLAUDE.md`
- Create: `go.mod`

- [ ] **Step 1: Initialize Go module**

Run: `go mod init github.com/byoungs/wtr`

- [ ] **Step 2: Add Bubble Tea dependencies**

Run: `go get github.com/charmbracelet/bubbletea@latest github.com/charmbracelet/lipgloss@latest github.com/charmbracelet/bubbles@latest`

- [ ] **Step 3: Create minimal main.go**

Create `cmd/wtr/main.go`:
```go
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// For now, just verify Bubble Tea works
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

type model struct{}

func initialModel() model { return model{} }

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	return "wtr - worktree reviewer\n\nPress q to quit.\n"
}
```

- [ ] **Step 4: Create Makefile**

Create `Makefile` following ai-scheduler patterns:
```makefile
.PHONY: build test install clean

BINARY := wtr
BUILD_DIR := bin

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/wtr

test:
	go test ./...

install: build
	cp $(BUILD_DIR)/$(BINARY) ~/go/bin/$(BINARY)

clean:
	rm -rf $(BUILD_DIR)
```

- [ ] **Step 5: Create CLAUDE.md**

Create `CLAUDE.md` with project description, build commands, and conventions.

- [ ] **Step 6: Verify build and run**

Run: `go build ./cmd/wtrr`
Expected: Binary builds with no errors.

Run: `go run ./cmd/wtrr` (then press q)
Expected: Shows "wtr - worktree reviewer" and exits cleanly on q.

- [ ] **Step 7: Commit**

```
git init && git add -A && git commit -m "scaffold: go module, bubble tea shell, makefile"
```

---

### Task 2: Git Worktree Discovery

**Files:**
- Create: `internal/git/worktree.go`
- Create: `internal/git/worktree_test.go`

- [ ] **Step 1: Define worktree types**

Create `internal/git/worktree.go` with types:
```go
package git

// Worktree represents a git worktree with its diff stats.
type Worktree struct {
	Path       string // Absolute path to worktree directory
	Branch     string // Branch name (e.g. "worktree-auth-fix")
	CommitHash string // HEAD commit short hash
	FilesChanged int  // Number of files changed vs main
	Insertions   int  // Lines added
	Deletions    int  // Lines deleted
}
```

- [ ] **Step 2: Write failing test for ListWorktrees**

Create `internal/git/worktree_test.go`:
```go
package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestListWorktrees(t *testing.T) {
	// Set up a temp git repo with a worktree
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}

	run("init", "-b", "main")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")

	// Create initial commit on main
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello"), 0644)
	run("add", ".")
	run("commit", "-m", "initial")

	// Create a worktree branch with changes
	wtDir := filepath.Join(dir, "wt-test")
	run("worktree", "add", "-b", "worktree-test", wtDir)
	os.WriteFile(filepath.Join(wtDir, "new.txt"), []byte("new file\n"), 0644)
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = wtDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "add new file")
	cmd.Dir = wtDir
	cmd.Run()

	worktrees, err := ListWorktrees(dir)
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}

	// Should find at least the worktree we created (excludes main)
	if len(worktrees) == 0 {
		t.Fatal("expected at least 1 worktree, got 0")
	}

	found := false
	for _, wt := range worktrees {
		if wt.Branch == "worktree-test" {
			found = true
			if wt.FilesChanged != 1 {
				t.Errorf("expected 1 file changed, got %d", wt.FilesChanged)
			}
		}
	}
	if !found {
		t.Error("worktree-test branch not found in worktree list")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/git/...`
Expected: FAIL — `ListWorktrees` not defined.

- [ ] **Step 4: Implement ListWorktrees**

Add to `internal/git/worktree.go`:
```go
// ListWorktrees discovers all git worktrees for the repo at repoDir,
// excluding the main worktree. Returns diff stats for each vs main.
func ListWorktrees(repoDir string) ([]Worktree, error) {
	// git worktree list --porcelain
	out, err := exec.Command("git", "-C", repoDir, "worktree", "list", "--porcelain").Output()
	if err != nil {
		return nil, fmt.Errorf("git worktree list: %w", err)
	}

	var worktrees []Worktree
	var current Worktree
	isMain := true

	for _, line := range strings.Split(string(out), "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			if !isMain && current.Path != "" {
				worktrees = append(worktrees, current)
			}
			current = Worktree{Path: strings.TrimPrefix(line, "worktree ")}
			if isMain {
				isMain = false
				continue
			}
		case strings.HasPrefix(line, "branch "):
			ref := strings.TrimPrefix(line, "branch ")
			current.Branch = filepath.Base(ref) // refs/heads/foo -> foo
		case strings.HasPrefix(line, "HEAD "):
			current.CommitHash = strings.TrimPrefix(line, "HEAD ")[:7]
		}
	}
	// Don't forget the last one
	if current.Path != "" && !isMain {
		worktrees = append(worktrees, current)
	}

	// Enrich with diff stats
	for i := range worktrees {
		enrichDiffStats(repoDir, &worktrees[i])
	}

	return worktrees, nil
}

func enrichDiffStats(repoDir string, wt *Worktree) {
	out, err := exec.Command("git", "-C", wt.Path, "diff", "--stat", "main..HEAD").Output()
	if err != nil {
		return // stats are best-effort
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 {
		return
	}
	// Last line is summary: " 3 files changed, 47 insertions(+), 12 deletions(-)"
	summary := lines[len(lines)-1]
	fmt.Sscanf(summary, " %d file", &wt.FilesChanged)
	if idx := strings.Index(summary, "insertion"); idx > 0 {
		// Walk back to find the number
		parts := strings.Fields(summary)
		for i, p := range parts {
			if strings.HasPrefix(p, "insertion") && i > 0 {
				fmt.Sscanf(parts[i-1], "%d", &wt.Insertions)
			}
			if strings.HasPrefix(p, "deletion") && i > 0 {
				fmt.Sscanf(parts[i-1], "%d", &wt.Deletions)
			}
		}
	}
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/git/...`
Expected: PASS

- [ ] **Step 6: Commit**

```
git add internal/git/worktree.go internal/git/worktree_test.go
git commit -m "feat: worktree discovery via git worktree list"
```

---

### Task 3: Git Diff Parsing

**Files:**
- Create: `internal/git/diff.go`
- Create: `internal/git/diff_test.go`

- [ ] **Step 1: Define diff types**

Create `internal/git/diff.go`:
```go
package git

// FileDiff represents the diff for a single file.
type FileDiff struct {
	OldName string // Original filename (empty for new files)
	NewName string // New filename (empty for deleted files)
	Hunks   []Hunk
	Binary  bool // True if binary file
}

// Hunk represents a contiguous block of changes.
type Hunk struct {
	OldStart int    // Starting line in old file
	OldLines int    // Number of lines in old file
	NewStart int    // Starting line in new file
	NewLines int    // Number of lines in new file
	Header   string // @@ line
	Lines    []DiffLine
}

// DiffLine is a single line in a diff hunk.
type DiffLine struct {
	Type    LineType
	Content string // Line content WITHOUT the +/- prefix
}

type LineType int

const (
	LineContext  LineType = iota // unchanged line (space prefix)
	LineAdded                    // added line (+ prefix)
	LineRemoved                  // removed line (- prefix)
)
```

- [ ] **Step 2: Write failing test for ParseDiff**

Create `internal/git/diff_test.go` with a hardcoded unified diff string:
```go
package git

import "testing"

const testDiff = `diff --git a/internal/auth/middleware.go b/internal/auth/middleware.go
index abc1234..def5678 100644
--- a/internal/auth/middleware.go
+++ b/internal/auth/middleware.go
@@ -42,7 +42,12 @@ func AuthMiddleware(next http.Handler) http.Handler {
 	mux := http.NewServeMux()
-	token := r.Header.Get("Authorization")
+	token, err := extractBearerToken(r)
+	if err != nil {
+		http.Error(w, "unauthorized", 401)
+		return
+	}
 	// validate token
diff --git a/internal/auth/token.go b/internal/auth/token.go
new file mode 100644
index 0000000..abc1234
--- /dev/null
+++ b/internal/auth/token.go
@@ -0,0 +1,5 @@
+package auth
+
+func extractBearerToken(r *http.Request) (string, error) {
+	return "", nil
+}
`

func TestParseDiff(t *testing.T) {
	files, err := ParseDiff(testDiff)
	if err != nil {
		t.Fatalf("ParseDiff: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	// First file: modification
	f := files[0]
	if f.NewName != "internal/auth/middleware.go" {
		t.Errorf("file 0 name = %q, want middleware.go", f.NewName)
	}
	if len(f.Hunks) != 1 {
		t.Fatalf("file 0: expected 1 hunk, got %d", len(f.Hunks))
	}
	hunk := f.Hunks[0]
	if hunk.OldStart != 42 {
		t.Errorf("hunk old start = %d, want 42", hunk.OldStart)
	}

	// Count added/removed lines
	var added, removed int
	for _, l := range hunk.Lines {
		switch l.Type {
		case LineAdded:
			added++
		case LineRemoved:
			removed++
		}
	}
	if added != 5 || removed != 1 {
		t.Errorf("hunk: added=%d removed=%d, want added=5 removed=1", added, removed)
	}

	// Second file: new file
	f2 := files[1]
	if f2.OldName != "/dev/null" {
		t.Errorf("file 1 old name = %q, want /dev/null", f2.OldName)
	}
	if f2.NewName != "internal/auth/token.go" {
		t.Errorf("file 1 new name = %q, want token.go", f2.NewName)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/git/...`
Expected: FAIL — `ParseDiff` not defined.

- [ ] **Step 4: Implement ParseDiff**

Add to `internal/git/diff.go` — a line-by-line parser for unified diff format:
```go
// ParseDiff parses unified diff output into structured FileDiff entries.
func ParseDiff(raw string) ([]FileDiff, error) {
	var files []FileDiff
	var current *FileDiff
	var currentHunk *Hunk
	lines := strings.Split(raw, "\n")

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "diff --git"):
			if current != nil {
				files = append(files, *current)
			}
			current = &FileDiff{}
			currentHunk = nil

		case strings.HasPrefix(line, "--- "):
			if current != nil {
				name := strings.TrimPrefix(line, "--- ")
				name = strings.TrimPrefix(name, "a/")
				current.OldName = name
			}

		case strings.HasPrefix(line, "+++ "):
			if current != nil {
				name := strings.TrimPrefix(line, "+++ ")
				name = strings.TrimPrefix(name, "b/")
				current.NewName = name
			}

		case strings.HasPrefix(line, "@@"):
			if current != nil {
				h := Hunk{Header: line}
				// Parse @@ -old,count +new,count @@
				fmt.Sscanf(line, "@@ -%d,%d +%d,%d",
					&h.OldStart, &h.OldLines, &h.NewStart, &h.NewLines)
				current.Hunks = append(current.Hunks, h)
				currentHunk = &current.Hunks[len(current.Hunks)-1]
			}

		case strings.HasPrefix(line, "Binary files"):
			if current != nil {
				current.Binary = true
			}

		default:
			if currentHunk == nil || current == nil {
				continue
			}
			if len(line) == 0 {
				continue
			}
			dl := DiffLine{Content: line[1:]} // strip prefix
			switch line[0] {
			case '+':
				dl.Type = LineAdded
			case '-':
				dl.Type = LineRemoved
			case ' ':
				dl.Type = LineContext
			default:
				continue
			}
			currentHunk.Lines = append(currentHunk.Lines, dl)
		}
	}

	if current != nil {
		files = append(files, *current)
	}

	return files, nil
}

// GetDiff returns the parsed diff between main and HEAD for a worktree.
func GetDiff(worktreePath string) ([]FileDiff, error) {
	out, err := exec.Command("git", "-C", worktreePath, "diff", "main..HEAD").Output()
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}
	return ParseDiff(string(out))
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/git/...`
Expected: PASS

- [ ] **Step 6: Commit**

```
git add internal/git/diff.go internal/git/diff_test.go
git commit -m "feat: unified diff parser with file/hunk/line types"
```

---

### Task 4: TUI Styles and Key Bindings

**Files:**
- Create: `internal/tui/styles.go`
- Create: `internal/tui/keys.go`

- [ ] **Step 1: Create styles.go**

Define Lip Gloss styles for the entire app:
```go
package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	colorGreen   = lipgloss.Color("#a6e3a1")
	colorRed     = lipgloss.Color("#f38ba8")
	colorYellow  = lipgloss.Color("#f9e2af")
	colorBlue    = lipgloss.Color("#89b4fa")
	colorSubtle  = lipgloss.Color("#6c7086")
	colorText    = lipgloss.Color("#cdd6f4")
	colorBg      = lipgloss.Color("#1e1e2e")
	colorHighlight = lipgloss.Color("#313244")

	// Diff line styles
	styleAdded   = lipgloss.NewStyle().Foreground(colorGreen)
	styleRemoved = lipgloss.NewStyle().Foreground(colorRed)
	styleContext = lipgloss.NewStyle().Foreground(colorText)
	styleHunkHeader = lipgloss.NewStyle().Foreground(colorBlue).Bold(true)

	// List styles
	styleSelected  = lipgloss.NewStyle().Background(colorHighlight).Bold(true)
	styleNormal    = lipgloss.NewStyle().Foreground(colorText)
	styleReviewed  = lipgloss.NewStyle().Foreground(colorGreen)

	// Status indicators
	stylePass = lipgloss.NewStyle().Foreground(colorGreen).SetString("✓")
	styleFail = lipgloss.NewStyle().Foreground(colorRed).SetString("✗")
	styleRunning = lipgloss.NewStyle().Foreground(colorYellow).SetString("⟳")
	stylePending = lipgloss.NewStyle().Foreground(colorSubtle).SetString("—")

	// Layout
	styleTitle = lipgloss.NewStyle().Bold(true).Foreground(colorBlue).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(colorSubtle)
	styleHelp = lipgloss.NewStyle().Foreground(colorSubtle)
)
```

- [ ] **Step 2: Create keys.go**

Define key bindings:
```go
package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
	Back     key.Binding
	Quit     key.Binding
	Land     key.Binding
	Delete   key.Binding
	Test     key.Binding
	Toggle   key.Binding // toggle side-by-side / unified
	Space    key.Binding // mark reviewed
	NextFile key.Binding
	PrevFile key.Binding
	NextHunk key.Binding
	PrevHunk key.Binding
}

var keys = keyMap{
	Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("k/↑", "up")),
	Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("j/↓", "down")),
	Enter:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	Back:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Quit:     key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	Land:     key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "land")),
	Delete:   key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete worktree")),
	Test:     key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "run tests")),
	Toggle:   key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "toggle view")),
	Space:    key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "mark reviewed")),
	NextFile: key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "next file")),
	PrevFile: key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "prev file")),
	NextHunk: key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "next hunk")),
	PrevHunk: key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "prev hunk")),
}
```

- [ ] **Step 3: Verify it compiles**

Run: `go build ./...`
Expected: Compiles with no errors.

- [ ] **Step 4: Commit**

```
git add internal/tui/styles.go internal/tui/keys.go
git commit -m "feat: tui styles (catppuccin mocha) and key bindings"
```

---

### Task 5: TUI App Shell with Screen Routing

**Files:**
- Create: `internal/tui/app.go`
- Modify: `cmd/wtr/main.go`

- [ ] **Step 1: Create app.go with screen enum and root model**

```go
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

// App is the root Bubble Tea model.
type App struct {
	screen       screen
	repoDir      string
	width        int
	height       int

	// Screen models
	worktreeList worktreeListModel
	fileList     fileListModel
	diffView     diffViewModel

	// State passed between screens
	worktrees      []git.Worktree
	selectedWorktree int
	files          []git.FileDiff
	selectedFile   int
	reviewed       map[string]bool // key: "worktree:filename"
	sideBySide     bool
}

func NewApp(repoDir string) App {
	return App{
		repoDir:    repoDir,
		reviewed:   make(map[string]bool),
		sideBySide: true, // default to side-by-side
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
	case tea.KeyMsg:
		if msg.String() == "q" && a.screen == screenWorktreeList {
			return a, tea.Quit
		}
	}

	// Delegate to current screen
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

func (a App) loadWorktrees() tea.Cmd {
	return func() tea.Msg {
		wts, err := git.ListWorktrees(a.repoDir)
		if err != nil {
			return errMsg{err}
		}
		return worktreesLoadedMsg{wts}
	}
}
```

- [ ] **Step 2: Update main.go to use App**

Replace the placeholder model in `cmd/wtr/main.go`:
```go
package main

import (
	"fmt"
	"os"

	"github.com/byoungs/wtr/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Default to current directory
	repoDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Optional: accept repo path as argument
	if len(os.Args) > 1 {
		repoDir = os.Args[1]
	}

	app := tui.NewApp(repoDir)
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 3: Verify it compiles**

Run: `go build ./...`
Expected: Compiles (screens are stubs that return empty strings for now).

- [ ] **Step 4: Commit**

```
git add internal/tui/app.go cmd/wtr/main.go
git commit -m "feat: tui app shell with screen routing"
```

---

### Task 6: Worktree List Screen

**Files:**
- Create: `internal/tui/worktree_list.go`

- [ ] **Step 1: Implement worktree list model and view**

Create `internal/tui/worktree_list.go`:
```go
package tui

import (
	"fmt"
	"strings"

	"github.com/byoungs/wtr/internal/git"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type worktreeListModel struct {
	cursor int
}

func (a App) updateWorktreeList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case worktreesLoadedMsg:
		a.worktrees = msg.worktrees
		return a, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Up):
			if a.worktreeList.cursor > 0 {
				a.worktreeList.cursor--
			}
		case key.Matches(msg, keys.Down):
			if a.worktreeList.cursor < len(a.worktrees)-1 {
				a.worktreeList.cursor++
			}
		case key.Matches(msg, keys.Enter):
			if len(a.worktrees) > 0 {
				a.selectedWorktree = a.worktreeList.cursor
				a.screen = screenFileList
				return a, a.loadDiff()
			}
		case key.Matches(msg, keys.Back):
			return a, tea.Quit
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

	if len(a.worktrees) == 0 {
		b.WriteString(styleSubtle("  No worktrees found.\n"))
	}

	for i, wt := range a.worktrees {
		cursor := "  "
		if i == a.worktreeList.cursor {
			cursor = "→ "
		}

		name := wt.Branch
		stats := fmt.Sprintf("%d files  %s%d %s%d",
			wt.FilesChanged,
			"+", wt.Insertions,
			"-", wt.Deletions)

		line := fmt.Sprintf("%s%-40s %s", cursor, name, stats)
		if i == a.worktreeList.cursor {
			line = styleSelected.Width(a.width).Render(line)
		} else {
			line = styleNormal.Render(line)
		}
		b.WriteString(line + "\n")
	}

	b.WriteString("\n")
	b.WriteString(styleHelp.Render("  enter: review  l: land  d: delete  t: run tests  q: quit"))

	return b.String()
}

func styleSubtle(s string) string {
	return lipgloss.NewStyle().Foreground(colorSubtle).Render(s)
}
```

- [ ] **Step 2: Verify it compiles and renders**

Run: `go build ./...`
Expected: Compiles.

- [ ] **Step 3: Commit**

```
git add internal/tui/worktree_list.go
git commit -m "feat: worktree list screen with cursor navigation"
```

---

### Task 7: File List Screen

**Files:**
- Create: `internal/tui/file_list.go`

- [ ] **Step 1: Implement file list model and view**

Create `internal/tui/file_list.go`:
```go
package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/byoungs/wtr/internal/git"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type fileListModel struct {
	cursor int
}

func (a App) updateFileList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case diffLoadedMsg:
		a.files = msg.files
		a.fileList.cursor = 0
		return a, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Up):
			if a.fileList.cursor > 0 {
				a.fileList.cursor--
			}
		case key.Matches(msg, keys.Down):
			if a.fileList.cursor < len(a.files)-1 {
				a.fileList.cursor++
			}
		case key.Matches(msg, keys.Enter):
			if len(a.files) > 0 {
				a.selectedFile = a.fileList.cursor
				a.screen = screenDiffView
			}
		case key.Matches(msg, keys.Space):
			if len(a.files) > 0 {
				k := a.reviewKey(a.fileList.cursor)
				a.reviewed[k] = !a.reviewed[k]
			}
		case key.Matches(msg, keys.Back):
			a.screen = screenWorktreeList
		}
	}
	return a, nil
}

func (a App) reviewKey(fileIdx int) string {
	wt := a.worktrees[a.selectedWorktree]
	f := a.files[fileIdx]
	name := f.NewName
	if name == "" {
		name = f.OldName
	}
	return wt.Branch + ":" + name
}

func (a App) viewFileList() string {
	var b strings.Builder

	wt := a.worktrees[a.selectedWorktree]
	title := styleTitle.Width(a.width).Render(fmt.Sprintf("Files — %s", wt.Branch))
	b.WriteString(title + "\n\n")

	// Count reviewed
	reviewedCount := 0
	for i := range a.files {
		if a.reviewed[a.reviewKey(i)] {
			reviewedCount++
		}
	}
	b.WriteString(fmt.Sprintf("  %d/%d reviewed\n\n", reviewedCount, len(a.files)))

	for i, f := range a.files {
		cursor := "  "
		if i == a.fileList.cursor {
			cursor = "→ "
		}

		// Reviewed indicator
		check := "○"
		if a.reviewed[a.reviewKey(i)] {
			check = styleReviewed.Render("✓")
		}

		name := f.NewName
		if name == "" || name == "/dev/null" {
			name = f.OldName + " (deleted)"
		}
		if f.OldName == "/dev/null" {
			name = f.NewName + " (new)"
		}

		// Stats
		var added, removed int
		for _, h := range f.Hunks {
			for _, l := range h.Lines {
				switch l.Type {
				case git.LineAdded:
					added++
				case git.LineRemoved:
					removed++
				}
			}
		}
		stats := fmt.Sprintf("+%d -%d", added, removed)

		// Shorten path: show filename, dim the directory
		dir := filepath.Dir(name)
		base := filepath.Base(name)
		var displayName string
		if dir == "." {
			displayName = base
		} else {
			displayName = styleSubtle(dir+"/") + base
		}

		line := fmt.Sprintf("%s%s %s  %s", cursor, check, displayName, stats)
		if i == a.fileList.cursor {
			line = fmt.Sprintf("%s%s %-50s  %s", cursor, check, displayName, stats)
		}
		b.WriteString(line + "\n")
	}

	b.WriteString("\n")
	b.WriteString(styleHelp.Render("  enter: view diff  space: mark reviewed  esc: back"))

	return b.String()
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./...`
Expected: Compiles.

- [ ] **Step 3: Commit**

```
git add internal/tui/file_list.go
git commit -m "feat: file list screen with reviewed checkmarks"
```

---

### Task 8: Side-by-Side Diff View

**Files:**
- Create: `internal/tui/diff_view.go`

This is the most complex screen. Side-by-side rendering pairs old lines (left) with new lines (right), showing context lines on both sides and added/removed on their respective side.

- [ ] **Step 1: Implement side-by-side diff renderer**

Create `internal/tui/diff_view.go`:
```go
package tui

import (
	"fmt"
	"strings"

	"github.com/byoungs/wtr/internal/git"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type diffViewModel struct {
	scrollY int
}

func (a App) updateDiffView(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Up):
			if a.diffView.scrollY > 0 {
				a.diffView.scrollY--
			}
		case key.Matches(msg, keys.Down):
			a.diffView.scrollY++
		case key.Matches(msg, keys.NextFile):
			if a.selectedFile < len(a.files)-1 {
				a.selectedFile++
				a.diffView.scrollY = 0
			}
		case key.Matches(msg, keys.PrevFile):
			if a.selectedFile > 0 {
				a.selectedFile--
				a.diffView.scrollY = 0
			}
		case key.Matches(msg, keys.Space):
			k := a.reviewKey(a.selectedFile)
			a.reviewed[k] = !a.reviewed[k]
		case key.Matches(msg, keys.Toggle):
			a.sideBySide = !a.sideBySide
		case key.Matches(msg, keys.Back):
			a.screen = screenFileList
			a.diffView.scrollY = 0
		}
	}
	return a, nil
}

func (a App) viewDiffView() string {
	if a.selectedFile >= len(a.files) {
		return "No file selected"
	}

	f := a.files[a.selectedFile]
	var b strings.Builder

	// Title bar
	name := f.NewName
	if name == "" || name == "/dev/null" {
		name = f.OldName
	}
	reviewed := ""
	if a.reviewed[a.reviewKey(a.selectedFile)] {
		reviewed = " ✓ reviewed"
	}
	viewMode := "side-by-side"
	if !a.sideBySide {
		viewMode = "unified"
	}
	title := styleTitle.Width(a.width).Render(
		fmt.Sprintf("%s  [%d/%d]%s  (%s)",
			name, a.selectedFile+1, len(a.files), reviewed, viewMode))
	b.WriteString(title + "\n")

	if f.Binary {
		b.WriteString("\n  Binary file\n")
	} else if a.sideBySide {
		b.WriteString(a.renderSideBySide(f))
	} else {
		b.WriteString(a.renderUnified(f))
	}

	b.WriteString("\n")
	b.WriteString(styleHelp.Render(
		"  j/k: scroll  f/b: next/prev file  v: toggle view  space: reviewed  esc: back"))

	return b.String()
}

// sideBySideLine pairs an old line with a new line for rendering.
type sideBySideLine struct {
	oldNum     int    // 0 means blank
	oldContent string
	oldType    git.LineType
	newNum     int    // 0 means blank
	newContent string
	newType    git.LineType
}

func (a App) renderSideBySide(f git.FileDiff) string {
	halfWidth := (a.width - 3) / 2 // -3 for the center divider " │ "
	if halfWidth < 20 {
		halfWidth = 20
	}

	var allLines []sideBySideLine

	for _, hunk := range f.Hunks {
		// Add hunk header
		allLines = append(allLines, sideBySideLine{
			oldContent: hunk.Header,
			oldType:    git.LineContext,
			newContent: "",
			newType:    git.LineContext,
		})

		// Pair lines: context goes on both sides, removed on left, added on right
		oldNum := hunk.OldStart
		newNum := hunk.NewStart

		// Collect runs of removed/added for pairing
		i := 0
		for i < len(hunk.Lines) {
			line := hunk.Lines[i]
			switch line.Type {
			case git.LineContext:
				allLines = append(allLines, sideBySideLine{
					oldNum: oldNum, oldContent: line.Content, oldType: git.LineContext,
					newNum: newNum, newContent: line.Content, newType: git.LineContext,
				})
				oldNum++
				newNum++
				i++

			case git.LineRemoved:
				// Collect consecutive removed lines
				var removed []git.DiffLine
				for i < len(hunk.Lines) && hunk.Lines[i].Type == git.LineRemoved {
					removed = append(removed, hunk.Lines[i])
					i++
				}
				// Collect consecutive added lines that follow
				var added []git.DiffLine
				for i < len(hunk.Lines) && hunk.Lines[i].Type == git.LineAdded {
					added = append(added, hunk.Lines[i])
					i++
				}
				// Pair them
				maxLen := len(removed)
				if len(added) > maxLen {
					maxLen = len(added)
				}
				for j := 0; j < maxLen; j++ {
					sl := sideBySideLine{}
					if j < len(removed) {
						sl.oldNum = oldNum
						sl.oldContent = removed[j].Content
						sl.oldType = git.LineRemoved
						oldNum++
					}
					if j < len(added) {
						sl.newNum = newNum
						sl.newContent = added[j].Content
						sl.newType = git.LineAdded
						newNum++
					}
					allLines = append(allLines, sl)
				}

			case git.LineAdded:
				// Added without preceding removed
				allLines = append(allLines, sideBySideLine{
					newNum: newNum, newContent: line.Content, newType: git.LineAdded,
				})
				newNum++
				i++
			}
		}
	}

	// Apply scroll and viewport
	viewHeight := a.height - 4 // title + help
	if a.diffView.scrollY > len(allLines)-viewHeight {
		a.diffView.scrollY = max(0, len(allLines)-viewHeight)
	}
	end := a.diffView.scrollY + viewHeight
	if end > len(allLines) {
		end = len(allLines)
	}
	visible := allLines[a.diffView.scrollY:end]

	var b strings.Builder
	lineNumWidth := 4

	for _, sl := range visible {
		// Left side
		left := formatSideBySideHalf(sl.oldNum, sl.oldContent, sl.oldType, halfWidth, lineNumWidth)
		// Right side
		right := formatSideBySideHalf(sl.newNum, sl.newContent, sl.newType, halfWidth, lineNumWidth)

		divider := lipgloss.NewStyle().Foreground(colorSubtle).Render("│")
		b.WriteString(left + " " + divider + " " + right + "\n")
	}

	return b.String()
}

func formatSideBySideHalf(num int, content string, lineType git.LineType, width, lineNumWidth int) string {
	// Line number
	numStr := strings.Repeat(" ", lineNumWidth)
	if num > 0 {
		numStr = fmt.Sprintf("%*d", lineNumWidth, num)
	}
	numStyled := lipgloss.NewStyle().Foreground(colorSubtle).Render(numStr)

	// Truncate content to fit
	contentWidth := width - lineNumWidth - 2 // -2 for spacing
	if len(content) > contentWidth {
		content = content[:contentWidth-1] + "…"
	}
	// Pad to width
	content = content + strings.Repeat(" ", max(0, contentWidth-len(content)))

	// Style based on line type
	var styled string
	switch lineType {
	case git.LineAdded:
		styled = styleAdded.Render(content)
	case git.LineRemoved:
		styled = styleRemoved.Render(content)
	default:
		styled = styleContext.Render(content)
	}

	return numStyled + " " + styled
}

func (a App) renderUnified(f git.FileDiff) string {
	viewHeight := a.height - 4
	var allLines []string

	for _, hunk := range f.Hunks {
		allLines = append(allLines, styleHunkHeader.Render(hunk.Header))
		oldNum := hunk.OldStart
		newNum := hunk.NewStart

		for _, line := range hunk.Lines {
			var prefix, numStr string
			switch line.Type {
			case git.LineContext:
				prefix = " "
				numStr = fmt.Sprintf("%4d %4d", oldNum, newNum)
				oldNum++
				newNum++
			case git.LineRemoved:
				prefix = "-"
				numStr = fmt.Sprintf("%4d     ", oldNum)
				oldNum++
			case git.LineAdded:
				prefix = "+"
				numStr = fmt.Sprintf("     %4d", newNum)
				newNum++
			}

			numStyled := lipgloss.NewStyle().Foreground(colorSubtle).Render(numStr)
			var contentStyled string
			switch line.Type {
			case git.LineAdded:
				contentStyled = styleAdded.Render(prefix + line.Content)
			case git.LineRemoved:
				contentStyled = styleRemoved.Render(prefix + line.Content)
			default:
				contentStyled = styleContext.Render(prefix + line.Content)
			}
			allLines = append(allLines, numStyled+" "+contentStyled)
		}
	}

	// Apply scroll
	if a.diffView.scrollY > len(allLines)-viewHeight {
		a.diffView.scrollY = max(0, len(allLines)-viewHeight)
	}
	end := a.diffView.scrollY + viewHeight
	if end > len(allLines) {
		end = len(allLines)
	}
	visible := allLines[a.diffView.scrollY:end]

	return strings.Join(visible, "\n")
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./...`
Expected: Compiles.

- [ ] **Step 3: Manual smoke test**

Run `go run ./cmd/wtrr` from a repo that has worktrees. Navigate through all three screens. Verify:
- Worktree list shows entries
- File list shows changed files
- Side-by-side diff renders with colors
- `v` toggles to unified view
- `space` marks files reviewed
- `esc` navigates back

- [ ] **Step 4: Commit**

```
git add internal/tui/diff_view.go
git commit -m "feat: diff view with side-by-side and unified rendering"
```

---

### Task 9: Land Workflow

**Files:**
- Create: `internal/land/land.go`
- Modify: `internal/tui/worktree_list.go` (add land action)
- Modify: `internal/tui/app.go` (add land messages)

- [ ] **Step 1: Implement land.go**

The land workflow mirrors the `.zshrc` `land()` function but executed step-by-step with status reporting back to the TUI:

```go
package land

import (
	"fmt"
	"os/exec"
)

type Step struct {
	Name    string
	Command string
	Args    []string
}

// Steps returns the ordered steps for landing a branch.
// All commands run from repoDir (which should be on main).
func Steps(branch string) []Step {
	return []Step{
		{"merge", "git", []string{"merge", "--ff-only", branch}},
		{"test", "make", []string{"test"}},
		{"validate", "make", []string{"validate"}},
		{"push", "git", []string{"push"}},
	}
}

type StepResult struct {
	Step   Step
	Output string
	Err    error
}

// Run executes all land steps sequentially from repoDir.
// Calls onStep before each step. Returns on first failure.
func Run(repoDir string, branch string, onStep func(Step)) ([]StepResult, error) {
	var results []StepResult
	for _, step := range Steps(branch) {
		if onStep != nil {
			onStep(step)
		}
		cmd := exec.Command(step.Command, step.Args...)
		cmd.Dir = repoDir
		out, err := cmd.CombinedOutput()
		result := StepResult{Step: step, Output: string(out), Err: err}
		results = append(results, result)
		if err != nil {
			return results, fmt.Errorf("%s failed: %w", step.Name, err)
		}
	}
	return results, nil
}
```

- [ ] **Step 2: Add land messages and handling to app.go**

Add to `internal/tui/app.go`:
```go
// Add to messages section:
type landStartMsg struct{ branch string }
type landStepMsg struct{ step string }
type landDoneMsg struct{ results []land.StepResult }
type landErrMsg struct{ results []land.StepResult; err error }
```

- [ ] **Step 3: Wire land key binding in worktree_list.go**

Add a `case key.Matches(msg, keys.Land):` handler in `updateWorktreeList` that:
- Gets the selected worktree's branch
- Dispatches a `tea.Cmd` that calls `land.Run()` in a goroutine
- Shows a "Landing..." status in the view

- [ ] **Step 4: Verify it compiles**

Run: `go build ./...`
Expected: Compiles.

- [ ] **Step 5: Commit**

```
git add internal/land/land.go internal/tui/app.go internal/tui/worktree_list.go
git commit -m "feat: land workflow (ff-only merge + test + validate + push)"
```

---

### Task 10: Background Test/Validate

**Files:**
- Modify: `internal/tui/app.go` (add test status tracking)
- Modify: `internal/tui/worktree_list.go` (add test trigger and status display)

- [ ] **Step 1: Add test status types to app.go**

```go
type testStatus int

const (
	testNone    testStatus = iota
	testRunning
	testPassed
	testFailed
)

// Add to App struct:
// testStatus map[string]testStatus  // key: branch name
// testOutput map[string]string      // key: branch name, value: combined output

type testStartMsg struct{ branch string }
type testDoneMsg struct{ branch string; passed bool; output string }
```

- [ ] **Step 2: Add test trigger to worktree list**

Add `case key.Matches(msg, keys.Test):` handler that:
- Gets the selected worktree
- Runs `make test` in the worktree directory as a background `tea.Cmd`
- Updates `testStatus[branch]` to `testRunning`
- On completion, sends `testDoneMsg`

- [ ] **Step 3: Show test status in worktree list view**

Display the status indicator (✓/✗/⟳/—) next to each worktree in the list.

- [ ] **Step 4: Verify it compiles**

Run: `go build ./...`
Expected: Compiles.

- [ ] **Step 5: Manual test**

Run `wtr` from a repo with worktrees. Press `t` on a worktree. Verify:
- Spinner shows while tests run
- ✓ or ✗ appears when done

- [ ] **Step 6: Commit**

```
git add internal/tui/app.go internal/tui/worktree_list.go
git commit -m "feat: background test/validate with status indicators"
```

---

### Task 11: Delete Worktree

**Files:**
- Modify: `internal/tui/worktree_list.go` (add delete action)
- Modify: `internal/tui/app.go` (add delete messages)

- [ ] **Step 1: Add delete confirmation and handler**

Add `case key.Matches(msg, keys.Delete):` handler that:
- Shows a confirmation prompt ("Delete worktree-foo? y/n")
- On confirm: runs `git worktree remove <path>` and `git branch -D <branch>`
- Reloads the worktree list

- [ ] **Step 2: Verify it compiles and test manually**

Run: `go build ./...`
Expected: Compiles.

- [ ] **Step 3: Commit**

```
git add internal/tui/worktree_list.go internal/tui/app.go
git commit -m "feat: delete worktree with confirmation"
```

---

### Task 12: CLAUDE.md and Polish

**Files:**
- Create: `CLAUDE.md`

- [ ] **Step 1: Write CLAUDE.md**

```markdown
# wtr — Worktree Review TUI

A terminal UI for reviewing git worktree diffs, running tests, and landing branches.

## Build & Run

\`\`\`bash
make build     # Build binary to bin/wtr
make test      # Run tests
make install   # Install to ~/go/bin/wtr
\`\`\`

## Usage

Run from any git repo root (on main branch):
\`\`\`bash
wtr             # Review worktrees in current repo
wtr /path/to/repo  # Review worktrees in another repo
\`\`\`

## Key Bindings

### Worktree List
- j/k: navigate
- enter: review files
- t: run tests in background
- l: land (ff-only merge + test + validate + push)
- d: delete worktree
- q: quit

### File List
- j/k: navigate
- enter: view diff
- space: mark reviewed
- esc: back

### Diff View
- j/k: scroll
- f/b: next/prev file
- v: toggle side-by-side / unified
- space: mark reviewed
- esc: back to file list

## Tech Stack

Go + Bubble Tea (charmbracelet). No database. State is git worktrees.
```

- [ ] **Step 2: Commit**

```
git add CLAUDE.md
git commit -m "docs: CLAUDE.md with build, usage, and keybindings"
```

---

## Dependency Summary

Tasks 1-3 are foundational (scaffold, git library). Tasks 4-8 are the TUI (styles → shell → screens). Tasks 9-11 are actions (land, test, delete). Task 12 is docs.

```
Task 1 (scaffold) ──→ Task 2 (worktrees) ──→ Task 5 (worktree list screen)
                  ──→ Task 3 (diff parse) ──→ Task 7 (file list screen)
                  ──→ Task 4 (styles/keys) ──→ Task 8 (diff view screen)
                                           ──→ Task 6 (app shell) ──→ all screens
Task 9 (land) depends on Task 5
Task 10 (bg test) depends on Task 5
Task 11 (delete) depends on Task 5
Task 12 (docs) is independent
```
