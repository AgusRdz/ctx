package snapshot

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// maxClaudeOutputBytes caps the output read from claude -p to prevent memory exhaustion.
const maxClaudeOutputBytes = 512 * 1024 // 512 KB

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
	Goal       string    `json:"goal"`
	Decisions  []string  `json:"decisions"`
	InProgress string    `json:"in_progress"`
	Next       string    `json:"next"`
	CapturedAt time.Time `json:"-"`
}

// Generate calls claude -p to produce a semantic snapshot from collected context
// and transcript lines. Returns formatted markdown.
func Generate(ctx Context, transcriptLines string, timeout time.Duration) (string, error) {
	prompt := fmt.Sprintf(promptTemplate, ctx.DiffStat, ctx.ProjectMD, transcriptLines)

	cmdCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "claude", "-p", prompt)
	cmd.Dir = ctx.ProjectDir
	// Clear CLAUDECODE to allow nested claude -p invocation.
	// CLAUDE_API_KEY is intentionally left in the environment — claude -p needs it.
	cmd.Env = filterEnv(os.Environ(), "CLAUDECODE")

	out, err := runLimited(cmd, cmdCtx, maxClaudeOutputBytes)
	if err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("ctx: claude -p timed out after %s", timeout)
		}
		return "", fmt.Errorf("ctx: claude -p failed: %w", err)
	}

	// Parse the JSON response — strip markdown code fences if present
	raw := stripCodeFences(strings.TrimSpace(string(out)))
	var data SnapshotData
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return "", fmt.Errorf("ctx: failed to parse claude response: %w", err)
	}

	data.CapturedAt = time.Now().UTC()
	return FormatSnapshot(data), nil
}

// GenerateFallback creates a deterministic snapshot without calling claude -p.
func GenerateFallback(ctx Context) string {
	// Filter git warnings from DiffStat
	diffStat := filterGitWarnings(ctx.DiffStat)

	// Build decisions from recent commits
	var decisions []string
	if ctx.RecentLog != "" {
		for _, line := range strings.Split(ctx.RecentLog, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, " ", 2)
			if len(parts) == 2 && parts[1] != "" {
				decisions = append(decisions, parts[1])
			}
		}
	}

	// Build In Progress from DiffStat
	inProgress := strings.TrimSpace(diffStat)
	if inProgress == "" && ctx.RecentLog != "" {
		inProgress = "See recent commits above"
	}

	data := SnapshotData{
		Goal:       inferGoal(ctx),
		Decisions:  decisions,
		InProgress: inProgress,
		Next:       "Review modified files and continue",
		CapturedAt: time.Now().UTC(),
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

// inferGoal derives the best available goal from project context without claude -p.
// Priority: CLAUDE.md description > latest commit message > project directory name.
func inferGoal(ctx Context) string {
	// Try CLAUDE.md first
	if ctx.ProjectMD != "" && ctx.ProjectMD != "Not available" {
		for _, line := range strings.Split(ctx.ProjectMD, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if len(line) > 120 {
				return line[:120] + "..."
			}
			return line
		}
	}

	// Fall back to most recent commit message (strip the hash prefix)
	if ctx.RecentLog != "" {
		first := strings.SplitN(ctx.RecentLog, "\n", 2)[0]
		parts := strings.SplitN(first, " ", 2)
		if len(parts) == 2 && parts[1] != "" {
			project := filepath.Base(ctx.ProjectDir)
			return project + ": " + parts[1]
		}
	}

	// Last resort: project directory name
	return filepath.Base(ctx.ProjectDir) + " development"
}

// stripCodeFences removes markdown code block delimiters (```json ... ``` or ``` ... ```)
// that claude -p sometimes wraps around JSON responses.
// If the stripped result doesn't look like JSON, the original is returned unchanged
// so the caller gets a useful parse error rather than silent garbage.
func stripCodeFences(s string) string {
	original := s
	// Remove opening fence: ```json or ```
	if i := strings.Index(s, "```"); i != -1 {
		end := strings.Index(s, "\n")
		if end > i {
			s = s[end+1:]
		}
	}
	// Remove closing fence
	if i := strings.LastIndex(s, "```"); i != -1 {
		s = s[:i]
	}
	stripped := strings.TrimSpace(s)
	// Sanity guard: if result is too short or doesn't look like JSON, return original.
	if len(stripped) < 10 || !strings.HasPrefix(stripped, "{") {
		return original
	}
	return stripped
}

// runLimited starts cmd, reads at most maxBytes of stdout, drains the rest,
// and waits for the process to exit. Prevents memory exhaustion from runaway output.
func runLimited(cmd *exec.Cmd, cmdCtx context.Context, maxBytes int64) ([]byte, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	out, readErr := io.ReadAll(io.LimitReader(stdout, maxBytes))
	// Drain remaining output so the process is not blocked on a full pipe buffer.
	_, _ = io.Copy(io.Discard, stdout)
	waitErr := cmd.Wait()
	if readErr != nil {
		return nil, readErr
	}
	return out, waitErr
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

// FormatSnapshot renders SnapshotData as structured markdown within the token budget.
func FormatSnapshot(data SnapshotData) string {
	// Enforce field-level token budget
	goal := truncateField(data.Goal, 120)

	decisions := data.Decisions
	if len(decisions) > 5 {
		decisions = decisions[:5]
	}

	inProgress := truncateField(data.InProgress, 400)
	next := truncateField(data.Next, 150)

	captured := data.CapturedAt
	if captured.IsZero() {
		captured = time.Now().UTC()
	}

	var b strings.Builder
	b.WriteString("# Session Context\n\n")
	b.WriteString(fmt.Sprintf("_Captured: %s_\n\n", captured.Format("2006-01-02T15:04Z")))
	b.WriteString("## Goal\n")
	b.WriteString(goal)
	b.WriteString("\n\n## Decisions\n")
	for _, d := range decisions {
		b.WriteString("- ")
		b.WriteString(truncateField(d, 100))
		b.WriteString("\n")
	}
	b.WriteString("\n## In Progress\n")
	b.WriteString(inProgress)
	b.WriteString("\n\n## Next\n")
	b.WriteString(next)
	b.WriteString("\n")
	return b.String()
}

// truncateField limits a string to n runes, appending "..." if truncated.
func truncateField(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}
