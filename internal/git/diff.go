package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type FileDiff struct {
	OldName string
	NewName string
	Hunks   []Hunk
	Binary  bool
}

type Hunk struct {
	OldStart int
	OldLines int
	NewStart int
	NewLines int
	Header   string
	Lines    []DiffLine
}

type DiffLine struct {
	Type    LineType
	Content string // Line content WITHOUT the +/- prefix
}

type LineType int

const (
	LineContext  LineType = iota
	LineAdded
	LineRemoved
)

// ParseDiff parses unified diff output into structured FileDiff entries.
func ParseDiff(raw string) ([]FileDiff, error) {
	var files []FileDiff
	var currentFile *FileDiff
	var currentHunk *Hunk

	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			// Finalize previous hunk and file
			if currentHunk != nil && currentFile != nil {
				currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
				currentHunk = nil
			}
			if currentFile != nil {
				files = append(files, *currentFile)
			}
			currentFile = &FileDiff{}

		case strings.HasPrefix(line, "--- "):
			if currentFile == nil {
				continue
			}
			name := strings.TrimPrefix(line, "--- ")
			if name == "/dev/null" {
				currentFile.OldName = "/dev/null"
			} else {
				currentFile.OldName = strings.TrimPrefix(name, "a/")
			}

		case strings.HasPrefix(line, "+++ "):
			if currentFile == nil {
				continue
			}
			name := strings.TrimPrefix(line, "+++ ")
			if name == "/dev/null" {
				currentFile.NewName = "/dev/null"
			} else {
				currentFile.NewName = strings.TrimPrefix(name, "b/")
			}

		case strings.HasPrefix(line, "@@ "):
			if currentFile == nil {
				continue
			}
			// Finalize previous hunk
			if currentHunk != nil {
				currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
			}
			hunk := Hunk{Header: line}
			_, err := fmt.Sscanf(line, "@@ -%d,%d +%d,%d @@", &hunk.OldStart, &hunk.OldLines, &hunk.NewStart, &hunk.NewLines)
			if err != nil {
				// Try without counts (e.g. @@ -0,0 +1 @@ or @@ -1 +1,0 @@)
				fmt.Sscanf(line, "@@ -%d +%d @@", &hunk.OldStart, &hunk.NewStart)
			}
			currentHunk = &hunk

		case strings.HasPrefix(line, "Binary files"):
			if currentFile != nil {
				currentFile.Binary = true
			}

		case len(line) > 0 && line[0] == '+':
			if currentHunk != nil {
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{
					Type:    LineAdded,
					Content: line[1:],
				})
			}

		case len(line) > 0 && line[0] == '-':
			if currentHunk != nil {
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{
					Type:    LineRemoved,
					Content: line[1:],
				})
			}

		case len(line) > 0 && line[0] == ' ':
			if currentHunk != nil {
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{
					Type:    LineContext,
					Content: line[1:],
				})
			}

		default:
			// Skip unrecognized lines (index lines, mode lines, empty lines, etc.)
		}
	}

	// Finalize the last hunk and file
	if currentHunk != nil && currentFile != nil {
		currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
	}
	if currentFile != nil {
		files = append(files, *currentFile)
	}

	return files, nil
}

// GetDiff runs `git -C <path> diff main..HEAD` and parses the output.
func GetDiff(worktreePath string) ([]FileDiff, error) {
	cmd := exec.Command("git", "-C", worktreePath, "diff", "main..HEAD")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}
	return ParseDiff(string(out))
}

// GetWorkingDiff returns the working tree diff for a specific file (unstaged changes).
// For untracked files, returns a synthetic diff showing the full file as added.
func GetWorkingDiff(worktreePath, filePath string) ([]FileDiff, error) {
	// Try normal diff first (for tracked modified files)
	cmd := exec.Command("git", "-C", worktreePath, "diff", "--", filePath)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}
	if len(out) > 0 {
		return ParseDiff(string(out))
	}

	// No diff output — might be untracked. Show full file as added.
	content, err := os.ReadFile(filepath.Join(worktreePath, filePath))
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(content), "\n")
	var diffLines []DiffLine
	for _, l := range lines {
		diffLines = append(diffLines, DiffLine{Type: LineAdded, Content: l})
	}
	return []FileDiff{{
		OldName: "/dev/null",
		NewName: filePath,
		Hunks: []Hunk{{
			NewStart: 1,
			NewLines: len(lines),
			Header:   fmt.Sprintf("@@ -0,0 +1,%d @@ (new file)", len(lines)),
			Lines:    diffLines,
		}},
	}}, nil
}
