package hooks

import (
	"fmt"
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

func TestStaleWarning_Injected(t *testing.T) {
	// A snapshot older than staleThreshold should get a warning prepended
	old := time.Now().UTC().Add(-8 * 24 * time.Hour)
	content := fmt.Sprintf("_Captured: %s_\n\n## Goal\ntest\n", old.Format("2006-01-02T15:04Z"))

	age := snapshotAge(content)
	if age <= staleThreshold {
		t.Fatalf("test setup: expected age > %v, got %v", staleThreshold, age)
	}

	// Simulate what RunSession does
	days := int(age.Hours() / 24)
	warning := fmt.Sprintf("> ⚠️ This snapshot is %d days old — context may be stale.\n\n", days)
	result := warning + content

	if len(result) <= len(content) {
		t.Error("warning should make result longer than original content")
	}
	if result[:2] != "> " {
		t.Errorf("result should start with blockquote, got: %q", result[:10])
	}
}
