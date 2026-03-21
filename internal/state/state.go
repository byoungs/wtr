package state

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const stateFile = ".git/wtr-state.json"

// State holds persistent data that survives restarts.
type State struct {
	// TestStatus per branch: 0=none, 2=passed, 3=failed (1=running is transient)
	TestStatus map[string]int `json:"test_status,omitempty"`
	// TestedAt per branch: commit hash when test was run
	TestedAt map[string]string `json:"tested_at,omitempty"`
	// Reviewed per "branch:filename": true if reviewed
	Reviewed map[string]bool `json:"reviewed,omitempty"`
	// ReviewedAt per branch: commit hash when reviews were done
	ReviewedAt map[string]string `json:"reviewed_at,omitempty"`
}

// Load reads state from .git/wtr-state.json in the repo directory.
// Returns empty state if file doesn't exist.
func Load(repoDir string) State {
	s := State{
		TestStatus: make(map[string]int),
		TestedAt:   make(map[string]string),
		Reviewed:   make(map[string]bool),
		ReviewedAt: make(map[string]string),
	}
	data, err := os.ReadFile(filepath.Join(repoDir, stateFile))
	if err != nil {
		return s
	}
	if err := json.Unmarshal(data, &s); err != nil {
		// Corrupt file — return empty state rather than partial
		return State{
			TestStatus: make(map[string]int),
			TestedAt:   make(map[string]string),
			Reviewed:   make(map[string]bool),
			ReviewedAt: make(map[string]string),
		}
	}
	if s.TestStatus == nil {
		s.TestStatus = make(map[string]int)
	}
	if s.TestedAt == nil {
		s.TestedAt = make(map[string]string)
	}
	if s.Reviewed == nil {
		s.Reviewed = make(map[string]bool)
	}
	if s.ReviewedAt == nil {
		s.ReviewedAt = make(map[string]string)
	}
	return s
}

// Save writes state to .git/wtr-state.json in the repo directory.
func Save(repoDir string, s State) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(filepath.Join(repoDir, stateFile), data, 0644)
}
