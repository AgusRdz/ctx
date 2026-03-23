package projectstate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestRoundTrip_CaptureAndFormat verifies that Capture → Format produces a
// coherent markdown block containing all expected fields.
func TestRoundTrip_CaptureAndFormat(t *testing.T) {
	dir := initTestRepo(t)
	// Modify a tracked file to create a dirty working tree
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("modified content\n"), 0o644)

	opts := CaptureOptions{
		Git:           true,
		MaxDirtyFiles: 10,
		MaxErrors:     5,
	}
	ps := Capture(dir, opts)
	out := Format(ps, opts.MaxDirtyFiles, opts.MaxErrors)

	if !strings.HasPrefix(out, "## Project State (at compaction)\n") {
		t.Errorf("output does not start with expected header:\n%s", out)
	}
	if !strings.Contains(out, "Git:") {
		t.Error("missing Git line")
	}
	if !strings.Contains(out, "main") {
		t.Error("missing branch name")
	}
	if !strings.Contains(out, "README.md") {
		t.Errorf("expected README.md in dirty files, output:\n%s", out)
	}
}

// TestRoundTrip_AppendToSnapshot verifies the pattern used by precompact.go:
// snapshot content + "\n" + project state block produces valid combined output.
func TestRoundTrip_AppendToSnapshot(t *testing.T) {
	dir := initTestRepo(t)

	snapshot := "# Session Context\n\n_Captured: 2026-01-01T00:00Z_\n\n## Goal\ntest goal\n\n## Next\ndo something\n"

	opts := CaptureOptions{Git: true, MaxDirtyFiles: 10, MaxErrors: 5}
	ps := Capture(dir, opts)
	combined := snapshot + "\n" + Format(ps, opts.MaxDirtyFiles, opts.MaxErrors)

	if !strings.Contains(combined, "## Goal") {
		t.Error("original snapshot content lost")
	}
	if !strings.Contains(combined, "## Project State (at compaction)") {
		t.Error("project state section missing from combined output")
	}
	// Sections should appear in order
	goalIdx := strings.Index(combined, "## Goal")
	stateIdx := strings.Index(combined, "## Project State")
	if goalIdx >= stateIdx {
		t.Error("## Project State should appear after ## Goal")
	}
}

// TestRoundTrip_JSONRoundTrip verifies that FormatJSON produces valid JSON
// with the expected fields.
func TestRoundTrip_JSONRoundTrip(t *testing.T) {
	dir := initTestRepo(t)

	opts := CaptureOptions{Git: true, MaxDirtyFiles: 10, MaxErrors: 5}
	ps := Capture(dir, opts)
	out, err := FormatJSON(ps)
	if err != nil {
		t.Fatalf("FormatJSON error: %v", err)
	}
	if !strings.Contains(out, `"branch"`) {
		t.Errorf("JSON missing branch field:\n%s", out)
	}
	if !strings.Contains(out, `"captured_at"`) {
		t.Errorf("JSON missing captured_at field:\n%s", out)
	}
	if !strings.Contains(out, "main") {
		t.Errorf("JSON missing branch value:\n%s", out)
	}
}

// TestRoundTrip_Timeout verifies that a very short timeout produces a partial
// state rather than a panic or hang.
func TestRoundTrip_Timeout(t *testing.T) {
	dir := initTestRepo(t)
	// 1 nanosecond timeout — git commands will exceed it
	state := CaptureGit(dir, 1*time.Nanosecond)
	// Should return empty/partial state, not panic
	_ = state
}
