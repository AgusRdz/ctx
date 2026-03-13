package agents

import (
	"os"
	"path/filepath"
	"strings"
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
		Name:        "fix-RES-376-20260313-174200",
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

	os.WriteFile(filepath.Join(dir, ".pending-session123.md"), []byte("temp"), 0o644)
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

func TestArchiveCurrentAgents(t *testing.T) {
	withTempDataDir(t)
	projectHash := "archivehash"

	s1 := AgentSnapshot{Name: "main-20260313-170000", Type: "general", StoppedAt: time.Now(), FinalOutput: "done 1"}
	s2 := AgentSnapshot{Name: "main-20260313-170100", Type: "general", StoppedAt: time.Now(), FinalOutput: "done 2"}
	WriteAgentSnapshot(projectHash, s1)
	WriteAgentSnapshot(projectHash, s2)

	if err := ArchiveCurrentAgents(projectHash); err != nil {
		t.Fatalf("ArchiveCurrentAgents: %v", err)
	}

	// Current agents dir should be empty
	current, _ := ReadAgentSnapshots(projectHash)
	if len(current) != 0 {
		t.Errorf("expected 0 current agents after archive, got %d", len(current))
	}

	// Archive should have a subdirectory with 2 files
	archiveBase := ArchiveDir(projectHash)
	entries, err := os.ReadDir(archiveBase)
	if err != nil {
		t.Fatalf("ReadDir archive: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 archive slot, got %d", len(entries))
	}
	slotEntries, _ := os.ReadDir(filepath.Join(archiveBase, entries[0].Name()))
	if len(slotEntries) != 2 {
		t.Errorf("expected 2 archived agents, got %d", len(slotEntries))
	}
}

func TestArchiveCurrentAgents_NoOp(t *testing.T) {
	withTempDataDir(t)
	// Should not error when there's nothing to archive
	if err := ArchiveCurrentAgents("emptyhash"); err != nil {
		t.Fatalf("ArchiveCurrentAgents on empty: %v", err)
	}
}

func TestAgentName(t *testing.T) {
	ts := time.Date(2026, 3, 13, 17, 42, 0, 0, time.UTC)
	// In a non-git directory, branch falls back to "no-branch"
	name := AgentName(t.TempDir(), ts)
	if !strings.HasSuffix(name, "-20260313-174200") {
		t.Errorf("expected name to end with -20260313-174200, got %q", name)
	}
}

func TestParseAgentSnapshot_LegacyFinalOutput(t *testing.T) {
	// Ensure old "## Final Output" section is still parsed
	content := "# Agent: old-agent\n_Stopped: 2026-03-13T17:42Z_\n_Type: general_\n\n## Final Output\nsome output\n"
	s := parseAgentSnapshot(content)
	if s.FinalOutput != "some output" {
		t.Errorf("expected legacy final output parsed, got %q", s.FinalOutput)
	}
}
