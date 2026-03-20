package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Worktree represents a git worktree with diff stats relative to main.
type Worktree struct {
	Path         string // Absolute path to worktree directory
	Branch       string // Branch name (e.g. "worktree-auth-fix")
	CommitHash   string // HEAD commit short hash
	FilesChanged int    // Number of files changed vs main
	Insertions   int    // Lines added
	Deletions    int    // Lines deleted
}

// ListWorktrees runs `git worktree list --porcelain` in repoDir, parses the
// output, skips the main (first) worktree, and enriches each entry with diff
// stats against main.
func ListWorktrees(repoDir string) ([]Worktree, error) {
	cmd := exec.Command("git", "-C", repoDir, "worktree", "list", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git worktree list: %w", err)
	}

	worktrees, err := parsePorcelain(string(out))
	if err != nil {
		return nil, err
	}

	for i := range worktrees {
		enrichDiffStats(repoDir, &worktrees[i])
	}

	return worktrees, nil
}

// parsePorcelain parses the --porcelain output of `git worktree list`.
// It skips the first block (the main worktree).
func parsePorcelain(output string) ([]Worktree, error) {
	blocks := strings.Split(strings.TrimSpace(output), "\n\n")

	var worktrees []Worktree
	for i, block := range blocks {
		if i == 0 {
			// Skip the main worktree.
			continue
		}
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}

		var wt Worktree
		for _, line := range strings.Split(block, "\n") {
			switch {
			case strings.HasPrefix(line, "worktree "):
				wt.Path = strings.TrimPrefix(line, "worktree ")
			case strings.HasPrefix(line, "HEAD "):
				hash := strings.TrimPrefix(line, "HEAD ")
				if len(hash) > 7 {
					hash = hash[:7]
				}
				wt.CommitHash = hash
			case strings.HasPrefix(line, "branch "):
				ref := strings.TrimPrefix(line, "branch ")
				wt.Branch = filepath.Base(ref)
			}
		}
		if wt.Path != "" {
			worktrees = append(worktrees, wt)
		}
	}

	return worktrees, nil
}

// enrichDiffStats runs `git diff --stat main..HEAD` inside the worktree
// directory and parses the summary line for file count, insertions, and
// deletions.
func enrichDiffStats(repoDir string, wt *Worktree) {
	cmd := exec.Command("git", "-C", wt.Path, "diff", "--stat", "main..HEAD")
	out, err := cmd.Output()
	if err != nil {
		// Non-fatal: leave stats at zero.
		return
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 {
		return
	}

	// The summary line is always the last line, e.g.:
	//  3 files changed, 47 insertions(+), 12 deletions(-)
	summary := lines[len(lines)-1]
	parseSummaryLine(summary, wt)
}

// parseSummaryLine extracts FilesChanged, Insertions, and Deletions from a
// `git diff --stat` summary line.
func parseSummaryLine(line string, wt *Worktree) {
	line = strings.TrimSpace(line)
	parts := strings.Split(line, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		fields := strings.Fields(part)
		if len(fields) < 2 {
			continue
		}
		n, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		switch {
		case strings.HasPrefix(fields[1], "file"):
			wt.FilesChanged = n
		case strings.HasPrefix(fields[1], "insertion"):
			wt.Insertions = n
		case strings.HasPrefix(fields[1], "deletion"):
			wt.Deletions = n
		}
	}
}
