package agents

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

func TestWriteReadAgentSnapshot(t *testing.T) {
	withTempDataDir(t)

	projectHash := "testhash123"
	s := AgentSnapshot{
		Name:        "refactor-agent",
		Type:        "custom",
		StoppedAt:   time.Now().UTC().Truncate(time.Minute),
		FinalOutput: "Extracted AuthService to /services/auth.go",
	}

	if err := WriteAgentSnapshot(projectHash, s); err != nil {
		t.Fatalf("WriteAgentSnapshot: %v", err)
	}

	snapshots, err := ReadAgentSnapshots(projectHash)
	if err != nil {
		t.Fatalf("ReadAgentSnapshots: %v", err)
	}
	if len(snapshots) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snapshots))
	}

	got := snapshots[0]
	if got.Name != s.Name {
		t.Errorf("Name: want %q got %q", s.Name, got.Name)
	}
	if got.Type != s.Type {
		t.Errorf("Type: want %q got %q", s.Type, got.Type)
	}
	if !got.StoppedAt.Equal(s.StoppedAt) {
		t.Errorf("StoppedAt: want %v got %v", s.StoppedAt, got.StoppedAt)
	}
	if got.FinalOutput != s.FinalOutput {
		t.Errorf("FinalOutput: want %q got %q", s.FinalOutput, got.FinalOutput)
	}
}

func TestWriteReadAgentSnapshot_WithInternalState(t *testing.T) {
	withTempDataDir(t)

	projectHash := "testhash456"
	s := AgentSnapshot{
		Name:          "refactor-agent",
		Type:          "custom",
		StoppedAt:     time.Now().UTC().Truncate(time.Minute),
		FinalOutput:   "Completed refactoring",
		InternalState: "### Goal\nRefactor auth\n\n### Decisions\n- Use JWT\n",
	}

	if err := WriteAgentSnapshot(projectHash, s); err != nil {
		t.Fatalf("WriteAgentSnapshot: %v", err)
	}

	snapshots, err := ReadAgentSnapshots(projectHash)
	if err != nil {
		t.Fatalf("ReadAgentSnapshots: %v", err)
	}
	if len(snapshots) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snapshots))
	}
}

func TestReadAgentSnapshots_Empty(t *testing.T) {
	withTempDataDir(t)
	snapshots, err := ReadAgentSnapshots("nonexistent-hash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(snapshots) != 0 {
		t.Errorf("expected 0 snapshots, got %d", len(snapshots))
	}
}

func TestReadAgentSnapshots_SkipsHiddenFiles(t *testing.T) {
	withTempDataDir(t)
	projectHash := "testhash789"
	dir := AgentsDir(projectHash)
	os.MkdirAll(dir, 0o755)

	// Write a hidden temp file
	os.WriteFile(filepath.Join(dir, ".pending-session123.md"), []byte("temp"), 0o644)
	// Write a real snapshot
	s := AgentSnapshot{Name: "real-agent", Type: "custom", StoppedAt: time.Now(), FinalOutput: "done"}
	WriteAgentSnapshot(projectHash, s)

	snapshots, err := ReadAgentSnapshots(projectHash)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(snapshots) != 1 {
		t.Errorf("expected 1 snapshot (hidden file should be skipped), got %d", len(snapshots))
	}
	if snapshots[0].Name != "real-agent" {
		t.Errorf("expected real-agent, got %q", snapshots[0].Name)
	}
}

func TestClearAgentSnapshots(t *testing.T) {
	withTempDataDir(t)
	projectHash := "clearhash"

	s := AgentSnapshot{Name: "agent-one", Type: "general", StoppedAt: time.Now(), FinalOutput: "done"}
	WriteAgentSnapshot(projectHash, s)

	if err := ClearAgentSnapshots(projectHash); err != nil {
		t.Fatalf("ClearAgentSnapshots: %v", err)
	}

	snapshots, _ := ReadAgentSnapshots(projectHash)
	if len(snapshots) != 0 {
		t.Errorf("expected 0 snapshots after clear, got %d", len(snapshots))
	}
}

func TestWriteInternalState(t *testing.T) {
	withTempDataDir(t)
	projectHash := "internalhash"

	if err := WriteInternalState(projectHash, "session-abc", "internal content"); err != nil {
		t.Fatalf("WriteInternalState: %v", err)
	}

	path := InternalStatePath(projectHash, "session-abc")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "internal content" {
		t.Errorf("expected %q got %q", "internal content", string(data))
	}
}
