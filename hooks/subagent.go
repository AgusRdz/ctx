package hooks

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/AgusRdz/ctx/agents"
	"github.com/AgusRdz/ctx/config"
	"github.com/AgusRdz/ctx/logging"
	"github.com/AgusRdz/ctx/snapshot"
)

// SubagentStopInput is the JSON payload Claude Code sends to SubagentStop hooks via stdin.
type SubagentStopInput struct {
	SessionID      string `json:"session_id"`
	AgentName      string `json:"agent_name"`
	Output         string `json:"output"`
	ProjectDir     string `json:"cwd"`
	TranscriptPath string `json:"transcript_path"`
}

// RunSubagentStop handles the SubagentStop hook invocation.
// Reads the sub-agent's transcript, summarizes it via claude -p, and stores a named snapshot.
func RunSubagentStop() error {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("ctx: reading stdin: %w", err)
	}

	var input SubagentStopInput
	if err := json.Unmarshal(data, &input); err != nil {
		return fmt.Errorf("ctx: parsing hook input: %w", err)
	}

	projectDir := input.ProjectDir
	if projectDir == "" {
		projectDir = os.Getenv("CLAUDE_PROJECT_DIR")
	}
	if projectDir == "" {
		projectDir, _ = os.Getwd()
	}
	// Normalize to git root so all subagents in the same repo share the same bucket
	projectDir = agents.GitRoot(projectDir)

	cfg, err := config.EffectiveConfig(projectDir)
	if err != nil {
		logging.Log("subagent | ERROR: reading config: %v", err)
		return nil // non-fatal
	}

	if cfg.Agents.Mode != "on" {
		return nil
	}

	projectHash := snapshot.ProjectHash(projectDir)
	now := time.Now()
	name := agents.AgentName(projectDir, now)
	agentType := "general"
	if input.AgentName != "" {
		agentType = "custom"
	}

	timeout := config.ClaudeTimeout(cfg.Core.ClaudeTimeoutSecs)

	// Find and summarize the agent's transcript.
	// Validate TranscriptPath is under ~/.claude/ before reading.
	transcriptPath := ""
	if input.TranscriptPath != "" && isValidTranscriptPath(input.TranscriptPath) {
		transcriptPath = input.TranscriptPath
	}
	if transcriptPath == "" {
		transcriptPath = agents.FindAgentTranscript(input.SessionID)
	}

	var summary string
	if transcriptPath != "" {
		lines, extractErr := snapshot.ExtractTranscriptLines(transcriptPath, 30)
		if extractErr == nil && lines != "" {
			generated, genErr := agents.GenerateAgentSummary(lines, projectDir, timeout)
			if genErr != nil {
				logging.Log("subagent | WARNING: summary generation failed: %v", genErr)
			} else {
				summary = generated
			}
		}
	}
	if summary == "" {
		summary = input.Output
	}

	s := agents.AgentSnapshot{
		Name:        name,
		Type:        agentType,
		StoppedAt:   now,
		FinalOutput: summary,
	}

	if err := agents.WriteAgentSnapshot(projectHash, s); err != nil {
		logging.Log("subagent | ERROR: writing snapshot: %v", err)
		return nil
	}

	logging.Log("subagent | agent=%s | type=%s | project=%s | status=ok", name, agentType, projectDir)
	return nil
}

