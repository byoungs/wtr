package land

import "testing"

func TestSteps(t *testing.T) {
	steps := Steps("worktree-test")
	if len(steps) != 4 {
		t.Fatalf("expected 4 steps, got %d", len(steps))
	}

	expected := []struct {
		name     string
		command  string
		optional bool
	}{
		{"merge", "git", false},
		{"test", "make", false},
		{"validate", "make", true},
		{"push", "git", false},
	}

	for i, exp := range expected {
		if steps[i].Name != exp.name {
			t.Errorf("step %d name = %q, want %q", i, steps[i].Name, exp.name)
		}
		if steps[i].Command != exp.command {
			t.Errorf("step %d command = %q, want %q", i, steps[i].Command, exp.command)
		}
		if steps[i].Optional != exp.optional {
			t.Errorf("step %d optional = %v, want %v", i, steps[i].Optional, exp.optional)
		}
	}

	// Verify merge uses --ff-only with the branch name
	if steps[0].Args[0] != "merge" || steps[0].Args[1] != "--ff-only" || steps[0].Args[2] != "worktree-test" {
		t.Errorf("merge args = %v, want [merge --ff-only worktree-test]", steps[0].Args)
	}
}

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
