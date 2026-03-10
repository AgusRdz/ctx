package hooks

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/AgusRdz/ctx/logging"
	"github.com/AgusRdz/ctx/snapshot"
)

const staleThreshold = 7 * 24 * time.Hour

// SessionInput is the JSON payload Claude Code sends to SessionStart hooks via stdin.
type SessionInput struct {
	SessionID      string `json:"session_id"`
	ProjectDir     string `json:"cwd"`
	TranscriptPath string `json:"transcript_path"`
	PermissionMode string `json:"permission_mode"`
	HookEventName  string `json:"hook_event_name"`
	Source         string `json:"source"`
	Model          string `json:"model"`
}

// RunSession handles the SessionStart hook invocation.
// If a snapshot exists for the project, prints it to stdout.
func RunSession() error {
	// Read stdin
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("ctx: reading stdin: %w", err)
	}

	var input SessionInput
	if err := json.Unmarshal(data, &input); err != nil {
		return fmt.Errorf("ctx: parsing hook input: %w", err)
	}

	projectDir := input.ProjectDir
	if projectDir == "" {
		projectDir, _ = os.Getwd()
	}

	content, err := snapshot.Read(projectDir)
	if err != nil {
		logging.Log("session | ERROR: %v", err)
		return err
	}

	if content == "" {
		logging.Log("session | project=%s | snapshot=none", projectDir)
		return nil
	}

	// Prepend staleness warning if the snapshot is old
	if age := snapshotAge(content); age > staleThreshold {
		days := int(age.Hours() / 24)
		content = fmt.Sprintf("> ⚠️ This snapshot is %d days old — context may be stale.\n\n", days) + content
	}

	// Print to stdout — Claude Code injects this as context
	fmt.Print(content)
	logging.Log("session | project=%s | snapshot=found", projectDir)
	return nil
}

// snapshotAge parses the _Captured:_ timestamp from a snapshot and returns its age.
// Returns 0 if the timestamp cannot be parsed.
func snapshotAge(content string) time.Duration {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "_Captured:") && strings.HasSuffix(trimmed, "_") {
			inner := strings.TrimPrefix(trimmed, "_Captured: ")
			inner = strings.TrimSuffix(inner, "_")
			t, err := time.Parse("2006-01-02T15:04Z", inner)
			if err == nil {
				return time.Since(t)
			}
		}
	}
	return 0
}
