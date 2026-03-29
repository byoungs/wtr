package wtr

import "testing"

func TestIsFailLine(t *testing.T) {
	shouldFail := []string{
		"--- FAIL: TestSomething (0.01s)",
		"FAIL\tgithub.com/foo/bar",
		"FAIL",
		"panic: runtime error: index out of range",
		"main.go:42:5: undefined: foo",
		"Error Trace:    /path/to/file_test.go:42",
		"Error:          Not equal",
		"make: *** [validate] Error 2",
		"make[1]: *** [test] Error 1",
		"exit status 1",
		"  FAIL\tgithub.com/foo/bar",
		"WARNING: DATA RACE",
		"Expected: 42",
		"Actual:   99",
		"error_test.go:42: assertion failed",
	}
	for _, line := range shouldFail {
		if !isFailLine(line) {
			t.Errorf("expected fail for %q", line)
		}
	}

	shouldNotFail := []string{
		"=== RUN   TestErrorHandler",
		"--- PASS: TestErrorHandler (0.00s)",
		"    error_test.go: test setup",
		"testing error handling correctly",
		"ok  \tgithub.com/foo/bar\t0.005s",
		"PASS",
		"handling errors in production",
		"TestParseError",
		"error",
	}
	for _, line := range shouldNotFail {
		if isFailLine(line) {
			t.Errorf("expected not fail for %q", line)
		}
	}
}

func TestIsPassLine(t *testing.T) {
	shouldPass := []string{
		"--- PASS: TestSomething (0.01s)",
		"ok  \tgithub.com/foo/bar\t0.005s",
		"PASS",
		"  --- PASS: TestNested (0.00s)",
	}
	for _, line := range shouldPass {
		if !isPassLine(line) {
			t.Errorf("expected pass for %q", line)
		}
	}

	shouldNotPass := []string{
		"=== RUN   TestPassParser",
		"testing password validation",
		"--- FAIL: TestSomething (0.01s)",
		"the token passed through",
	}
	for _, line := range shouldNotPass {
		if isPassLine(line) {
			t.Errorf("expected not pass for %q", line)
		}
	}
}

func TestIsCompilerError(t *testing.T) {
	shouldMatch := []string{
		"main.go:42:5: undefined: foo",
		"./internal/wtr/app.go:10:2: imported and not used",
	}
	for _, line := range shouldMatch {
		if !isCompilerError(line) {
			t.Errorf("expected compiler error for %q", line)
		}
	}

	shouldNotMatch := []string{
		"error_test.go: test setup",
		"reading config.go settings",
		"foo.go",
	}
	for _, line := range shouldNotMatch {
		if isCompilerError(line) {
			t.Errorf("expected not compiler error for %q", line)
		}
	}
}
