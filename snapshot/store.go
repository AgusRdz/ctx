package snapshot

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/AgusRdz/ctx/config"
)

// AgentInfo holds lightweight metadata about a captured agent snapshot.
type AgentInfo struct {
	Name      string
	Type      string
	StoppedAt time.Time
}

// ProjectHash returns the sha256 hex of the absolute project path.
func ProjectHash(projectDir string) string {
	abs, err := filepath.Abs(projectDir)
	if err != nil {
		abs = projectDir
	}
	h := sha256.Sum256([]byte(abs))
	return fmt.Sprintf("%x", h)
}

// snapshotPath returns the full path to a project's snapshot file.
func snapshotPath(projectDir string) string {
	return filepath.Join(config.DataDir(), ProjectHash(projectDir), "snapshot.md")
}

// Read returns the snapshot content for a project, or empty string if none exists.
func Read(projectDir string) (string, error) {
	data, err := os.ReadFile(snapshotPath(projectDir))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("ctx: %w", err)
	}
	return string(data), nil
}

// Write saves a snapshot for a project, creating directories as needed.
// It also stores the project path in path.txt for use by List().
func Write(projectDir string, content string) error {
	p := snapshotPath(projectDir)
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	// Store project path so List() can reverse the hash
	_ = os.WriteFile(filepath.Join(dir, "path.txt"), []byte(projectDir), 0o644)
	return nil
}

// Clear deletes the snapshot directory for a project (snapshot.md + path.txt).
func Clear(projectDir string) error {
	dir := filepath.Join(config.DataDir(), ProjectHash(projectDir))
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	return nil
}

// ClearAll deletes all project snapshot directories, preserving the log file.
func ClearAll() error {
	dataDir := config.DataDir()
	entries, err := os.ReadDir(dataDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue // preserve debug.log and other top-level files
		}
		if err := os.RemoveAll(filepath.Join(dataDir, entry.Name())); err != nil {
			return fmt.Errorf("ctx: %w", err)
		}
	}
	return nil
}

// ClearAgents deletes the agents/ subdirectory for a project, leaving the main snapshot intact.
func ClearAgents(projectDir string) error {
	dir := filepath.Join(config.DataDir(), ProjectHash(projectDir), "agents")
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	return nil
}

// ListAgents returns lightweight info about all captured agents for a project hash.
func ListAgents(projectHash string) ([]AgentInfo, error) {
	dir := filepath.Join(config.DataDir(), projectHash, "agents")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ctx: %w", err)
	}

	var results []AgentInfo
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		info := parseAgentInfo(string(data))
		results = append(results, info)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].StoppedAt.After(results[j].StoppedAt)
	})
	return results, nil
}

// parseAgentInfo extracts name, type, and stopped time from an agent snapshot file.
func parseAgentInfo(content string) AgentInfo {
	var info AgentInfo
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "# Agent: "):
			info.Name = strings.TrimPrefix(trimmed, "# Agent: ")
		case strings.HasPrefix(trimmed, "_Stopped: ") && strings.HasSuffix(trimmed, "_"):
			inner := strings.TrimPrefix(trimmed, "_Stopped: ")
			inner = strings.TrimSuffix(inner, "_")
			t, _ := time.Parse("2006-01-02T15:04Z", inner)
			info.StoppedAt = t
		case strings.HasPrefix(trimmed, "_Type: ") && strings.HasSuffix(trimmed, "_"):
			info.Type = strings.TrimPrefix(trimmed, "_Type: ")
			info.Type = strings.TrimSuffix(info.Type, "_")
		}
	}
	if info.Type == "" {
		if strings.HasPrefix(info.Name, "agent-") {
			info.Type = "general"
		} else {
			info.Type = "custom"
		}
	}
	return info
}

// SnapshotInfo holds metadata about a stored snapshot.
type SnapshotInfo struct {
	ProjectDir string
	Goal       string
	CapturedAt time.Time
}

// List returns info about all stored snapshots across all projects.
// The second return value is the count of legacy snapshots (pre-v0.1.7)
// that exist on disk but cannot be listed because they lack path.txt.
func List() ([]SnapshotInfo, int, error) {
	dataDir := config.DataDir()
	entries, err := os.ReadDir(dataDir)
	if os.IsNotExist(err) {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, fmt.Errorf("ctx: %w", err)
	}

	var results []SnapshotInfo
	legacy := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		entryDir := filepath.Join(dataDir, entry.Name())

		// Read project path
		pathData, err := os.ReadFile(filepath.Join(entryDir, "path.txt"))
		if err != nil {
			// Has snapshot.md but no path.txt — legacy entry
			if _, serr := os.Stat(filepath.Join(entryDir, "snapshot.md")); serr == nil {
				legacy++
			}
			continue
		}
		projectDir := strings.TrimSpace(string(pathData))

		// Read snapshot for goal extraction
		snapData, err := os.ReadFile(filepath.Join(entryDir, "snapshot.md"))
		if err != nil {
			continue
		}
		goal := goalFromSnapshot(string(snapData))

		// Use snapshot file mod time as capture time
		snapInfo, err := os.Stat(filepath.Join(entryDir, "snapshot.md"))
		capturedAt := time.Time{}
		if err == nil {
			capturedAt = snapInfo.ModTime()
		}

		results = append(results, SnapshotInfo{
			ProjectDir: projectDir,
			Goal:       goal,
			CapturedAt: capturedAt,
		})
	}
	// Sort most recently captured first
	sort.Slice(results, func(i, j int) bool {
		return results[i].CapturedAt.After(results[j].CapturedAt)
	})

	return results, legacy, nil
}

// goalFromSnapshot extracts the goal line from a formatted snapshot.
func goalFromSnapshot(content string) string {
	inGoal := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "## Goal" {
			inGoal = true
			continue
		}
		if inGoal {
			if trimmed == "" || strings.HasPrefix(trimmed, "_") {
				continue // skip blank lines and _Captured:_ lines
			}
			if strings.HasPrefix(trimmed, "##") {
				break // reached next section
			}
			return trimmed
		}
	}
	return "unknown"
}
