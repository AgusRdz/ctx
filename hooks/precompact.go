package hooks

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AgusRdz/ctx/config"
	"github.com/AgusRdz/ctx/logging"
	"github.com/AgusRdz/ctx/projectstate"
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

	// Extract transcript lines — validate path is under ~/.claude/ before reading.
	var transcriptLines string
	if input.TranscriptPath != "" && isValidTranscriptPath(input.TranscriptPath) {
		var extractErr error
		transcriptLines, extractErr = snapshot.ExtractTranscriptLines(input.TranscriptPath, 20)
		if extractErr != nil {
			logging.Log("precompact | WARNING: transcript extraction failed: %v", extractErr)
		}
	}
	logging.Debug("precompact | transcript_path=%s | extracted_bytes=%d",
		sanitizeForLog(input.TranscriptPath), len(transcriptLines))

	// Generate snapshot via claude -p, with fallback
	cfg, _ := config.EffectiveConfig(projectDir)
	timeout := config.ClaudeTimeout(cfg.Core.ClaudeTimeoutSecs)
	content, err := snapshot.Generate(ctx, transcriptLines, timeout)
	if err != nil {
		logging.Log("precompact | WARNING: %v, using fallback", err)
		content = snapshot.GenerateFallback(ctx)
	}
	logging.Debug("precompact | snapshot_bytes=%d", len(content))

	// Append project state if enabled
	if cfg.ProjectState.Enabled {
		opts := projectstate.CaptureOptions{
			Git:                 cfg.ProjectState.Git,
			MaxDirtyFiles:       cfg.ProjectState.MaxDirtyFiles,
			MaxErrors:           cfg.ProjectState.MaxErrors,
			TypeCheck:           cfg.ProjectState.TypeCheck.Enabled,
			TypeCheckTimeout:    config.ClaudeTimeout(cfg.ProjectState.TypeCheck.TimeoutSeconds),
			TypeCheckCommand:    cfg.ProjectState.TypeCheck.Command,
			Tests:               cfg.ProjectState.Tests.Enabled,
			TestsTimeout:        config.ClaudeTimeout(cfg.ProjectState.Tests.TimeoutSeconds),
			TestsMaxFailedNames: cfg.ProjectState.Tests.MaxFailedNames,
			TestsCommand:        cfg.ProjectState.Tests.Command,
		}
		ps := projectstate.Capture(projectDir, opts)
		content += "\n" + projectstate.Format(ps, opts.MaxDirtyFiles, opts.MaxErrors)
		logging.Debug("precompact | project_state=captured | dirty_files=%d | typecheck=%s | tc_errors=%d | tests=%s",
			len(ps.Git.DirtyFiles), ps.TypeCheck.Tool, ps.TypeCheck.ErrorCount, ps.Tests.Tool)
	}

	// Write snapshot
	if err := snapshot.Write(projectDir, content); err != nil {
		logging.Log("precompact | ERROR: %v", err)
		return err
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

// isValidTranscriptPath returns true if path is a .jsonl file under ~/.claude/.
// Prevents hook input from being used to read arbitrary files.
func isValidTranscriptPath(path string) bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	claudeDir := filepath.Join(home, ".claude")
	// EvalSymlinks resolves symlinks so a symlink inside ~/.claude/ pointing
	// outside cannot bypass the prefix check. Fall back to Abs if the file
	// doesn't exist yet (transcript may not exist at hook invocation time).
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		// File may not exist yet — fall back to lexical resolution.
		resolved, err = filepath.Abs(path)
		if err != nil {
			return false
		}
	}
	return strings.HasPrefix(resolved, claudeDir+string(filepath.Separator)) &&
		strings.HasSuffix(resolved, ".jsonl")
}

// sanitizeForLog replaces newlines and carriage returns with spaces to prevent log injection.
func sanitizeForLog(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return s
}
