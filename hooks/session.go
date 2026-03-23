package hooks

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/AgusRdz/ctx/logging"
	"github.com/AgusRdz/ctx/snapshot"
)

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
	} else {
		fmt.Print(content)
		logging.Log("session | project=%s | snapshot=found", projectDir)
	}

	return nil
}
