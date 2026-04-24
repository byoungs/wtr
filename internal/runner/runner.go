// Package runner manages background test/validate processes that survive wtr exit.
// Output goes to .git/wtr/<branch>.log, exit status to .git/wtr/<branch>.status.
package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const wtrDir = ".git/wtr"

// Status values written to .status files
const (
	StatusRunning = "running"
	StatusPassed  = "passed"
	StatusFailed  = "failed"
)

// LogPath returns the path to the log file for a branch.
func LogPath(repoDir, branch string) string {
	return filepath.Join(repoDir, wtrDir, branch+".log")
}

func logPath(repoDir, branch string) string {
	return LogPath(repoDir, branch)
}

func statusPath(repoDir, branch string) string {
	return filepath.Join(repoDir, wtrDir, branch+".status")
}

func pidPath(repoDir, branch string) string {
	return filepath.Join(repoDir, wtrDir, branch+".pid")
}

// Start launches `make validate` in the worktree as a detached process.
// Output streams to .git/wtr/<branch>.log. Status is written on completion.
// The process survives wtr exit.
func Start(repoDir, worktreePath, branch string) error {
	dir := filepath.Join(repoDir, wtrDir)
	os.MkdirAll(dir, 0755)

	logFile := logPath(repoDir, branch)
	statusFile := statusPath(repoDir, branch)
	pidFile := pidPath(repoDir, branch)

	// Clear previous output
	os.Remove(logFile)
	os.WriteFile(statusFile, []byte(StatusRunning), 0644)

	// Run make validate if the target exists, skip gracefully if not.
	script := fmt.Sprintf(
		`if make -n validate >/dev/null 2>&1; then make validate >>%q 2>&1; if [ $? -eq 0 ]; then echo passed > %q; else echo failed > %q; fi; else echo "No validate target found, skipping." >>%q; echo passed > %q; fi`,
		logFile, statusFile, statusFile, logFile, statusFile,
	)

	cmd := exec.Command("sh", "-c", script)
	cmd.Dir = worktreePath
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	// Create the log file so it exists immediately for tailing
	os.WriteFile(logFile, []byte{}, 0644)

	if err := cmd.Start(); err != nil {
		debugLog(repoDir, "[start] branch=%s err=%v", branch, err)
		return fmt.Errorf("starting validate: %w", err)
	}

	pid := cmd.Process.Pid
	debugLog(repoDir, "[start] branch=%s pid=%d dir=%s", branch, pid, worktreePath)

	// Write PID so we can check if still running
	os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644)

	// Release the process — it's fully detached
	cmd.Process.Release()

	return nil
}

// ReadLog returns the current contents of the log file.
func ReadLog(repoDir, branch string) string {
	data, err := os.ReadFile(logPath(repoDir, branch))
	if err != nil {
		return ""
	}
	return string(data)
}

// ReadStatus returns the current status: "running", "passed", "failed", or "".
func ReadStatus(repoDir, branch string) string {
	data, err := os.ReadFile(statusPath(repoDir, branch))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// IsRunning checks if a process is still running for the given branch.
func IsRunning(repoDir, branch string) bool {
	status := ReadStatus(repoDir, branch)
	if status != StatusRunning {
		debugLog(repoDir, "[isrunning] branch=%s status=%q (not running)", branch, status)
		return false
	}
	// Double-check the PID is alive
	data, err := os.ReadFile(pidPath(repoDir, branch))
	if err != nil {
		debugLog(repoDir, "[isrunning] branch=%s no pid file: %v", branch, err)
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		debugLog(repoDir, "[isrunning] branch=%s bad pid: %v", branch, err)
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		debugLog(repoDir, "[isrunning] branch=%s FindProcess(%d): %v", branch, pid, err)
		return false
	}
	// Signal 0 checks if process exists
	err = proc.Signal(syscall.Signal(0))
	alive := err == nil
	debugLog(repoDir, "[isrunning] branch=%s pid=%d alive=%v signal_err=%v", branch, pid, alive, err)
	return alive
}

// StatusToInt converts file status to the int used by the app.
func StatusToInt(status string) int {
	switch status {
	case StatusRunning:
		return 1
	case StatusPassed:
		return 2
	case StatusFailed:
		return 3
	default:
		return 0
	}
}

// Clean removes output files for a branch.
func Clean(repoDir, branch string) {
	os.Remove(logPath(repoDir, branch))
	os.Remove(statusPath(repoDir, branch))
	os.Remove(pidPath(repoDir, branch))
}

// --- make dev (long-running server) ---

func devLogPath(repoDir, branch string) string {
	return filepath.Join(repoDir, wtrDir, branch+".dev.log")
}

func devPidPath(repoDir, branch string) string {
	return filepath.Join(repoDir, wtrDir, branch+".dev.pid")
}

// DevLogPath returns the path to the make-dev log for a branch.
func DevLogPath(repoDir, branch string) string {
	return devLogPath(repoDir, branch)
}

// StartDev launches `make dev` in the worktree as a detached process group.
// Output streams to .git/wtr/<branch>.dev.log. No status file — the process is
// expected to run until killed via KillDev.
func StartDev(repoDir, worktreePath, branch string) error {
	dir := filepath.Join(repoDir, wtrDir)
	os.MkdirAll(dir, 0755)

	logFile := devLogPath(repoDir, branch)
	pidFile := devPidPath(repoDir, branch)

	os.Remove(logFile)
	os.WriteFile(logFile, []byte{}, 0644)

	script := fmt.Sprintf(
		`if make -n dev >/dev/null 2>&1; then exec make dev >>%q 2>&1; else echo "No dev target found." >>%q; exit 1; fi`,
		logFile, logFile,
	)

	cmd := exec.Command("sh", "-c", script)
	cmd.Dir = worktreePath
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		debugLog(repoDir, "[dev start] branch=%s err=%v", branch, err)
		return fmt.Errorf("starting make dev: %w", err)
	}

	pid := cmd.Process.Pid
	debugLog(repoDir, "[dev start] branch=%s pid=%d dir=%s", branch, pid, worktreePath)
	os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644)
	cmd.Process.Release()
	return nil
}

// ReadDevLog returns the current contents of the dev log.
func ReadDevLog(repoDir, branch string) string {
	data, err := os.ReadFile(devLogPath(repoDir, branch))
	if err != nil {
		return ""
	}
	return string(data)
}

// DevIsRunning returns true if a make-dev process is alive for the branch.
func DevIsRunning(repoDir, branch string) bool {
	data, err := os.ReadFile(devPidPath(repoDir, branch))
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

// KillDev terminates the make-dev process group for the branch. Safe to call
// when nothing is running.
func KillDev(repoDir, branch string) {
	data, err := os.ReadFile(devPidPath(repoDir, branch))
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		os.Remove(devPidPath(repoDir, branch))
		return
	}
	// Kill the whole process group (Setsid made pid == pgid).
	if err := syscall.Kill(-pid, syscall.SIGTERM); err != nil {
		debugLog(repoDir, "[dev kill] branch=%s pgid=%d SIGTERM err=%v", branch, pid, err)
	}
	// Give it a moment, then force-kill if still alive.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if syscall.Kill(-pid, syscall.Signal(0)) != nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if syscall.Kill(-pid, syscall.Signal(0)) == nil {
		syscall.Kill(-pid, syscall.SIGKILL)
	}
	os.Remove(devPidPath(repoDir, branch))
	debugLog(repoDir, "[dev kill] branch=%s pgid=%d done", branch, pid)
}

// debugLog appends a timestamped line to .git/wtr/debug.log.
func debugLog(repoDir, format string, args ...any) {
	dir := filepath.Join(repoDir, wtrDir)
	os.MkdirAll(dir, 0755)
	f, err := os.OpenFile(filepath.Join(dir, "debug.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(f, "%s %s\n", time.Now().Format("15:04:05.000"), msg)
}
