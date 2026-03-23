package projectstate

import (
	"context"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// CaptureTests runs the detected test runner and returns its state.
// Returns a state with Tool="none" if no runner is detected.
func CaptureTests(projectDir string, timeout time.Duration, maxFailedNames int) TestState {
	tool := DetectTestRunner(projectDir)
	if tool == "none" {
		return TestState{Tool: "none"}
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var cmd *exec.Cmd
	switch tool {
	case "jest":
		cmd = exec.CommandContext(ctx, "jest", "--passWithNoTests", "--silent")
	case "vitest":
		cmd = exec.CommandContext(ctx, "vitest", "run", "--reporter=verbose")
	case "go test":
		cmd = exec.CommandContext(ctx, "go", "test", "./...", "-short")
	}
	cmd.Dir = projectDir

	out, _ := cmd.CombinedOutput()
	durationMs := time.Since(start).Milliseconds()

	if ctx.Err() == context.DeadlineExceeded {
		return TestState{Tool: tool, TimedOut: true, DurationMs: durationMs}
	}

	var state TestState
	switch tool {
	case "jest":
		state = parseJestOutput(string(out))
	case "vitest":
		state = parseVitestOutput(string(out))
	case "go test":
		state = parseGoTestOutput(string(out))
	}
	state.Tool = tool
	state.DurationMs = durationMs

	if maxFailedNames > 0 && len(state.FailedNames) > maxFailedNames {
		state.FailedNames = state.FailedNames[:maxFailedNames]
	}
	return state
}

// CaptureCustomTests runs a user-configured command and interprets its exit code.
// exit 0 → passed, non-zero → failed. On failure, the last few output lines are
// included so Claude has context without requiring output format knowledge.
func CaptureCustomTests(projectDir, command string, timeout time.Duration) TestState {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/c", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}
	cmd.Dir = projectDir

	out, err := cmd.CombinedOutput()
	durationMs := time.Since(start).Milliseconds()

	if ctx.Err() == context.DeadlineExceeded {
		return TestState{Tool: "custom", TimedOut: true, DurationMs: durationMs}
	}

	state := TestState{Tool: "custom", DurationMs: durationMs}
	if err == nil {
		state.Pass = 1 // exit 0 — treat as a single "passed" result
	} else {
		state.Fail = 1
		// Include last 3 lines of output as failed names for context
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		start := len(lines) - 3
		if start < 0 {
			start = 0
		}
		for _, line := range lines[start:] {
			if line = strings.TrimSpace(line); line != "" {
				state.FailedNames = append(state.FailedNames, line)
			}
		}
	}
	return state
}

// Jest output parsing
// Summary line: "Tests: 2 failed, 47 passed, 49 total"
// or:           "Tests: 47 passed, 47 total"
// Failed test:  "  ● AuthService › should validate token expiry"

var jestSummaryRe = regexp.MustCompile(`Tests:\s+(.*?)\d+ total`)
var jestCountRe = regexp.MustCompile(`(\d+)\s+(failed|passed|skipped|pending)`)
var jestFailedTestRe = regexp.MustCompile(`^\s+● (.+)$`)

func parseJestOutput(output string) TestState {
	var state TestState
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "Tests:") && strings.Contains(line, "total") {
			for _, m := range jestCountRe.FindAllStringSubmatch(line, -1) {
				n, _ := strconv.Atoi(m[1])
				switch m[2] {
				case "passed":
					state.Pass = n
				case "failed":
					state.Fail = n
				case "skipped", "pending":
					state.Skip += n
				}
			}
		}
		if m := jestFailedTestRe.FindStringSubmatch(line); m != nil {
			name := strings.TrimSpace(m[1])
			if name != "" {
				state.FailedNames = append(state.FailedNames, name)
			}
		}
	}
	return state
}

// Vitest output parsing
// Summary line: "Test Files  1 failed | 2 passed (3)"
//            or "Tests       2 failed | 47 passed | 1 skipped (50)"
// Failed test:  " ✗ AuthService > should validate token expiry"
// or:           " × AuthService > should validate token expiry"

var vitestTestCountRe = regexp.MustCompile(`^Tests\s+(.+)$`)
var vitestCountRe = regexp.MustCompile(`(\d+)\s+(failed|passed|skipped)`)
var vitestFailedRe = regexp.MustCompile(`^\s+[✗×x]\s+(.+)$`)

func parseVitestOutput(output string) TestState {
	var state TestState
	for _, line := range strings.Split(output, "\n") {
		if vitestTestCountRe.MatchString(line) {
			for _, m := range vitestCountRe.FindAllStringSubmatch(line, -1) {
				n, _ := strconv.Atoi(m[1])
				switch m[2] {
				case "passed":
					state.Pass = n
				case "failed":
					state.Fail = n
				case "skipped":
					state.Skip = n
				}
			}
		}
		if m := vitestFailedRe.FindStringSubmatch(line); m != nil {
			name := strings.TrimSpace(m[1])
			if name != "" {
				state.FailedNames = append(state.FailedNames, name)
			}
		}
	}
	return state
}

// Go test output parsing
// Pass:  "ok  	github.com/foo/bar	0.123s"
// Fail:  "FAIL	github.com/foo/bar	0.456s"
// Skip:  "?   	github.com/foo/bar	[no test files]"
// Failed test: "--- FAIL: TestFoo (0.00s)"

var goTestOkRe = regexp.MustCompile(`^ok\s+\S+`)
var goTestFailRe = regexp.MustCompile(`^FAIL\s+\S+`)
var goTestSkipRe = regexp.MustCompile(`^\?\s+\S+\s+\[no test files\]`)
var goTestFailedRe = regexp.MustCompile(`^--- FAIL: (\S+)`)

func parseGoTestOutput(output string) TestState {
	var state TestState
	for _, line := range strings.Split(output, "\n") {
		switch {
		case goTestOkRe.MatchString(line):
			state.Pass++
		case goTestFailRe.MatchString(line):
			state.Fail++
		case goTestSkipRe.MatchString(line):
			state.Skip++
		}
		if m := goTestFailedRe.FindStringSubmatch(line); m != nil {
			state.FailedNames = append(state.FailedNames, m[1])
		}
	}
	return state
}
