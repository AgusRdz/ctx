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

// SubagentStopInput is the JSON payload Claude Code sends to SubagentStop hooks via stdin.
type SubagentStopInput struct {
	SessionID  string `json:"session_id"`
	AgentName  string `json:"agent_name"`
	Output     string `json:"output"`
	ProjectDir string `json:"cwd"`
}

// RunSubagentStop handles the SubagentStop hook invocation.
// Captures subagent final output (v1 and v2) and internal state (v2).
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

	// Read effective config for this project
	cfg, err := config.EffectiveConfig(projectDir)
	if err != nil {
		logging.Log("subagent | ERROR: reading config: %v", err)
		return nil // non-fatal
	}

	if cfg.Agents.Mode == "off" || cfg.Agents.Mode == "" {
		return nil
	}

	projectHash := snapshot.ProjectHash(projectDir)

	// Determine agent name and type
	name := strings.TrimSpace(input.AgentName)
	agentType := "custom"
	if name == "" {
		name = fmt.Sprintf("agent-%d", time.Now().Unix())
		agentType = "general"
	}

	// For v2: check for internal state file written by PreCompact in subagent context
	var internalState string
	if cfg.Agents.Mode == "v2" && input.SessionID != "" {
		internalPath := agents.InternalStatePath(projectHash, input.SessionID)
		if stateData, readErr := os.ReadFile(internalPath); readErr == nil {
			internalState = string(stateData)
			os.Remove(internalPath) // clean up temp file
		}
	}

	s := agents.AgentSnapshot{
		Name:          name,
		Type:          agentType,
		StoppedAt:     time.Now(),
		FinalOutput:   input.Output,
		InternalState: internalState,
	}

	if err := agents.WriteAgentSnapshot(projectHash, s); err != nil {
		logging.Log("subagent | ERROR: writing snapshot: %v", err)
		return nil
	}

	logging.Log("subagent | agent=%s | type=%s | project=%s | mode=%s | status=ok",
		name, agentType, projectDir, cfg.Agents.Mode)
	return nil
}
