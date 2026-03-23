package projectstate

import (
	"context"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// CaptureTypeCheck runs the detected typecheck tool and returns its state.
// Returns a state with Tool="none" if no tool is detected.
func CaptureTypeCheck(projectDir string, timeout time.Duration, maxErrors int) TypeCheckState {
	tool := DetectTypeChecker(projectDir)
	if tool == "none" {
		return TypeCheckState{Tool: "none"}
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var cmd *exec.Cmd
	switch tool {
	case "tsc":
		cmd = exec.CommandContext(ctx, "tsc", "--noEmit", "--incremental")
	case "go build":
		cmd = exec.CommandContext(ctx, "go", "build", "./...")
	}
	cmd.Dir = projectDir

	// Combined output — both tsc and go build write errors to stderr
	out, _ := cmd.CombinedOutput()
	durationMs := time.Since(start).Milliseconds()

	if ctx.Err() == context.DeadlineExceeded {
		return TypeCheckState{Tool: tool, TimedOut: true, DurationMs: durationMs}
	}

	var errors []string
	switch tool {
	case "tsc":
		errors = parseTscErrors(string(out))
	case "go build":
		errors = parseGoBuildErrors(string(out))
	}

	shown := errors
	if maxErrors > 0 && len(shown) > maxErrors {
		shown = shown[:maxErrors]
	}

	return TypeCheckState{
		Tool:       tool,
		ErrorCount: len(errors),
		Errors:     shown,
		DurationMs: durationMs,
		Note:       MonorepoNote(projectDir, tool),
	}
}

// CaptureCustomTypeCheck runs a user-configured typecheck command and interprets its exit code.
// exit 0 → ok (no errors), non-zero → errors. On failure, the last few output lines are
// included so Claude has context without requiring output format knowledge.
func CaptureCustomTypeCheck(projectDir, command string, timeout time.Duration, maxErrors int) TypeCheckState {
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
		return TypeCheckState{Tool: "custom", TimedOut: true, DurationMs: durationMs}
	}

	state := TypeCheckState{Tool: "custom", DurationMs: durationMs}
	if err != nil {
		// Include last N lines of output as errors for context
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		n := maxErrors
		if n <= 0 {
			n = 5
		}
		start := len(lines) - n
		if start < 0 {
			start = 0
		}
		for _, line := range lines[start:] {
			if line = strings.TrimSpace(line); line != "" {
				state.Errors = append(state.Errors, line)
			}
		}
		state.ErrorCount = len(state.Errors)
	}
	return state
}

// tscErrorRe matches: path/to/file.ts(line,col): error TSxxxx: message
var tscErrorRe = regexp.MustCompile(`^(.+?)\((\d+),\d+\): error TS\d+: (.+)$`)

// parseTscErrors extracts compact error lines from tsc output.
// Input format: "src/file.ts(23,5): error TS2322: Type 'x' is not assignable..."
// Output format: "L23 src/file.ts  Type 'x' is not assignable..."
func parseTscErrors(output string) []string {
	var errors []string
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if m := tscErrorRe.FindStringSubmatch(line); m != nil {
			errors = append(errors, "L"+m[2]+" "+m[1]+"  "+m[3])
		}
	}
	return errors
}

// goBuildErrorRe matches: ./path/to/file.go:line:col: message
var goBuildErrorRe = regexp.MustCompile(`^(\.?/?[^\s:]+\.go):(\d+):\d+: (.+)$`)

// parseGoBuildErrors extracts compact error lines from go build output.
// Input format: "./src/main.go:23:5: undefined: foo"
// Output format: "L23 src/main.go  undefined: foo"
func parseGoBuildErrors(output string) []string {
	var errors []string
	seen := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if m := goBuildErrorRe.FindStringSubmatch(line); m != nil {
			file := strings.TrimPrefix(m[1], "./")
			lineNum := m[2]
			msg := m[3]
			// Skip duplicate "too many errors" and note lines
			if strings.HasPrefix(msg, "too many errors") {
				continue
			}
			entry := "L" + lineNum + " " + file + "  " + msg
			if !seen[entry] {
				seen[entry] = true
				errors = append(errors, entry)
			}
		}
	}
	return errors
}

