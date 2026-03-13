package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/AgusRdz/ctx/config"
)

// AgentSnapshot holds metadata and content for a captured subagent.
type AgentSnapshot struct {
	Name          string
	Type          string // custom | general
	StoppedAt     time.Time
	FinalOutput   string
	InternalState string // empty for v1, populated for v2
}

// AgentsDir returns the agents directory for a project hash.
func AgentsDir(projectHash string) string {
	return filepath.Join(config.DataDir(), projectHash, "agents")
}

// AgentSnapshotPath returns the path for an agent's snapshot file.
func AgentSnapshotPath(projectHash, name string) string {
	return filepath.Join(AgentsDir(projectHash), name+".md")
}

// InternalStatePath returns the path for a temporary v2 internal state file.
// Keyed by session ID to link PreCompact to SubagentStop.
func InternalStatePath(projectHash, sessionID string) string {
	return filepath.Join(AgentsDir(projectHash), ".pending-"+sessionID+".md")
}

// WriteAgentSnapshot writes an agent snapshot to disk.
func WriteAgentSnapshot(projectHash string, s AgentSnapshot) error {
	dir := AgentsDir(projectHash)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	content := formatAgentSnapshot(s)
	path := AgentSnapshotPath(projectHash, s.Name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	return nil
}

// WriteInternalState writes a v2 temporary internal state file keyed by session ID.
func WriteInternalState(projectHash, sessionID, content string) error {
	dir := AgentsDir(projectHash)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	path := InternalStatePath(projectHash, sessionID)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	return nil
}

// ReadAgentSnapshots reads all non-hidden agent snapshot files for a project.
// Returns snapshots sorted by most recent first.
func ReadAgentSnapshots(projectHash string) ([]AgentSnapshot, error) {
	dir := AgentsDir(projectHash)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ctx: %w", err)
	}

	var snapshots []AgentSnapshot
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue // skip hidden files and directories
		}
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		snapshots = append(snapshots, parseAgentSnapshot(string(data)))
	}

	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].StoppedAt.After(snapshots[j].StoppedAt)
	})
	return snapshots, nil
}

// ClearAgentSnapshots removes all agent snapshots (including hidden temp files) for a project.
func ClearAgentSnapshots(projectHash string) error {
	dir := AgentsDir(projectHash)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
				return fmt.Errorf("ctx: %w", err)
			}
		}
	}
	return nil
}

func formatAgentSnapshot(s AgentSnapshot) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Agent: %s\n", s.Name))
	b.WriteString(fmt.Sprintf("_Stopped: %s_\n", s.StoppedAt.UTC().Format("2006-01-02T15:04Z")))
	b.WriteString(fmt.Sprintf("_Type: %s_\n", s.Type))
	b.WriteString("\n")

	if s.InternalState != "" {
		b.WriteString("## Internal State (captured pre-compaction)\n")
		b.WriteString(s.InternalState)
		b.WriteString("\n\n")
	}

	b.WriteString("## Final Output\n")
	b.WriteString(s.FinalOutput)
	b.WriteString("\n")
	return b.String()
}

func parseAgentSnapshot(content string) AgentSnapshot {
	var s AgentSnapshot
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "# Agent: "):
			s.Name = strings.TrimPrefix(trimmed, "# Agent: ")
		case strings.HasPrefix(trimmed, "_Stopped: ") && strings.HasSuffix(trimmed, "_"):
			inner := strings.TrimPrefix(trimmed, "_Stopped: ")
			inner = strings.TrimSuffix(inner, "_")
			t, _ := time.Parse("2006-01-02T15:04Z", inner)
			s.StoppedAt = t
		case strings.HasPrefix(trimmed, "_Type: ") && strings.HasSuffix(trimmed, "_"):
			s.Type = strings.TrimPrefix(trimmed, "_Type: ")
			s.Type = strings.TrimSuffix(s.Type, "_")
		}
	}

	// Extract final output section
	inOutput := false
	var outputLines []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "## Final Output" {
			inOutput = true
			continue
		}
		if inOutput {
			// Stop at next ## section
			if strings.HasPrefix(strings.TrimSpace(line), "## ") {
				break
			}
			outputLines = append(outputLines, line)
		}
	}
	s.FinalOutput = strings.TrimSpace(strings.Join(outputLines, "\n"))

	// Infer type from name if not parsed
	if s.Type == "" {
		if strings.HasPrefix(s.Name, "agent-") {
			s.Type = "general"
		} else {
			s.Type = "custom"
		}
	}
	return s
}
