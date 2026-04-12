package wtr

import (
	"strings"
	"testing"

	"github.com/byoungs/wtr/internal/git"
)

// --- wrapContent helper tests ---

func TestWrapContent_FitsInWidth(t *testing.T) {
	cases := []struct {
		name    string
		content string
		width   int
	}{
		{"short", "hello", 10},
		{"exact", "abcd", 4},
		{"empty", "", 10},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := wrapContent(c.content, c.width, true)
			if len(got) != 1 || got[0] != c.content {
				t.Errorf("expected single chunk %q, got %v", c.content, got)
			}
		})
	}
}

func TestWrapContent_WrapsLongContent(t *testing.T) {
	got := wrapContent("abcdefghij", 4, true)
	want := []string{"abcd", "efgh", "ij"}
	if len(got) != len(want) {
		t.Fatalf("expected %d chunks, got %d: %v", len(want), len(got), got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("chunk %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestWrapContent_Disabled(t *testing.T) {
	long := strings.Repeat("x", 100)
	got := wrapContent(long, 10, false)
	if len(got) != 1 || got[0] != long {
		t.Errorf("wrap disabled: should return single chunk, got %d chunks", len(got))
	}
}

func TestWrapContent_WidthZeroOrNegative(t *testing.T) {
	long := strings.Repeat("x", 100)
	got := wrapContent(long, 0, true)
	if len(got) != 1 || got[0] != long {
		t.Errorf("width=0: should return single chunk, got %d chunks", len(got))
	}
	got = wrapContent(long, -5, true)
	if len(got) != 1 || got[0] != long {
		t.Errorf("width<0: should return single chunk, got %d chunks", len(got))
	}
}

func TestWrapContent_UnicodeRunesCountedByRune(t *testing.T) {
	// "héllo" is 5 runes but 6 bytes — wrap should use rune count, not byte count
	got := wrapContent("héllo", 3, true)
	want := []string{"hél", "lo"}
	if len(got) != len(want) {
		t.Fatalf("expected %d chunks, got %d: %v", len(want), len(got), got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("chunk %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

// --- renderUnified wrapping integration tests ---

func TestRenderUnified_WrapsLongLinesAtWideWidth(t *testing.T) {
	a := NewApp("/tmp/fake")
	a.width = 80 // contentWidth = 70
	a.height = 50

	// 200-char line + 1-char prefix = 201 chars, wraps into ceil(201/70) = 3 chunks
	longLine := strings.Repeat("a", 200)
	file := git.FileDiff{
		NewName: "file.go",
		Hunks: []git.Hunk{{
			Header:   "@@ -1,1 +1,1 @@",
			OldStart: 1, OldLines: 1, NewStart: 1, NewLines: 1,
			Lines: []git.DiffLine{
				{Type: git.LineAdded, Content: longLine},
			},
		}},
	}

	a.renderUnified(file)

	// 1 hunk header + 3 wrapped visual lines = 4
	if diffTotalLines != 4 {
		t.Errorf("expected 4 visual lines (1 header + 3 wrapped), got %d", diffTotalLines)
	}
}

// --- viewAllDiffs wrapping tests ---

func TestViewAllDiffs_WrapsLongLinesAtWideWidth(t *testing.T) {
	allDiffsScrollY = 0
	a := NewApp("/tmp/fake")
	a.width = 80
	a.height = 100
	a.worktrees = []git.Worktree{{Branch: "test"}}
	a.selectedWorktree = 0

	longLine := strings.Repeat("a", 200)
	a.files = []git.FileDiff{{
		NewName: "file.go",
		Hunks: []git.Hunk{{
			Header:   "@@ -1,1 +1,1 @@",
			OldStart: 1, OldLines: 1, NewStart: 1, NewLines: 1,
			Lines: []git.DiffLine{
				{Type: git.LineAdded, Content: longLine},
			},
		}},
	}}

	view := a.viewAllDiffs()

	// With wrapping on (contentWidth = 70), no visual chunk contains 100 'a's.
	if strings.Contains(view, strings.Repeat("a", 100)) {
		t.Error("expected long line to be wrapped at width=80, but found 100 contiguous 'a' chars")
	}
	// Sanity: the content should still appear in wrapped form
	if !strings.Contains(view, strings.Repeat("a", 50)) {
		t.Error("expected wrapped line to still contain the content")
	}
}

func TestViewAllDiffs_NoWrapBelowMinWidth(t *testing.T) {
	allDiffsScrollY = 0
	a := NewApp("/tmp/fake")
	a.width = 70 // below minWrapWidth
	a.height = 100
	a.worktrees = []git.Worktree{{Branch: "test"}}
	a.selectedWorktree = 0

	longLine := strings.Repeat("a", 200)
	a.files = []git.FileDiff{{
		NewName: "file.go",
		Hunks: []git.Hunk{{
			Header:   "@@ -1,1 +1,1 @@",
			OldStart: 1, OldLines: 1, NewStart: 1, NewLines: 1,
			Lines: []git.DiffLine{
				{Type: git.LineAdded, Content: longLine},
			},
		}},
	}}

	view := a.viewAllDiffs()

	// Below minWrapWidth, long line should not wrap — 200 'a' chars appear
	// contiguously on a single styled line.
	if !strings.Contains(view, strings.Repeat("a", 200)) {
		t.Error("expected no wrap below minWrapWidth=80 — 200 contiguous 'a' chars should appear")
	}
}

func TestRenderUnified_HunkPositionsTrackWrappedLines(t *testing.T) {
	a := NewApp("/tmp/fake")
	a.width = 80
	a.height = 50

	longLine := strings.Repeat("a", 200) // wraps into 3 chunks at contentWidth=70
	file := git.FileDiff{
		NewName: "file.go",
		Hunks: []git.Hunk{
			{
				Header:   "@@ -1,1 +1,1 @@",
				OldStart: 1, OldLines: 1, NewStart: 1, NewLines: 1,
				Lines: []git.DiffLine{
					{Type: git.LineAdded, Content: longLine},
				},
			},
			{
				Header:   "@@ -10,1 +10,1 @@",
				OldStart: 10, OldLines: 1, NewStart: 10, NewLines: 1,
				Lines: []git.DiffLine{
					{Type: git.LineAdded, Content: "short"},
				},
			},
		},
	}

	a.renderUnified(file)

	// Layout:
	//   0: hunk1 header
	//   1: wrapped chunk 1
	//   2: wrapped chunk 2
	//   3: wrapped chunk 3
	//   4: hunk2 header
	//   5: short line
	if len(diffHunkPositions) != 2 {
		t.Fatalf("expected 2 hunk positions, got %d", len(diffHunkPositions))
	}
	if diffHunkPositions[0] != 0 {
		t.Errorf("hunk 1 should be at visual line 0, got %d", diffHunkPositions[0])
	}
	if diffHunkPositions[1] != 4 {
		t.Errorf("hunk 2 should be at visual line 4 (after wrapped chunks), got %d", diffHunkPositions[1])
	}
}
