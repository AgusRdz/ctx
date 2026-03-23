package projectstate

import "time"

// ProjectState holds all captured project state at a point in time.
type ProjectState struct {
	CapturedAt int64          `json:"captured_at"`
	Git        GitState       `json:"git"`
	TypeCheck  TypeCheckState `json:"typecheck,omitempty"`
	Tests      TestState      `json:"tests,omitempty"`
}

// GitState holds captured git state.
type GitState struct {
	Branch      string   `json:"branch"`
	DirtyFiles  []string `json:"dirty_files"`  // from git diff --name-only HEAD
	LastCommit  string   `json:"last_commit"`  // "a3f2b1c Add auth middleware"
	AheadBehind string   `json:"ahead_behind"` // "↑2 ↓0", empty if no upstream
}

// TypeCheckState holds the result of a typecheck run.
type TypeCheckState struct {
	Tool       string   `json:"tool"`                 // "tsc" | "go build" | "none"
	ErrorCount int      `json:"error_count"`
	Errors     []string `json:"errors,omitempty"`     // first N errors, compact format
	DurationMs int64    `json:"duration_ms"`
	TimedOut   bool     `json:"timed_out,omitempty"`
	Note       string   `json:"note,omitempty"`       // e.g. monorepo warning
}

// TestState holds the result of a test run.
type TestState struct {
	Tool        string   `json:"tool"`                      // "jest" | "vitest" | "go test" | "none"
	Pass        int      `json:"pass"`
	Fail        int      `json:"fail"`
	Skip        int      `json:"skip"`
	FailedNames []string `json:"failed_names,omitempty"`    // first N failed test names
	DurationMs  int64    `json:"duration_ms"`
	TimedOut    bool     `json:"timed_out,omitempty"`
}

// CaptureOptions controls what gets captured.
type CaptureOptions struct {
	Git                 bool
	MaxDirtyFiles       int
	MaxErrors           int
	TypeCheck           bool
	TypeCheckTimeout    time.Duration
	Tests               bool
	TestsTimeout        time.Duration
	TestsMaxFailedNames int
	TestsCommand        string // custom command; overrides auto-detection when set
}

// Capture gathers project state for the given directory.
func Capture(projectDir string, opts CaptureOptions) ProjectState {
	ps := ProjectState{CapturedAt: time.Now().Unix()}
	if opts.Git {
		ps.Git = CaptureGit(projectDir, 10*time.Second)
	}
	if opts.TypeCheck {
		timeout := opts.TypeCheckTimeout
		if timeout <= 0 {
			timeout = 20 * time.Second
		}
		ps.TypeCheck = CaptureTypeCheck(projectDir, timeout, opts.MaxErrors)
	}
	if opts.Tests {
		timeout := opts.TestsTimeout
		if timeout <= 0 {
			timeout = 60 * time.Second
		}
		maxFailed := opts.TestsMaxFailedNames
		if maxFailed <= 0 {
			maxFailed = 5
		}
		if opts.TestsCommand != "" {
			ps.Tests = CaptureCustomTests(projectDir, opts.TestsCommand, timeout)
		} else {
			ps.Tests = CaptureTests(projectDir, timeout, maxFailed)
		}
	}
	return ps
}
