package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// mustRun executes a git command in dir and fails the test on error.
func mustRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	fullArgs := append([]string{"-C", dir}, args...)
	cmd := exec.Command("git", fullArgs...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@test.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestListWorktrees(t *testing.T) {
	// Create a temp directory for the main repo.
	repoDir := t.TempDir()

	// Initialise with an explicit branch name so we always have "main".
	mustRun(t, repoDir, "init", "-b", "main")
	mustRun(t, repoDir, "config", "user.email", "test@test.com")
	mustRun(t, repoDir, "config", "user.name", "Test")

	// Create an initial commit so that HEAD is valid.
	readmeFile := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(readmeFile, []byte("# repo\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mustRun(t, repoDir, "add", ".")
	mustRun(t, repoDir, "commit", "-m", "initial commit")

	// Create a worktree on a new branch.
	wtDir := filepath.Join(t.TempDir(), "wt-test")
	mustRun(t, repoDir, "worktree", "add", "-b", "worktree-test", wtDir)

	// Add a file and commit inside the worktree.
	newFile := filepath.Join(wtDir, "feature.txt")
	if err := os.WriteFile(newFile, []byte("hello worktree\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mustRun(t, wtDir, "add", ".")
	mustRun(t, wtDir, "commit", "-m", "add feature file")

	// Exercise the function under test.
	worktrees, err := ListWorktrees(repoDir)
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}

	if len(worktrees) < 1 {
		t.Fatalf("expected at least 1 worktree, got 0")
	}

	wt := worktrees[0]

	if wt.Branch != "worktree-test" {
		t.Errorf("Branch = %q, want %q", wt.Branch, "worktree-test")
	}

	if wt.FilesChanged != 1 {
		t.Errorf("FilesChanged = %d, want 1", wt.FilesChanged)
	}
}
