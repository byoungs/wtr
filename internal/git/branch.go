package git

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// CommitInfo represents a single commit.
type CommitInfo struct {
	Hash    string // short hash
	Subject string // first line of commit message
}

// BranchInfo holds the current branch's relationship to its upstream.
type BranchInfo struct {
	Name         string       // e.g. "main"
	CommitHash   string       // HEAD short hash
	AheadOrigin  int          // commits ahead of origin
	BehindOrigin int          // commits behind origin
	HasUpstream  bool         // whether origin/<branch> exists
	Uncommitted  int          // uncommitted changes count
	Commits      []CommitInfo // unpushed commits (newest first)
	FilesChanged int          // files changed vs origin
	Insertions   int          // lines added vs origin
	Deletions    int          // lines removed vs origin
}

// GetBranchInfo returns info about the current branch relative to its origin.
func GetBranchInfo(repoDir string) (BranchInfo, error) {
	var info BranchInfo

	// Current branch name
	out, err := exec.Command("git", "-C", repoDir, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return info, fmt.Errorf("get branch: %w", err)
	}
	info.Name = strings.TrimSpace(string(out))

	// HEAD hash
	out, err = exec.Command("git", "-C", repoDir, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return info, fmt.Errorf("get hash: %w", err)
	}
	info.CommitHash = strings.TrimSpace(string(out))

	// Check if upstream exists
	upstream := "origin/" + info.Name
	err = exec.Command("git", "-C", repoDir, "rev-parse", "--verify", upstream).Run()
	if err != nil {
		// No upstream — everything local is "ahead"
		info.HasUpstream = false
		out, _ := exec.Command("git", "-C", repoDir, "rev-list", "--count", "HEAD").Output()
		info.AheadOrigin, _ = strconv.Atoi(strings.TrimSpace(string(out)))
		info.Commits = listCommits(repoDir, "HEAD")
		info.Uncommitted = UncommittedCount(repoDir)
		return info, nil
	}
	info.HasUpstream = true

	// Ahead/behind
	out, err = exec.Command("git", "-C", repoDir, "rev-list", "--count", upstream+"..HEAD").Output()
	if err == nil {
		info.AheadOrigin, _ = strconv.Atoi(strings.TrimSpace(string(out)))
	}
	out, err = exec.Command("git", "-C", repoDir, "rev-list", "--count", "HEAD.."+upstream).Output()
	if err == nil {
		info.BehindOrigin, _ = strconv.Atoi(strings.TrimSpace(string(out)))
	}

	info.Commits = listCommits(repoDir, upstream+"..HEAD")
	enrichBranchDiffStats(repoDir, upstream+"..HEAD", &info)
	info.Uncommitted = UncommittedCount(repoDir)
	return info, nil
}

// enrichBranchDiffStats runs `git diff --stat` and parses the summary line.
func enrichBranchDiffStats(repoDir string, rangeSpec string, info *BranchInfo) {
	out, err := exec.Command("git", "-C", repoDir, "diff", "--stat", rangeSpec).Output()
	if err != nil {
		return
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 {
		return
	}
	summary := lines[len(lines)-1]
	for _, part := range strings.Split(summary, ",") {
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
			info.FilesChanged = n
		case strings.HasPrefix(fields[1], "insertion"):
			info.Insertions = n
		case strings.HasPrefix(fields[1], "deletion"):
			info.Deletions = n
		}
	}
}

// listCommits returns commits from a git log range (e.g. "origin/main..HEAD" or "HEAD").
func listCommits(repoDir string, rangeSpec string) []CommitInfo {
	out, err := exec.Command("git", "-C", repoDir, "log", "--oneline", rangeSpec).Output()
	if err != nil {
		return nil
	}
	var commits []CommitInfo
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		c := CommitInfo{Hash: parts[0]}
		if len(parts) > 1 {
			c.Subject = parts[1]
		}
		commits = append(commits, c)
	}
	return commits
}

// UpstreamRef returns "origin/<branch>" for the current branch.
// Returns empty string if no upstream is configured.
func UpstreamRef(repoDir string) string {
	out, err := exec.Command("git", "-C", repoDir, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return ""
	}
	branch := strings.TrimSpace(string(out))
	ref := "origin/" + branch
	if exec.Command("git", "-C", repoDir, "rev-parse", "--verify", ref).Run() != nil {
		return ""
	}
	return ref
}
