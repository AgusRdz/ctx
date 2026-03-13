package hooks

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/AgusRdz/ctx/agents"
	"github.com/AgusRdz/ctx/config"
	"github.com/AgusRdz/ctx/logging"
	"github.com/AgusRdz/ctx/snapshot"
)

// PreCompactInput is the JSON payload Claude Code sends to PreCompact hooks via stdin.
type PreCompactInput struct {
	SessionID         string `json:"session_id"`
	ProjectDir        string `json:"cwd"`
	TranscriptPath    string `json:"transcript_path"`
	PermissionMode    string `json:"permission_mode"`
	HookEventName     string `json:"hook_event_name"`
	Trigger           string `json:"trigger"`
	CustomInstructions string `json:"custom_instructions"`
}

// RunPreCompact handles the PreCompact hook invocation.
// Reads JSON from stdin, collects context, generates snapshot, writes it.
func RunPreCompact() error {
	start := time.Now()

	// Read stdin
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("ctx: reading stdin: %w", err)
	}

	var input PreCompactInput
	if err := json.Unmarshal(data, &input); err != nil {
		return fmt.Errorf("ctx: parsing hook input: %w", err)
	}

	projectDir := input.ProjectDir
	if projectDir == "" {
		projectDir, _ = os.Getwd()
	}

	// Collect deterministic context
	ctx := snapshot.Collect(projectDir)
	logging.Debug("precompact | diff_stat_bytes=%d | project_md_bytes=%d | recent_log_lines=%d",
		len(ctx.DiffStat), len(ctx.ProjectMD), len(strings.Split(strings.TrimSpace(ctx.RecentLog), "\n")))

	// Extract transcript lines
	var transcriptLines string
	if input.TranscriptPath != "" {
		transcriptLines, _ = snapshot.ExtractTranscriptLines(input.TranscriptPath, 20)
	}
	logging.Debug("precompact | transcript_path=%s | extracted_bytes=%d",
		input.TranscriptPath, len(transcriptLines))

	// Generate snapshot via claude -p, with fallback
	content, err := snapshot.Generate(ctx, transcriptLines)
	if err != nil {
		logging.Log("precompact | WARNING: %v, using fallback", err)
		content = snapshot.GenerateFallback(ctx)
	}
	logging.Debug("precompact | snapshot_bytes=%d", len(content))

	// Write snapshot
	if err := snapshot.Write(projectDir, content); err != nil {
		logging.Log("precompact | ERROR: %v", err)
		return err
	}

	// v2: also write internal state keyed by session ID for SubagentStop to pick up
	if input.SessionID != "" {
		projectHash := snapshot.ProjectHash(projectDir)
		cfg, cfgErr := config.EffectiveConfig(projectDir)
		if cfgErr == nil && cfg.Agents.Mode == "v2" {
			_ = agents.WriteInternalState(projectHash, input.SessionID, content)
		}
	}

	duration := time.Since(start)
	trigger := parseTriggerFromArgs()
	if trigger == "" {
		trigger = input.Trigger
	}
	if trigger == "" {
		trigger = "unknown"
	}
	logging.Log("precompact | trigger=%s | project=%s | duration=%.1fs | status=ok",
		trigger, projectDir, duration.Seconds())

	return nil
}

// parseTriggerFromArgs reads --trigger=<value> from os.Args.
func parseTriggerFromArgs() string {
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "--trigger=") {
			return strings.TrimPrefix(arg, "--trigger=")
		}
	}
	return ""
}
