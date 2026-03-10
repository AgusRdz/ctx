package snapshot

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AgusRdz/ctx/config"
)

// cleanup removes the snapshot directory for a given project path.
func cleanup(t *testing.T, projectDir string) {
	t.Helper()
	dir := filepath.Join(config.DataDir(), ProjectHash(projectDir))
	os.RemoveAll(dir)
}

func TestProjectHash_Consistent(t *testing.T) {
	path := t.TempDir()
	h1 := ProjectHash(path)
	h2 := ProjectHash(path)
	if h1 != h2 {
		t.Fatalf("expected same hash for same path, got %s and %s", h1, h2)
	}
}

func TestProjectHash_DifferentPaths(t *testing.T) {
	p1 := t.TempDir()
	p2 := t.TempDir()
	h1 := ProjectHash(p1)
	h2 := ProjectHash(p2)
	if h1 == h2 {
		t.Fatalf("expected different hashes for different paths, both got %s", h1)
	}
}

func TestWriteReadRoundtrip(t *testing.T) {
	projectDir := t.TempDir()
	t.Cleanup(func() { cleanup(t, projectDir) })

	content := "# Snapshot\n\ngoal: test roundtrip\n"
	if err := Write(projectDir, content); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	got, err := Read(projectDir)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if got != content {
		t.Fatalf("expected %q, got %q", content, got)
	}
}

func TestRead_Nonexistent(t *testing.T) {
	projectDir := t.TempDir()
	// no cleanup needed — nothing written

	got, err := Read(projectDir)
	if err != nil {
		t.Fatalf("Read should not error for nonexistent snapshot: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestClear_RemovesSnapshot(t *testing.T) {
	projectDir := t.TempDir()
	t.Cleanup(func() { cleanup(t, projectDir) })

	if err := Write(projectDir, "to be cleared"); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if err := Clear(projectDir); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	got, err := Read(projectDir)
	if err != nil {
		t.Fatalf("Read after Clear failed: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty string after Clear, got %q", got)
	}
}

func TestClear_Nonexistent(t *testing.T) {
	projectDir := t.TempDir()

	err := Clear(projectDir)
	if err != nil {
		t.Fatalf("Clear on nonexistent snapshot should not error: %v", err)
	}
}

func TestGoalFromSnapshot_Standard(t *testing.T) {
	content := "# Session Context\n\n_Captured: 2026-03-09T14:00Z_\n\n## Goal\nBuild authentication middleware\n\n## Decisions\n- Use JWT\n"
	got := goalFromSnapshot(content)
	if got != "Build authentication middleware" {
		t.Errorf("expected goal, got %q", got)
	}
}

func TestGoalFromSnapshot_NoTimestamp(t *testing.T) {
	content := "# Session Context\n\n## Goal\nDeploy to production\n\n## Decisions\n"
	got := goalFromSnapshot(content)
	if got != "Deploy to production" {
		t.Errorf("expected goal without timestamp, got %q", got)
	}
}

func TestGoalFromSnapshot_NoGoalSection(t *testing.T) {
	content := "# Session Context\n\n## Decisions\n- some decision\n"
	got := goalFromSnapshot(content)
	if got != "unknown" {
		t.Errorf("expected 'unknown' when no Goal section, got %q", got)
	}
}

func TestGoalFromSnapshot_EmptyContent(t *testing.T) {
	got := goalFromSnapshot("")
	if got != "unknown" {
		t.Errorf("expected 'unknown' for empty content, got %q", got)
	}
}

func TestGoalFromSnapshot_BlankLinesAroundGoal(t *testing.T) {
	content := "## Goal\n\n\nActual goal text\n\n## Next\n"
	got := goalFromSnapshot(content)
	if got != "Actual goal text" {
		t.Errorf("expected goal text after blank lines, got %q", got)
	}
}
