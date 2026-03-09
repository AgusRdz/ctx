package snapshot

import (
	"encoding/json"
	"fmt"
	"os"
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
	// Clear CLAUDECODE env var to allow nested claude -p invocation
	cmd.Env = filterEnv(os.Environ(), "CLAUDECODE")
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
	// Filter git warnings from DiffStat
	diffStat := filterGitWarnings(ctx.DiffStat)

	// Build In Progress from DiffStat and RecentLog
	var inProgress strings.Builder
	if diffStat != "" {
		inProgress.WriteString(diffStat)
	}
	if ctx.RecentLog != "" {
		if inProgress.Len() > 0 {
			inProgress.WriteString("\n\n")
		}
		inProgress.WriteString("Recent commits:\n")
		inProgress.WriteString(ctx.RecentLog)
	}

	data := SnapshotData{
		Goal:       extractGoalFromMD(ctx.ProjectMD),
		Decisions:  []string{},
		InProgress: inProgress.String(),
		Next:       "Review modified files and continue",
	}
	return FormatSnapshot(data)
}

// filterGitWarnings removes git warning lines from output.
func filterGitWarnings(s string) string {
	if s == "" {
		return ""
	}
	var filtered []string
	for _, line := range strings.Split(s, "\n") {
		if !strings.HasPrefix(line, "warning:") {
			filtered = append(filtered, line)
		}
	}
	return strings.TrimSpace(strings.Join(filtered, "\n"))
}

// extractGoalFromMD tries to extract a meaningful goal from CLAUDE.md content.
func extractGoalFromMD(md string) string {
	if md == "" || md == "Not available" {
		return "Unable to determine (claude -p unavailable)"
	}
	// Look for a "What it does" or first paragraph after the title
	for _, line := range strings.Split(md, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// First non-empty, non-heading line is likely a description
		if len(line) > 120 {
			return line[:120] + "..."
		}
		return line
	}
	return "Unable to determine (claude -p unavailable)"
}

// filterEnv returns a copy of env with the named variable removed.
func filterEnv(env []string, key string) []string {
	prefix := key + "="
	filtered := make([]string, 0, len(env))
	for _, e := range env {
		if !strings.HasPrefix(e, prefix) {
			filtered = append(filtered, e)
		}
	}
	return filtered
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
