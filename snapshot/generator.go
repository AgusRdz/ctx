package snapshot

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

const promptTemplate = `Analyze this development session context and respond ONLY in valid JSON,
no explanations, no markdown, no backticks.

MODIFIED FILES:
%s

PROJECT GOAL:
%s

LAST EXECUTED COMMANDS:
%s

Respond with exactly this JSON:
{
  "goal": "one line describing the session objective",
  "decisions": ["relevant technical decision", "..."],
  "in_progress": "what was left unfinished",
  "next": "what to do first when resuming"
}`

// SnapshotData represents the structured snapshot content.
type SnapshotData struct {
	Goal       string   `json:"goal"`
	Decisions  []string `json:"decisions"`
	InProgress string   `json:"in_progress"`
	Next       string   `json:"next"`
}

// Generate calls claude -p to produce a semantic snapshot from collected context
// and transcript lines. Returns formatted markdown.
func Generate(ctx Context, transcriptLines string) (string, error) {
	prompt := fmt.Sprintf(promptTemplate, ctx.DiffStat, ctx.ProjectMD, transcriptLines)

	cmd := exec.Command("claude", "-p", prompt)
	cmd.Dir = ctx.ProjectDir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("ctx: claude -p failed: %w", err)
	}

	// Parse the JSON response
	raw := strings.TrimSpace(string(out))
	var data SnapshotData
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return "", fmt.Errorf("ctx: failed to parse claude response: %w", err)
	}

	return FormatSnapshot(data), nil
}

// GenerateFallback creates a deterministic snapshot without calling claude -p.
func GenerateFallback(ctx Context) string {
	data := SnapshotData{
		Goal:       "Unable to determine (claude -p unavailable)",
		Decisions:  []string{},
		InProgress: ctx.DiffStat,
		Next:       "Review modified files and continue",
	}
	return FormatSnapshot(data)
}

// FormatSnapshot renders SnapshotData as structured markdown.
func FormatSnapshot(data SnapshotData) string {
	var b strings.Builder
	b.WriteString("# Session Context\n\n")
	b.WriteString("## Goal\n")
	b.WriteString(data.Goal)
	b.WriteString("\n\n## Decisions\n")
	for _, d := range data.Decisions {
		b.WriteString("- ")
		b.WriteString(d)
		b.WriteString("\n")
	}
	b.WriteString("\n## In Progress\n")
	b.WriteString(data.InProgress)
	b.WriteString("\n\n## Next\n")
	b.WriteString(data.Next)
	b.WriteString("\n")
	return b.String()
}
