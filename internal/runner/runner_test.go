package runner

import "testing"

func TestStatusToInt(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{StatusRunning, 1},
		{StatusPassed, 2},
		{StatusFailed, 3},
		{"", 0},
		{"unknown", 0},
	}
	for _, tt := range tests {
		got := StatusToInt(tt.input)
		if got != tt.want {
			t.Errorf("StatusToInt(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestReadLogMissing(t *testing.T) {
	got := ReadLog(t.TempDir(), "nonexistent")
	if got != "" {
		t.Errorf("ReadLog for missing file = %q, want empty", got)
	}
}

func TestReadStatusMissing(t *testing.T) {
	got := ReadStatus(t.TempDir(), "nonexistent")
	if got != "" {
		t.Errorf("ReadStatus for missing file = %q, want empty", got)
	}
}

func TestIsRunningMissing(t *testing.T) {
	got := IsRunning(t.TempDir(), "nonexistent")
	if got {
		t.Error("IsRunning for missing branch should be false")
	}
}
