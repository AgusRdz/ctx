package projectstate

import "time"

// ProjectState holds all captured project state at a point in time.
type ProjectState struct {
	CapturedAt int64
	Git        GitState
	TypeCheck  TypeCheckState
	Tests      TestState
}

// GitState holds captured git state.
type GitState struct {
	Branch      string
	DirtyFiles  []string // from git diff --name-only HEAD
	LastCommit  string   // "a3f2b1c Add auth middleware"
	AheadBehind string   // "↑2 ↓0", empty if no upstream
}

// TypeCheckState holds the result of a typecheck run.
type TypeCheckState struct {
	Tool       string   // "tsc" | "go build" | "none"
	ErrorCount int
	Errors     []string // first N errors, compact format
	DurationMs int64
	TimedOut   bool
}

// TestState holds the result of a test run.
type TestState struct {
	Tool        string   // "jest" | "vitest" | "go test" | "none"
	Pass        int
	Fail        int
	Skip        int
	FailedNames []string // first N failed test names
	DurationMs  int64
	TimedOut    bool
}

// CaptureOptions controls what gets captured.
type CaptureOptions struct {
	Git                  bool
	MaxDirtyFiles        int
	MaxErrors            int
	TypeCheck            bool
	TypeCheckTimeout     time.Duration
	Tests                bool
	TestsTimeout         time.Duration
	TestsMaxFailedNames  int
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
	// Tests captured in Phase 3
	return ps
}
