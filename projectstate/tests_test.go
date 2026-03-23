package projectstate

import (
	"testing"
	"time"
)

func TestParseJestOutput_PassOnly(t *testing.T) {
	input := `
PASS src/auth.test.ts
Tests: 47 passed, 47 total
`
	state := parseJestOutput(input)
	if state.Pass != 47 || state.Fail != 0 || state.Skip != 0 {
		t.Errorf("want 47/0/0, got %d/%d/%d", state.Pass, state.Fail, state.Skip)
	}
}

func TestParseJestOutput_WithFailures(t *testing.T) {
	input := `
  ● AuthService › should validate token expiry
  ● SessionStore › should handle concurrent writes

Tests: 2 failed, 45 passed, 47 total
`
	state := parseJestOutput(input)
	if state.Pass != 45 || state.Fail != 2 {
		t.Errorf("want 45 pass / 2 fail, got %d/%d", state.Pass, state.Fail)
	}
	if len(state.FailedNames) != 2 {
		t.Fatalf("want 2 failed names, got %d: %v", len(state.FailedNames), state.FailedNames)
	}
	if state.FailedNames[0] != "AuthService › should validate token expiry" {
		t.Errorf("unexpected failed name: %q", state.FailedNames[0])
	}
}

func TestParseJestOutput_WithSkipped(t *testing.T) {
	input := `Tests: 1 skipped, 46 passed, 47 total`
	state := parseJestOutput(input)
	if state.Skip != 1 || state.Pass != 46 {
		t.Errorf("want skip=1 pass=46, got skip=%d pass=%d", state.Skip, state.Pass)
	}
}

func TestParseVitestOutput_PassOnly(t *testing.T) {
	input := `
 ✓ src/auth.test.ts (3)
Tests  47 passed (47)
`
	state := parseVitestOutput(input)
	if state.Pass != 47 || state.Fail != 0 {
		t.Errorf("want 47/0, got %d/%d", state.Pass, state.Fail)
	}
}

func TestParseVitestOutput_WithFailures(t *testing.T) {
	input := `
 × AuthService > should validate token expiry
 × SessionStore > should handle concurrent writes
Tests  2 failed | 45 passed (47)
`
	state := parseVitestOutput(input)
	if state.Pass != 45 || state.Fail != 2 {
		t.Errorf("want 45/2, got %d/%d", state.Pass, state.Fail)
	}
	if len(state.FailedNames) != 2 {
		t.Fatalf("want 2 failed names, got %d", len(state.FailedNames))
	}
}

func TestParseGoTestOutput_AllPass(t *testing.T) {
	input := `ok  	github.com/foo/bar	0.123s
ok  	github.com/foo/baz	0.456s`
	state := parseGoTestOutput(input)
	if state.Pass != 2 || state.Fail != 0 {
		t.Errorf("want 2/0, got %d/%d", state.Pass, state.Fail)
	}
}

func TestParseGoTestOutput_WithFailure(t *testing.T) {
	input := `--- FAIL: TestFoo (0.00s)
--- FAIL: TestBar (0.01s)
FAIL	github.com/foo/bar	0.123s
ok  	github.com/foo/baz	0.456s`
	state := parseGoTestOutput(input)
	if state.Pass != 1 || state.Fail != 1 {
		t.Errorf("want 1/1, got %d/%d", state.Pass, state.Fail)
	}
	if len(state.FailedNames) != 2 {
		t.Fatalf("want 2 failed test names, got %d: %v", len(state.FailedNames), state.FailedNames)
	}
	if state.FailedNames[0] != "TestFoo" {
		t.Errorf("unexpected failed name: %q", state.FailedNames[0])
	}
}

func TestParseGoTestOutput_WithSkip(t *testing.T) {
	input := `?   	github.com/foo/noop	[no test files]
ok  	github.com/foo/bar	0.123s`
	state := parseGoTestOutput(input)
	if state.Pass != 1 || state.Skip != 1 {
		t.Errorf("want pass=1 skip=1, got pass=%d skip=%d", state.Pass, state.Skip)
	}
}

func TestParseGoTestOutput_Empty(t *testing.T) {
	state := parseGoTestOutput("")
	if state.Pass != 0 || state.Fail != 0 {
		t.Errorf("expected zeros, got %+v", state)
	}
}

func TestCaptureCustomTests_Pass(t *testing.T) {
	dir := t.TempDir()
	state := CaptureCustomTests(dir, "exit 0", 10*time.Second)
	if state.Tool != "custom" {
		t.Errorf("want tool=custom, got %q", state.Tool)
	}
	if state.Pass != 1 || state.Fail != 0 {
		t.Errorf("want pass=1 fail=0, got pass=%d fail=%d", state.Pass, state.Fail)
	}
}

func TestCaptureCustomTests_Fail(t *testing.T) {
	dir := t.TempDir()
	state := CaptureCustomTests(dir, "echo 'test failed output' && exit 1", 10*time.Second)
	if state.Pass != 0 || state.Fail != 1 {
		t.Errorf("want pass=0 fail=1, got pass=%d fail=%d", state.Pass, state.Fail)
	}
	if len(state.FailedNames) == 0 {
		t.Error("expected last output lines in FailedNames on failure")
	}
}

func TestCaptureCustomTests_Timeout(t *testing.T) {
	dir := t.TempDir()
	state := CaptureCustomTests(dir, "sleep 60", 1*time.Millisecond)
	if !state.TimedOut {
		t.Error("expected TimedOut=true")
	}
}
