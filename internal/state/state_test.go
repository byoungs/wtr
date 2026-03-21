package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)

	s := State{
		TestStatus: map[string]int{"branch-a": 2, "branch-b": 3},
		TestedAt:   map[string]string{"branch-a": "abc123", "branch-b": "def456"},
		Reviewed:   map[string]bool{"branch-a:file.go": true},
		ReviewedAt: map[string]string{"branch-a": "abc123"},
	}

	Save(dir, s)

	loaded := Load(dir)
	if loaded.TestStatus["branch-a"] != 2 {
		t.Errorf("branch-a status = %d, want 2", loaded.TestStatus["branch-a"])
	}
	if loaded.TestStatus["branch-b"] != 3 {
		t.Errorf("branch-b status = %d, want 3", loaded.TestStatus["branch-b"])
	}
	if loaded.TestedAt["branch-a"] != "abc123" {
		t.Errorf("branch-a testedAt = %q, want abc123", loaded.TestedAt["branch-a"])
	}
	if !loaded.Reviewed["branch-a:file.go"] {
		t.Error("branch-a:file.go should be reviewed")
	}
	if loaded.ReviewedAt["branch-a"] != "abc123" {
		t.Errorf("branch-a reviewedAt = %q, want abc123", loaded.ReviewedAt["branch-a"])
	}
}

func TestLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	s := Load(dir)
	if s.TestStatus == nil {
		t.Error("TestStatus should be initialized, not nil")
	}
	if s.TestedAt == nil {
		t.Error("TestedAt should be initialized, not nil")
	}
	if s.Reviewed == nil {
		t.Error("Reviewed should be initialized, not nil")
	}
	if s.ReviewedAt == nil {
		t.Error("ReviewedAt should be initialized, not nil")
	}
}

func TestLoadCorruptJSON(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)
	os.WriteFile(filepath.Join(dir, ".git", "wtr-state.json"), []byte("{corrupt"), 0644)

	s := Load(dir)
	// Should return empty state, not panic or partial state
	if s.TestStatus == nil {
		t.Error("TestStatus should be initialized after corrupt load")
	}
	if len(s.TestStatus) != 0 {
		t.Error("TestStatus should be empty after corrupt load")
	}
	if s.Reviewed == nil {
		t.Error("Reviewed should be initialized after corrupt load")
	}
}
