package land

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Step struct {
	Name     string
	Command  string
	Args     []string
	Optional bool
}

func Steps(branch string) []Step {
	return []Step{
		{"merge", "git", []string{"merge", "--ff-only", branch}, false},
		{"test", "make", []string{"test"}, false},
		{"validate", "make", []string{"validate"}, true},
		{"push", "git", []string{"push"}, false},
	}
}

// DirectSteps returns the steps for pushing the current branch to origin.
// No merge step — we're already on the branch.
func DirectSteps() []Step {
	return []Step{
		{"test", "make", []string{"test"}, false},
		{"validate", "make", []string{"validate"}, true},
		{"push", "git", []string{"push"}, false},
	}
}

// HasMakeTarget checks if the Makefile in repoDir has the given target.
func HasMakeTarget(repoDir, target string) bool {
	cmd := exec.Command("make", "-n", target)
	cmd.Dir = repoDir
	return cmd.Run() == nil
}

// FilterMissingTargets removes make steps whose targets don't exist in the Makefile.
// Returns the filtered steps and the names of removed steps.
func FilterMissingTargets(repoDir string, steps []Step) ([]Step, []string) {
	var filtered []Step
	var missing []string
	for _, s := range steps {
		if s.Command == "make" && !HasMakeTarget(repoDir, s.Args[0]) {
			missing = append(missing, s.Name)
			continue
		}
		filtered = append(filtered, s)
	}
	return filtered, missing
}

type StepResult struct {
	Step   Step
	Output string
	Err    error
}

// Run executes land steps sequentially from repoDir.
// Output is appended to logPath if provided (empty string to skip).
func Run(repoDir string, steps []Step, logPath string, onStep func(Step)) ([]StepResult, error) {
	var logFile *os.File
	if logPath != "" {
		os.MkdirAll(filepath.Dir(logPath), 0755)
		var err error
		logFile, err = os.Create(logPath)
		if err == nil {
			defer logFile.Close()
		}
	}

	writeLog := func(format string, args ...any) {
		if logFile != nil {
			fmt.Fprintf(logFile, format, args...)
		}
	}

	var results []StepResult
	for _, step := range steps {
		if onStep != nil {
			onStep(step)
		}
		writeLog("==> %s: %s %s\n", step.Name, step.Command, fmt.Sprint(step.Args))

		cmd := exec.Command(step.Command, step.Args...)
		cmd.Dir = repoDir

		// Stream output to log file in real-time (not buffered)
		var out []byte
		var err error
		if logFile != nil {
			cmd.Stdout = logFile
			cmd.Stderr = logFile
			err = cmd.Run()
		} else {
			out, err = cmd.CombinedOutput()
		}

		if err != nil {
			writeLog("==> %s FAILED: %v\n", step.Name, err)
		} else {
			writeLog("==> %s OK\n", step.Name)
		}

		result := StepResult{Step: step, Output: string(out), Err: err}
		results = append(results, result)
		if err != nil {
			if step.Optional {
				writeLog("==> %s skipped (optional)\n", step.Name)
				continue
			}
			// When streaming to log, output is in the file, not in out
			return results, fmt.Errorf("%s failed: %w", step.Name, err)
		}
	}
	writeLog("\n==> Landing complete.\n")
	return results, nil
}
