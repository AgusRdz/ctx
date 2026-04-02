package hooks

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/AgusRdz/ctx/logging"
	"github.com/AgusRdz/ctx/snapshot"
)

// PostCompactInput is the JSON payload Claude Code sends to PostCompact hooks via stdin.
type PostCompactInput struct {
	SessionID      string `json:"session_id"`
	ProjectDir     string `json:"cwd"`
	Trigger        string `json:"trigger"`
	HookEventName  string `json:"hook_event_name"`
	PermissionMode string `json:"permission_mode"`
}

// RunPostCompact handles the PostCompact hook invocation.
// Reads the current snapshot for the project+branch and prints it to stdout
// so Claude Code re-injects context after compaction.
func RunPostCompact() error {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("ctx: reading stdin: %w", err)
	}

	var input PostCompactInput
	if err := json.Unmarshal(data, &input); err != nil {
		return fmt.Errorf("ctx: parsing hook input: %w", err)
	}

	projectDir := input.ProjectDir
	if projectDir == "" {
		projectDir, _ = os.Getwd()
	}

	branch := snapshot.BranchForProject(projectDir)

	content, err := snapshot.Read(projectDir, branch)
	if err != nil {
		logging.Log("postcompact | ERROR: %v", err)
		return err
	}

	trigger := parseTriggerFromArgs()
	if trigger == "" {
		trigger = input.Trigger
	}
	if trigger == "" {
		trigger = "unknown"
	}

	if content == "" {
		logging.Log("postcompact | trigger=%s | project=%s | branch=%s | snapshot=none",
			trigger, projectDir, branch)
		return nil
	}

	fmt.Print(content)
	logging.Log("postcompact | trigger=%s | project=%s | branch=%s | snapshot=found",
		trigger, projectDir, branch)
	return nil
}
