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
	Path           string       // Absolute path to worktree directory
	Branch         string       // Branch name (e.g. "worktree-auth-fix")
	CommitHash     string       // HEAD commit short hash
	FilesChanged   int          // Number of files changed vs main
	Insertions     int          // Lines added
	Deletions      int          // Lines deleted
	CommitsAhead   int          // Commits ahead of main
	CommitsBehind  int          // Commits behind main
	Uncommitted    int          // Number of uncommitted changes (git status)
	Commits        []CommitInfo // Commits ahead of main (newest first)
}

// CurrentHash returns the full HEAD hash for a worktree path.
func CurrentHash(worktreePath string) string {
	out, err := exec.Command("git", "-C", worktreePath, "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
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
		enrichCommitCounts(&worktrees[i])
		enrichUncommitted(&worktrees[i])
		worktrees[i].Commits = listCommits(worktrees[i].Path, "main..HEAD")
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
				wt.CommitHash = strings.TrimSpace(strings.TrimPrefix(line, "HEAD "))
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

// enrichCommitCounts runs git rev-list to determine how many commits
// the worktree is ahead of and behind main.
func enrichCommitCounts(wt *Worktree) {
	// Commits ahead: commits in HEAD that aren't in main
	if out, err := exec.Command("git", "-C", wt.Path, "rev-list", "--count", "main..HEAD").Output(); err == nil {
		wt.CommitsAhead, _ = strconv.Atoi(strings.TrimSpace(string(out)))
	}
	// Commits behind: commits in main that aren't in HEAD
	if out, err := exec.Command("git", "-C", wt.Path, "rev-list", "--count", "HEAD..main").Output(); err == nil {
		wt.CommitsBehind, _ = strconv.Atoi(strings.TrimSpace(string(out)))
	}
}

// UncommittedCount returns the number of uncommitted changes in a directory.
func UncommittedCount(path string) int {
	out, err := exec.Command("git", "-C", path, "status", "--short").Output()
	if err != nil {
		return 0
	}
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return 0
	}
	return len(strings.Split(trimmed, "\n"))
}

func enrichUncommitted(wt *Worktree) {
	wt.Uncommitted = UncommittedCount(wt.Path)
}

// SquashOntoMain rebases on main then squashes all commits into one.
// Returns the combined output and any error.
func SquashOntoMain(worktreePath string) (string, error) {
	var allOutput strings.Builder

	// First, rebase onto main to catch up
	cmd := exec.Command("git", "-C", worktreePath, "rebase", "main")
	out, err := cmd.CombinedOutput()
	allOutput.Write(out)
	if err != nil {
		exec.Command("git", "-C", worktreePath, "rebase", "--abort").Run()
		return allOutput.String(), fmt.Errorf("rebase: %s", strings.TrimSpace(string(out)))
	}

	// Get the commit message from the first commit after main
	cmd = exec.Command("git", "-C", worktreePath, "log", "--format=%B", "main..HEAD")
	msgOut, err := cmd.Output()
	if err != nil {
		return allOutput.String(), fmt.Errorf("reading commit messages: %s", strings.TrimSpace(string(out)))
	}
	msg := strings.TrimSpace(string(msgOut))
	if msg == "" {
		msg = "squashed commits"
	}

	// Soft reset to main (keeps changes staged)
	cmd = exec.Command("git", "-C", worktreePath, "reset", "--soft", "main")
	out, err = cmd.CombinedOutput()
	allOutput.Write(out)
	if err != nil {
		return allOutput.String(), fmt.Errorf("reset --soft: %s", strings.TrimSpace(string(out)))
	}

	// Commit with the combined message
	cmd = exec.Command("git", "-C", worktreePath, "commit", "-m", msg)
	out, err = cmd.CombinedOutput()
	allOutput.Write(out)
	if err != nil {
		return allOutput.String(), fmt.Errorf("commit: %s", strings.TrimSpace(string(out)))
	}

	return allOutput.String(), nil
}

// RebaseOnMain runs git rebase main in the worktree.
// Returns the combined output and any error. If there are conflicts,
// the error message will indicate that.
func RebaseOnMain(worktreePath string) (string, error) {
	cmd := exec.Command("git", "-C", worktreePath, "rebase", "main")
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Abort the rebase so the worktree isn't left in a broken state
		exec.Command("git", "-C", worktreePath, "rebase", "--abort").Run()
		return string(out), fmt.Errorf("rebase failed (aborted): %w\n%s", err, out)
	}
	return string(out), nil
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

// ChangedFilesBetween returns the list of files that changed between two commits.
// Returns nil if the diff can't be computed (e.g. old hash was garbage collected).
func ChangedFilesBetween(repoPath, oldHash, newHash string) []string {
	out, err := exec.Command("git", "-C", repoPath, "diff", "--name-only", oldHash, newHash).Output()
	if err != nil {
		return nil
	}
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}
