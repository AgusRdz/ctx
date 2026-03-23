package hooks

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestSnapshotAge_ValidTimestamp(t *testing.T) {
	past := time.Now().UTC().Add(-48 * time.Hour)
	content := fmt.Sprintf("# Session Context\n\n_Captured: %s_\n\n## Goal\ntest\n", past.Format("2006-01-02T15:04Z"))

	age := snapshotAge(content)
	if age < 47*time.Hour || age > 49*time.Hour {
		t.Errorf("expected age ~48h, got %v", age)
	}
}

func TestSnapshotAge_NoTimestamp(t *testing.T) {
	content := "# Session Context\n\n## Goal\ntest\n"
	age := snapshotAge(content)
	if age != 0 {
		t.Errorf("expected 0 when no timestamp, got %v", age)
	}
}

func TestSnapshotAge_MalformedTimestamp(t *testing.T) {
	content := "# Session Context\n\n_Captured: not-a-date_\n\n## Goal\ntest\n"
	age := snapshotAge(content)
	if age != 0 {
		t.Errorf("expected 0 for malformed timestamp, got %v", age)
	}
}

func TestSnapshotAge_RecentSnapshot(t *testing.T) {
	now := time.Now().UTC()
	content := fmt.Sprintf("_Captured: %s_\n", now.Format("2006-01-02T15:04Z"))
	age := snapshotAge(content)
	if age > time.Hour {
		t.Errorf("expected recent snapshot to have age < 1h, got %v", age)
	}
}

func TestExtractSnapshotCommit_Found(t *testing.T) {
	content := "## Project State (at compaction)\nGit:  main | ↑1 ↓0 | last: a3f2b1c Add auth middleware\n"
	hash := extractSnapshotCommit(content)
	if hash != "a3f2b1c" {
		t.Errorf("want a3f2b1c, got %q", hash)
	}
}

func TestExtractSnapshotCommit_NoProjectState(t *testing.T) {
	content := "# Session Context\n\n## Goal\ntest\n"
	hash := extractSnapshotCommit(content)
	if hash != "" {
		t.Errorf("want empty, got %q", hash)
	}
}

func TestExtractSnapshotCommit_NoUpstream(t *testing.T) {
	// Git line without ahead/behind (no upstream)
	content := "Git:  main | last: b4c3d2e Fix typo\n"
	hash := extractSnapshotCommit(content)
	if hash != "b4c3d2e" {
		t.Errorf("want b4c3d2e, got %q", hash)
	}
}

func TestFormatAge(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{2 * time.Hour, "2 hours ago"},
		{1 * time.Hour, "1 hour ago"},
		{24 * time.Hour, "1 day ago"},
		{48 * time.Hour, "2 days ago"},
		{72 * time.Hour, "3 days ago"},
	}
	for _, c := range cases {
		got := formatAge(c.d)
		if got != c.want {
			t.Errorf("formatAge(%v): want %q, got %q", c.d, c.want, got)
		}
	}
}

func TestStalenessWarning_AgeOnly_Stale(t *testing.T) {
	old := time.Now().UTC().Add(-8 * 24 * time.Hour)
	content := fmt.Sprintf("_Captured: %s_\n\n## Goal\ntest\n", old.Format("2006-01-02T15:04Z"))

	warning := stalenessWarning(content, "")
	if !strings.Contains(warning, "days old") {
		t.Errorf("expected age-based warning, got: %q", warning)
	}
}

func TestStalenessWarning_AgeOnly_Fresh(t *testing.T) {
	now := time.Now().UTC()
	content := fmt.Sprintf("_Captured: %s_\n\n## Goal\ntest\n", now.Format("2006-01-02T15:04Z"))

	warning := stalenessWarning(content, "")
	if warning != "" {
		t.Errorf("expected no warning for fresh snapshot, got: %q", warning)
	}
}

func TestStalenessWarning_NoGitDir_FreshAge_NoWarning(t *testing.T) {
	// Non-git temp dir: gitShortHead returns "" → fall back to age check → fresh → no warning
	dir := t.TempDir()
	now := time.Now().UTC()
	content := fmt.Sprintf("_Captured: %s_\nGit:  main | last: abc1234 Some commit\n", now.Format("2006-01-02T15:04Z"))

	warning := stalenessWarning(content, dir)
	if warning != "" {
		t.Errorf("expected no warning for fresh snapshot in non-git dir, got: %q", warning)
	}
}
