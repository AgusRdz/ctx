package updater

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func withTempDataDir(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("LOCALAPPDATA", tmp)
	t.Setenv("HOME", tmp)
}

func TestShouldCheck_NeverChecked(t *testing.T) {
	withTempDataDir(t)
	if !shouldCheck() {
		t.Error("should return true when never checked")
	}
}

func TestShouldCheck_RecentlyChecked(t *testing.T) {
	withTempDataDir(t)
	touchLastCheck()
	if shouldCheck() {
		t.Error("should return false when recently checked")
	}
}

func TestShouldCheck_StaleCheck(t *testing.T) {
	withTempDataDir(t)
	path := lastCheckPath()
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, []byte("old"), 0o644)
	stale := time.Now().Add(-25 * time.Hour)
	os.Chtimes(path, stale, stale)
	if !shouldCheck() {
		t.Error("should return true when check is stale (>24h)")
	}
}

func TestApplyPendingUpdate_DevVersion(t *testing.T) {
	ApplyPendingUpdate("dev") // no-op, just verify no panic
}

func TestApplyPendingUpdate_NoPending(t *testing.T) {
	withTempDataDir(t)
	ApplyPendingUpdate("v1.0.0") // silent when no pending file
}

func TestApplyPendingUpdate_InvalidMarker(t *testing.T) {
	withTempDataDir(t)
	path := pendingUpdatePath()
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, []byte("v2.0.0"), 0o644) // missing binary path

	ApplyPendingUpdate("v1.0.0")

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("should clean up invalid marker file")
	}
}

func TestApplyPendingUpdate_MissingBinary(t *testing.T) {
	withTempDataDir(t)
	path := pendingUpdatePath()
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, []byte("v2.0.0\n/nonexistent/ctx.new"), 0o644)

	ApplyPendingUpdate("v1.0.0")

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("should clean up marker when binary is missing")
	}
}

func TestBackgroundCheck_DevVersion(t *testing.T) {
	BackgroundCheck("dev")
	BackgroundCheck("v1.0.0-dirty")
}

func TestTouchLastCheck(t *testing.T) {
	withTempDataDir(t)
	touchLastCheck()
	if _, err := os.Stat(lastCheckPath()); os.IsNotExist(err) {
		t.Error("touch should create the check file")
	}
}

func TestReplaceBinary(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "ctx")
	src := filepath.Join(dir, "ctx.new")

	os.WriteFile(dest, []byte("old"), 0o755)
	os.WriteFile(src, []byte("new"), 0o755)

	if err := replaceBinary(dest, src); err != nil {
		t.Fatalf("replaceBinary failed: %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new" {
		t.Errorf("expected 'new', got %q", string(data))
	}

	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("source file should be removed after rename")
	}
}
