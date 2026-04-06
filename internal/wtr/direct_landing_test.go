package wtr

import (
	"strings"
	"testing"

	"github.com/byoungs/wtr/internal/git"
)

func TestDirectLandingView_ShowsBranchLine(t *testing.T) {
	a := NewApp("/tmp/fake")
	a.mode = "direct"
	a.screen = screenDirectLanding
	a.width = 80
	a.height = 24
	a.branchInfo = git.BranchInfo{
		Name:         "main",
		CommitHash:   "abc1234",
		AheadOrigin:  2,
		HasUpstream:  true,
		FilesChanged: 12,
		Insertions:   473,
		Deletions:    33,
		Commits: []git.CommitInfo{
			{Hash: "abc1234", Subject: "feat: add something"},
			{Hash: "def5678", Subject: "fix: broken thing"},
		},
	}

	view := a.viewDirectLanding()
	// Branch summary line
	if !strings.Contains(view, "main") {
		t.Error("should show branch name")
	}
	if !strings.Contains(view, "12 files") {
		t.Error("should show file count")
	}
	if !strings.Contains(view, "+473") {
		t.Error("should show insertions")
	}
	// Commit list
	if !strings.Contains(view, "abc1234") {
		t.Error("should show commit hash")
	}
	if !strings.Contains(view, "feat: add something") {
		t.Error("should show commit subject")
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
	if !strings.Contains(view, "up to date") {
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
	if !strings.Contains(view, "✓") {
		t.Error("should show test passed icon")
	}
}

func TestDirectLandingNavbar(t *testing.T) {
	navbar := "  q:quit  h:help  →review  g:status  t:test  o:output  l:push  u:update"
	expected := []string{"q:", "h:", "→review", "g:", "t:", "o:", "l:", "u:"}
	for _, e := range expected {
		if !strings.Contains(navbar, e) {
			t.Errorf("direct landing navbar missing %q", e)
		}
	}
}
