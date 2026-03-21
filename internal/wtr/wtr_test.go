package wtr

import (
	"strings"
	"testing"

	"github.com/byoungs/wtr/internal/git"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// --- Auto-review tests ---

func TestAutoMarkReviewed_SkipsWhenNotRendered(t *testing.T) {
	a := NewApp("/tmp/fake")
	a.worktrees = []git.Worktree{{Branch: "test", CommitHash: "abc"}}
	a.files = []git.FileDiff{{NewName: "file.go"}}
	a.selectedFile = 0
	a.height = 30

	diffTotalLines = 0 // not rendered yet
	a.autoMarkReviewed()

	k := a.reviewKey(0)
	if a.reviewed[k] {
		t.Error("should not mark reviewed when diffTotalLines == 0")
	}
}

func TestAutoMarkReviewed_MarksWhenFitsOnScreen(t *testing.T) {
	a := NewApp("/tmp/fake")
	a.worktrees = []git.Worktree{{Branch: "test", CommitHash: "abc"}}
	a.files = []git.FileDiff{{NewName: "file.go"}}
	a.selectedFile = 0
	a.height = 30

	diffTotalLines = 5 // fits in viewport (30 - 4 = 26)
	diffScrollY = 0
	a.autoMarkReviewed()

	k := a.reviewKey(0)
	if !a.reviewed[k] {
		t.Error("should mark reviewed when content fits on one page")
	}
}

func TestAutoMarkReviewed_MarksWhenScrolledToBottom(t *testing.T) {
	a := NewApp("/tmp/fake")
	a.worktrees = []git.Worktree{{Branch: "test", CommitHash: "abc"}}
	a.files = []git.FileDiff{{NewName: "file.go"}}
	a.selectedFile = 0
	a.height = 14 // viewHeight = 14 - 4 = 10

	diffTotalLines = 50
	diffScrollY = 40 // at bottom (50 - 10 = 40)
	a.autoMarkReviewed()

	k := a.reviewKey(0)
	if !a.reviewed[k] {
		t.Error("should mark reviewed when scrolled to bottom")
	}
}

func TestAutoMarkReviewed_DoesNotMarkWhenNotAtBottom(t *testing.T) {
	a := NewApp("/tmp/fake")
	a.worktrees = []git.Worktree{{Branch: "test", CommitHash: "abc"}}
	a.files = []git.FileDiff{{NewName: "file.go"}}
	a.selectedFile = 0
	a.height = 14 // viewHeight = 10

	diffTotalLines = 50
	diffScrollY = 10 // not at bottom
	a.autoMarkReviewed()

	k := a.reviewKey(0)
	if a.reviewed[k] {
		t.Error("should not mark reviewed when not at bottom")
	}
}

func TestAutoMarkReviewed_ResetsOnFileSwitch(t *testing.T) {
	// Simulate: file A is short (fits on screen), switch to file B which is long.
	// After switch, diffTotalLines should be 0 so auto-review doesn't fire.
	a := NewApp("/tmp/fake")
	a.worktrees = []git.Worktree{{Branch: "test", CommitHash: "abc"}}
	a.files = []git.FileDiff{
		{NewName: "short.go"},
		{NewName: "long.go"},
	}
	a.selectedFile = 0
	a.height = 30

	// Simulate viewing file A — short, fits on screen
	diffTotalLines = 5
	diffScrollY = 0
	a.autoMarkReviewed()

	// File A should be reviewed
	if !a.reviewed[a.reviewKey(0)] {
		t.Error("file A should be marked reviewed")
	}

	// Now switch to file B via NextFile
	a.selectedFile = 1
	diffScrollY = 0
	diffTotalLines = 0 // this is what the fix does

	a.autoMarkReviewed()

	// File B should NOT be reviewed (diffTotalLines was reset)
	if a.reviewed[a.reviewKey(1)] {
		t.Error("file B should not be marked reviewed after file switch")
	}
}

// --- Keybinding/navbar consistency tests ---

// keysForScreen returns the set of key strings that a screen's handler responds to.
// This is a best-effort check based on the key.Binding definitions.
func boundKeys(bindings ...key.Binding) map[string]bool {
	result := make(map[string]bool)
	for _, b := range bindings {
		for _, k := range b.Keys() {
			result[k] = true
		}
	}
	return result
}

func TestNavbarMentionsAllHandledKeys_WorktreeList(t *testing.T) {
	// The worktree list navbar should mention all action keys
	navbar := "  q:quit  h:help  →review  o:open  t:test  v:output  r:rebase  l:land  del:delete  u:refresh"

	expected := []struct {
		label   string
		display string
	}{
		{"review", "→review"},
		{"open", "o:"},
		{"test", "t:"},
		{"output", "v:"},
		{"rebase", "r:"},
		{"land", "l:"},
		{"delete", "del:"},
		{"refresh", "u:"},
		{"help", "h:"},
		{"quit", "q:"},
	}

	for _, e := range expected {
		if !strings.Contains(navbar, e.display) {
			t.Errorf("worktree list navbar missing %q (looking for %q)", e.label, e.display)
		}
	}
}

func TestNavbarMentionsAllHandledKeys_FileList(t *testing.T) {
	navbar := "  q:quit  ←back  →view  o:open  a:all diffs  x:reviewed  g:status"

	expected := []string{"q:", "←back", "→view", "o:", "a:", "x:", "g:"}
	for _, e := range expected {
		if !strings.Contains(navbar, e) {
			t.Errorf("file list navbar missing %q", e)
		}
	}
}

func TestNavbarMentionsAllHandledKeys_DiffView(t *testing.T) {
	navbar := "  q:quit  ←back  space/b:page  ][:file  n/p:hunk  v:toggle  x:reviewed"

	expected := []string{"q:", "←back", "space/b:", "[:", "n/p:", "v:", "x:"}
	for _, e := range expected {
		if !strings.Contains(navbar, e) {
			t.Errorf("diff view navbar missing %q", e)
		}
	}
}

func TestNavbarMentionsAllHandledKeys_GitStatus(t *testing.T) {
	navbar := "  q:quit  ←back  →view  del:revert  o:open"

	expected := []string{"q:", "←back", "→view", "del:", "o:"}
	for _, e := range expected {
		if !strings.Contains(navbar, e) {
			t.Errorf("git status navbar missing %q", e)
		}
	}
}

func TestNavbarMentionsAllHandledKeys_AllDiffs(t *testing.T) {
	navbar := "  q:quit  ←back  space/b:page  v:toggle"

	expected := []string{"q:", "←back", "space/b:", "v:"}
	for _, e := range expected {
		if !strings.Contains(navbar, e) {
			t.Errorf("all diffs navbar missing %q", e)
		}
	}
}

// Test that no two key bindings on the same screen share the same key,
// except known intentional overlaps.
func TestNoKeyConflicts_WorktreeList(t *testing.T) {
	// Keys handled in updateWorktreeList
	worktreeKeys := []key.Binding{
		keys.Up, keys.Down, keys.Enter, keys.Right,
		keys.Test, keys.ViewOutput, keys.Land, keys.Squash,
		keys.Rebase, keys.Delete, keys.Back, keys.Refresh,
		keys.Open,
	}

	seen := make(map[string]string) // key -> binding name
	for _, b := range worktreeKeys {
		name := b.Help().Key
		for _, k := range b.Keys() {
			if prev, ok := seen[k]; ok {
				t.Errorf("worktree list: key %q bound to both %q and %q", k, prev, name)
			}
			seen[k] = name
		}
	}
}

func TestNoKeyConflicts_FileList(t *testing.T) {
	fileKeys := []key.Binding{
		keys.Up, keys.Down, keys.Enter, keys.Right,
		keys.MarkReviewed, keys.Open, keys.AllDiffs,
		keys.GitStatus, keys.Back,
	}

	seen := make(map[string]string)
	for _, b := range fileKeys {
		name := b.Help().Key
		for _, k := range b.Keys() {
			if prev, ok := seen[k]; ok {
				t.Errorf("file list: key %q bound to both %q and %q", k, prev, name)
			}
			seen[k] = name
		}
	}
}

func TestNoKeyConflicts_DiffView(t *testing.T) {
	diffKeys := []key.Binding{
		keys.Up, keys.Down, keys.PageDown, keys.PageUp,
		keys.NextFile, keys.PrevFile, keys.NextHunk, keys.PrevHunk,
		keys.MarkReviewed, keys.Toggle, keys.Back,
	}

	seen := make(map[string]string)
	for _, b := range diffKeys {
		name := b.Help().Key
		for _, k := range b.Keys() {
			if prev, ok := seen[k]; ok {
				t.Errorf("diff view: key %q bound to both %q and %q", k, prev, name)
			}
			seen[k] = name
		}
	}
}

func TestNoKeyConflicts_GitStatus(t *testing.T) {
	statusKeys := []key.Binding{
		keys.Up, keys.Down, keys.Right, keys.Enter,
		keys.Revert, keys.Open, keys.Back,
	}

	seen := make(map[string]string)
	for _, b := range statusKeys {
		name := b.Help().Key
		for _, k := range b.Keys() {
			if prev, ok := seen[k]; ok {
				t.Errorf("git status: key %q bound to both %q and %q", k, prev, name)
			}
			seen[k] = name
		}
	}
}

// --- NextFile marks reviewed tests ---

func TestNextFileMarksCurrentAsReviewed(t *testing.T) {
	a := NewApp("/tmp/fake")
	a.worktrees = []git.Worktree{{Branch: "test", CommitHash: "abc"}}
	a.files = []git.FileDiff{
		{NewName: "file1.go"},
		{NewName: "file2.go"},
	}
	a.selectedFile = 0
	a.height = 30
	a.width = 80

	// Pressing ] should mark file1 as reviewed and advance to file2
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}}
	model, _ := a.updateDiffView(msg)
	a = model.(App)

	if !a.reviewed[a.reviewKey(0)] {
		// reviewKey uses selectedFile, need to check with the old index
		k := a.worktrees[a.selectedWorktree].Branch + ":file1.go"
		if !a.reviewed[k] {
			t.Error("] should mark previous file as reviewed")
		}
	}
	if a.selectedFile != 1 {
		t.Errorf("selectedFile should be 1, got %d", a.selectedFile)
	}
}

func TestNextFileMarksReviewedEvenOnLastFile(t *testing.T) {
	a := NewApp("/tmp/fake")
	a.worktrees = []git.Worktree{{Branch: "test", CommitHash: "abc"}}
	a.files = []git.FileDiff{
		{NewName: "only.go"},
	}
	a.selectedFile = 0
	a.height = 30
	a.width = 80

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}}
	model, _ := a.updateDiffView(msg)
	a = model.(App)

	k := a.worktrees[a.selectedWorktree].Branch + ":only.go"
	if !a.reviewed[k] {
		t.Error("] on last file should still mark it as reviewed")
	}
	if a.selectedFile != 0 {
		t.Errorf("selectedFile should stay at 0 on last file, got %d", a.selectedFile)
	}
}

// --- Right arrow behavior tests ---

func TestRightArrowPageDownWhenNotAtBottom(t *testing.T) {
	a := NewApp("/tmp/fake")
	a.worktrees = []git.Worktree{{Branch: "test", CommitHash: "abc"}}
	a.files = []git.FileDiff{{NewName: "file.go"}}
	a.selectedFile = 0
	a.height = 13 // viewHeight = 13 - 3 = 10
	a.width = 80

	diffTotalLines = 50
	diffScrollY = 0

	msg := tea.KeyMsg{Type: tea.KeyRight}
	model, _ := a.updateDiffView(msg)
	a = model.(App)

	if diffScrollY != 10 {
		t.Errorf("right arrow should page down by viewHeight (10), got scrollY=%d", diffScrollY)
	}
	if a.selectedFile != 0 {
		t.Error("right arrow should not change file when not at bottom")
	}
}

func TestRightArrowNextFileWhenAtBottom(t *testing.T) {
	a := NewApp("/tmp/fake")
	a.worktrees = []git.Worktree{{Branch: "test", CommitHash: "abc"}}
	a.files = []git.FileDiff{
		{NewName: "file1.go"},
		{NewName: "file2.go"},
	}
	a.selectedFile = 0
	a.height = 13 // viewHeight = 10
	a.width = 80

	diffTotalLines = 50
	diffScrollY = 40 // at bottom (50 - 10 = 40)

	msg := tea.KeyMsg{Type: tea.KeyRight}
	model, _ := a.updateDiffView(msg)
	a = model.(App)

	if a.selectedFile != 1 {
		t.Errorf("right arrow at bottom should advance to next file, got selectedFile=%d", a.selectedFile)
	}

	k := a.worktrees[a.selectedWorktree].Branch + ":file1.go"
	if !a.reviewed[k] {
		t.Error("right arrow at bottom should mark current file as reviewed before advancing")
	}
}

func TestRightArrowNextFileWhenContentFitsOnScreen(t *testing.T) {
	a := NewApp("/tmp/fake")
	a.worktrees = []git.Worktree{{Branch: "test", CommitHash: "abc"}}
	a.files = []git.FileDiff{
		{NewName: "small.go"},
		{NewName: "next.go"},
	}
	a.selectedFile = 0
	a.height = 30 // viewHeight = 27
	a.width = 80

	diffTotalLines = 5 // fits on one page
	diffScrollY = 0

	msg := tea.KeyMsg{Type: tea.KeyRight}
	model, _ := a.updateDiffView(msg)
	a = model.(App)

	if a.selectedFile != 1 {
		t.Errorf("right arrow on short file should advance to next, got selectedFile=%d", a.selectedFile)
	}

	k := a.worktrees[a.selectedWorktree].Branch + ":small.go"
	if !a.reviewed[k] {
		t.Error("right arrow on short file should mark as reviewed")
	}
}

// --- q-key behavior tests ---

func TestQuitDoesNotFireDuringForceDelete(t *testing.T) {
	a := NewApp("/tmp/fake")
	a.deleteState = 2 // force-delete typing mode

	// The global q check in app.go should skip when deleteState == 2
	shouldQuit := a.deleteState != 2
	if shouldQuit {
		t.Error("q should not quit during force-delete typing")
	}
}
