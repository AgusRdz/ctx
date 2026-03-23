package projectstate

import (
	"strings"
	"testing"
)

func TestFormat_GitOnly_Clean(t *testing.T) {
	ps := ProjectState{
		Git: GitState{
			Branch:     "main",
			LastCommit: "a3f2b1c Add auth middleware",
		},
	}
	out := Format(ps, 10, 5)
	if !strings.Contains(out, "## Project State (at compaction)") {
		t.Error("missing section header")
	}
	if !strings.Contains(out, "Git:  main") {
		t.Error("missing branch")
	}
	if !strings.Contains(out, "last: a3f2b1c") {
		t.Error("missing last commit")
	}
	if strings.Contains(out, "Dirty:") {
		t.Error("should not show Dirty line when no dirty files")
	}
}

func TestFormat_GitOnly_Dirty(t *testing.T) {
	ps := ProjectState{
		Git: GitState{
			Branch:      "feat/auth",
			AheadBehind: "↑2 ↓0",
			DirtyFiles:  []string{"src/auth.ts", "src/session.ts", "src/types.ts"},
		},
	}
	out := Format(ps, 2, 5)
	if !strings.Contains(out, "↑2 ↓0") {
		t.Error("missing ahead/behind")
	}
	if !strings.Contains(out, "Dirty: src/auth.ts, src/session.ts (+1 more)") {
		t.Errorf("unexpected dirty line in:\n%s", out)
	}
}

func TestFormat_NotGitRepo(t *testing.T) {
	ps := ProjectState{Git: GitState{}}
	out := Format(ps, 10, 5)
	if !strings.Contains(out, "not a git repository") {
		t.Errorf("expected non-git message, got:\n%s", out)
	}
}

func TestFormat_TypeCheckOk(t *testing.T) {
	ps := ProjectState{
		Git: GitState{Branch: "main"},
		TypeCheck: TypeCheckState{
			Tool:       "tsc",
			ErrorCount: 0,
		},
	}
	out := Format(ps, 10, 5)
	if !strings.Contains(out, "TypeCheck: tsc — ok") {
		t.Errorf("unexpected typecheck output:\n%s", out)
	}
}

func TestFormat_TypeCheckErrors(t *testing.T) {
	ps := ProjectState{
		Git: GitState{Branch: "main"},
		TypeCheck: TypeCheckState{
			Tool:       "tsc",
			ErrorCount: 3,
			Errors:     []string{"err1", "err2", "err3", "err4"},
		},
	}
	out := Format(ps, 10, 2)
	if !strings.Contains(out, "TypeCheck: tsc — 3 error(s)") {
		t.Errorf("missing error count:\n%s", out)
	}
	if !strings.Contains(out, "err1") || !strings.Contains(out, "err2") {
		t.Error("missing first errors")
	}
	if strings.Contains(out, "err3") {
		t.Error("should not show errors beyond maxErrors")
	}
}

func TestFormat_TypeCheckTimedOut(t *testing.T) {
	ps := ProjectState{
		Git:       GitState{Branch: "main"},
		TypeCheck: TypeCheckState{Tool: "go build", TimedOut: true},
	}
	out := Format(ps, 10, 5)
	if !strings.Contains(out, "timed out") {
		t.Errorf("missing timed out:\n%s", out)
	}
}

func TestFormat_Tests(t *testing.T) {
	ps := ProjectState{
		Git: GitState{Branch: "main"},
		Tests: TestState{
			Tool:        "jest",
			Pass:        47,
			Fail:        2,
			Skip:        1,
			FailedNames: []string{"Auth > should validate token", "Session > concurrent writes"},
		},
	}
	out := Format(ps, 10, 5)
	if !strings.Contains(out, "Tests: jest — 47 pass | 2 fail | 1 skip") {
		t.Errorf("unexpected tests line:\n%s", out)
	}
	if !strings.Contains(out, "Auth > should validate token") {
		t.Error("missing failed test name")
	}
}

func TestFormatAheadBehind(t *testing.T) {
	cases := []struct {
		ahead, behind, want string
	}{
		{"2", "0", "↑2 ↓0"},
		{"0", "3", "↑0 ↓3"},
		{"0", "0", ""},
		{"", "", ""},
		{"1", "", "↑1 ↓0"},
		{"", "1", "↑0 ↓1"},
	}
	for _, c := range cases {
		got := formatAheadBehind(c.ahead, c.behind)
		if got != c.want {
			t.Errorf("formatAheadBehind(%q, %q) = %q, want %q", c.ahead, c.behind, got, c.want)
		}
	}
}
