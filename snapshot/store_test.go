package snapshot

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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

func TestBranchForProject_NotGit(t *testing.T) {
	dir := t.TempDir()
	branch := BranchForProject(dir)
	if branch != "_" {
		t.Fatalf("expected '_' for non-git dir, got %q", branch)
	}
}

func TestSanitizeBranch_SlashReplaced(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"feature/my-thing", "feature-my-thing"},
		{"fix/auth/login", "fix-auth-login"},
		{"main", "main"},
		{"HEAD", "HEAD"},
		{"feature//double", "feature-double"},
	}
	for _, c := range cases {
		got := sanitizeBranch(c.input)
		if got != c.want {
			t.Errorf("sanitizeBranch(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestWriteReadRoundtrip(t *testing.T) {
	projectDir := t.TempDir()
	t.Cleanup(func() { cleanup(t, projectDir) })

	content := "# Snapshot\n\ngoal: test roundtrip\n"
	if err := Write(projectDir, "main", content); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	got, err := Read(projectDir, "main")
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if got != content {
		t.Fatalf("expected %q, got %q", content, got)
	}
}

func TestRead_Nonexistent(t *testing.T) {
	projectDir := t.TempDir()

	got, err := Read(projectDir, "main")
	if err != nil {
		t.Fatalf("Read should not error for nonexistent snapshot: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestWriteRead_BranchScoped(t *testing.T) {
	projectDir := t.TempDir()
	t.Cleanup(func() { cleanup(t, projectDir) })

	if err := Write(projectDir, "main", "main content"); err != nil {
		t.Fatalf("Write main failed: %v", err)
	}
	if err := Write(projectDir, "feature-auth", "feature content"); err != nil {
		t.Fatalf("Write feature failed: %v", err)
	}

	gotMain, err := Read(projectDir, "main")
	if err != nil || gotMain != "main content" {
		t.Fatalf("main branch: got %q, err %v", gotMain, err)
	}

	gotFeature, err := Read(projectDir, "feature-auth")
	if err != nil || gotFeature != "feature content" {
		t.Fatalf("feature branch: got %q, err %v", gotFeature, err)
	}
}

func TestRead_MigrateOnRead(t *testing.T) {
	projectDir := t.TempDir()
	t.Cleanup(func() { cleanup(t, projectDir) })

	// Manually write a legacy flat snapshot.md
	hashDir := filepath.Join(config.DataDir(), ProjectHash(projectDir))
	if err := os.MkdirAll(hashDir, 0o700); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	legacyContent := "legacy snapshot content"
	if err := os.WriteFile(filepath.Join(hashDir, "snapshot.md"), []byte(legacyContent), 0o600); err != nil {
		t.Fatalf("writing legacy snapshot failed: %v", err)
	}

	// Read with branch — should fall back to legacy
	got, err := Read(projectDir, "main")
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if got != legacyContent {
		t.Fatalf("expected legacy content %q, got %q", legacyContent, got)
	}
}

func TestClear_RemovesSnapshot(t *testing.T) {
	projectDir := t.TempDir()
	t.Cleanup(func() { cleanup(t, projectDir) })

	if err := Write(projectDir, "main", "to be cleared"); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if err := Clear(projectDir, "main"); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	got, err := Read(projectDir, "main")
	if err != nil {
		t.Fatalf("Read after Clear failed: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty string after Clear, got %q", got)
	}
}

func TestClear_BranchScoped(t *testing.T) {
	projectDir := t.TempDir()
	t.Cleanup(func() { cleanup(t, projectDir) })

	if err := Write(projectDir, "main", "main content"); err != nil {
		t.Fatalf("Write main failed: %v", err)
	}
	if err := Write(projectDir, "feature-x", "feature content"); err != nil {
		t.Fatalf("Write feature failed: %v", err)
	}

	if err := Clear(projectDir, "main"); err != nil {
		t.Fatalf("Clear main failed: %v", err)
	}

	// main should be gone
	gotMain, err := Read(projectDir, "main")
	if err != nil || gotMain != "" {
		t.Fatalf("expected main to be cleared, got %q, err %v", gotMain, err)
	}

	// feature should survive
	gotFeature, err := Read(projectDir, "feature-x")
	if err != nil || gotFeature != "feature content" {
		t.Fatalf("expected feature to survive, got %q, err %v", gotFeature, err)
	}
}

func TestClear_Nonexistent(t *testing.T) {
	projectDir := t.TempDir()

	err := Clear(projectDir, "main")
	if err != nil {
		t.Fatalf("Clear on nonexistent snapshot should not error: %v", err)
	}
}

func TestClearAll_RemovesAll(t *testing.T) {
	projectDir := t.TempDir()
	t.Cleanup(func() { cleanup(t, projectDir) })

	if err := Write(projectDir, "main", "main content"); err != nil {
		t.Fatalf("Write main failed: %v", err)
	}
	if err := Write(projectDir, "feature-x", "feature content"); err != nil {
		t.Fatalf("Write feature failed: %v", err)
	}

	if err := ClearAll(projectDir); err != nil {
		t.Fatalf("ClearAll failed: %v", err)
	}

	gotMain, _ := Read(projectDir, "main")
	gotFeature, _ := Read(projectDir, "feature-x")
	if gotMain != "" || gotFeature != "" {
		t.Fatalf("expected all snapshots cleared, got main=%q feature=%q", gotMain, gotFeature)
	}
}

func TestList_BranchEntries(t *testing.T) {
	projectDir := t.TempDir()
	t.Cleanup(func() { cleanup(t, projectDir) })

	if err := Write(projectDir, "main", "## Goal\nmain goal\n"); err != nil {
		t.Fatalf("Write main failed: %v", err)
	}
	if err := Write(projectDir, "feature-auth", "## Goal\nauth goal\n"); err != nil {
		t.Fatalf("Write feature failed: %v", err)
	}

	infos, _, err := List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	found := map[string]string{} // branch -> goal
	for _, info := range infos {
		if info.ProjectDir == projectDir {
			found[info.Branch] = info.Goal
		}
	}

	if found["main"] != "main goal" {
		t.Errorf("expected main goal, got %q", found["main"])
	}
	if found["feature-auth"] != "auth goal" {
		t.Errorf("expected auth goal, got %q", found["feature-auth"])
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

func TestClearStale_RemovesOldKeepsFresh(t *testing.T) {
	oldDir := t.TempDir()
	freshDir := t.TempDir()
	t.Cleanup(func() {
		cleanup(t, oldDir)
		cleanup(t, freshDir)
	})

	if err := Write(oldDir, "main", "## Goal\nold work\n"); err != nil {
		t.Fatalf("Write old: %v", err)
	}
	if err := Write(freshDir, "main", "## Goal\nfresh work\n"); err != nil {
		t.Fatalf("Write fresh: %v", err)
	}

	// Backdate the old snapshot's mtime by 100 days.
	oldPath := snapshotPathForBranch(oldDir, "main")
	past := time.Now().Add(-100 * 24 * time.Hour)
	if err := os.Chtimes(oldPath, past, past); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}

	removed, err := ClearStale(60 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("ClearStale: %v", err)
	}

	// Filter to entries we created (other tests may share DataDir).
	var ours []SnapshotInfo
	for _, info := range removed {
		if info.ProjectDir == oldDir || info.ProjectDir == freshDir {
			ours = append(ours, info)
		}
	}
	if len(ours) != 1 || ours[0].ProjectDir != oldDir {
		t.Fatalf("expected only oldDir removed, got %+v", ours)
	}

	if got, _ := Read(oldDir, "main"); got != "" {
		t.Errorf("expected old snapshot gone, got %q", got)
	}
	if got, _ := Read(freshDir, "main"); got == "" {
		t.Errorf("expected fresh snapshot preserved")
	}
}

func TestClearStale_CleansEmptyHashDir(t *testing.T) {
	projectDir := t.TempDir()
	t.Cleanup(func() { cleanup(t, projectDir) })

	if err := Write(projectDir, "main", "## Goal\nold\n"); err != nil {
		t.Fatalf("Write: %v", err)
	}
	oldPath := snapshotPathForBranch(projectDir, "main")
	past := time.Now().Add(-100 * 24 * time.Hour)
	if err := os.Chtimes(oldPath, past, past); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}

	if _, err := ClearStale(60 * 24 * time.Hour); err != nil {
		t.Fatalf("ClearStale: %v", err)
	}

	hashDir := filepath.Join(config.DataDir(), ProjectHash(projectDir))
	if _, err := os.Stat(hashDir); !os.IsNotExist(err) {
		t.Errorf("expected hash dir removed, stat err = %v", err)
	}
}

func TestGoalFromSnapshot_BlankLinesAroundGoal(t *testing.T) {
	content := "## Goal\n\n\nActual goal text\n\n## Next\n"
	got := goalFromSnapshot(content)
	if got != "Actual goal text" {
		t.Errorf("expected goal text after blank lines, got %q", got)
	}
}
