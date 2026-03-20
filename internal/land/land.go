package land

import (
	"fmt"
	"os/exec"
)

type Step struct {
	Name    string
	Command string
	Args    []string
}

func Steps(branch string) []Step {
	return []Step{
		{"merge", "git", []string{"merge", "--ff-only", branch}},
		{"test", "make", []string{"test"}},
		{"validate", "make", []string{"validate"}},
		{"push", "git", []string{"push"}},
	}
}

type StepResult struct {
	Step   Step
	Output string
	Err    error
}

func Run(repoDir string, branch string, onStep func(Step)) ([]StepResult, error) {
	var results []StepResult
	for _, step := range Steps(branch) {
		if onStep != nil {
			onStep(step)
		}
		cmd := exec.Command(step.Command, step.Args...)
		cmd.Dir = repoDir
		out, err := cmd.CombinedOutput()
		result := StepResult{Step: step, Output: string(out), Err: err}
		results = append(results, result)
		if err != nil {
			return results, fmt.Errorf("%s failed: %w", step.Name, err)
		}
	}
	return results, nil
}
