package agents

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/AgusRdz/ctx/config"
)

// AgentSnapshot holds metadata and content for a captured subagent.
type AgentSnapshot struct {
	Name        string
	Type        string // custom | general
	StoppedAt   time.Time
	FinalOutput string
}

// AgentsDir returns the agents directory for a project hash.
func AgentsDir(projectHash string) string {
	return filepath.Join(config.DataDir(), projectHash, "agents")
}

// ArchiveDir returns the archive parent directory for a project hash.
func ArchiveDir(projectHash string) string {
	return filepath.Join(AgentsDir(projectHash), "archive")
}

// AgentSnapshotPath returns the path for an agent's snapshot file.
func AgentSnapshotPath(projectHash, name string) string {
	return filepath.Join(AgentsDir(projectHash), name+".md")
}

// AgentName builds a name for an agent snapshot: {branch}-{YYYYMMDD-HHMMSS}.
// Falls back to no-branch-{timestamp} if git is unavailable.
func AgentName(projectDir string, t time.Time) string {
	branch := gitBranch(projectDir)
	ts := t.UTC().Format("20060102-150405")
	return branch + "-" + ts
}

// gitBranch returns the current git branch name, sanitized for use as a filename
// on all platforms. Only [a-zA-Z0-9._-] are kept; everything else becomes "-".
// The result is truncated to 64 characters to avoid path-length issues.
func gitBranch(projectDir string) string {
	cmd := exec.Command("git", "-C", projectDir, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "no-branch"
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" || branch == "HEAD" {
		return "no-branch"
	}
	return sanitizeBranchName(branch)
}

// sanitizeBranchName replaces any character outside [a-zA-Z0-9._-] with "-"
// and trims leading/trailing separators. Safe for filenames on all OSes.
func sanitizeBranchName(branch string) string {
	var b strings.Builder
	for _, r := range branch {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	result := strings.Trim(b.String(), "-.")
	if result == "" {
		return "no-branch"
	}
	const maxLen = 64
	if len(result) > maxLen {
		result = result[:maxLen]
	}
	return result
}

// FindAgentTranscript searches ~/.claude/projects/ for a .jsonl file matching sessionID.
func FindAgentTranscript(sessionID string) string {
	if sessionID == "" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	claudeProjects := filepath.Join(home, ".claude", "projects")
	entries, err := os.ReadDir(claudeProjects)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		candidate := filepath.Join(claudeProjects, entry.Name(), sessionID+".jsonl")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

// ArchiveCurrentAgents moves all current agent snapshot files to archive/{YYYYMMDD-HHMMSS}/.
// No-ops if there are no current agents.
func ArchiveCurrentAgents(projectHash string) error {
	dir := AgentsDir(projectHash)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("ctx: %w", err)
	}

	var toMove []string
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".md") {
			toMove = append(toMove, entry.Name())
		}
	}
	if len(toMove) == 0 {
		return nil
	}

	archiveSlot := filepath.Join(ArchiveDir(projectHash), time.Now().UTC().Format("20060102-150405"))
	if err := os.MkdirAll(archiveSlot, 0o755); err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	for _, name := range toMove {
		src := filepath.Join(dir, name)
		dst := filepath.Join(archiveSlot, name)
		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("ctx: %w", err)
		}
	}
	return nil
}

// maxClaudeOutputBytes caps output from claude -p to prevent memory exhaustion.
const maxClaudeOutputBytes = 512 * 1024 // 512 KB

// GenerateAgentSummary calls claude -p to produce a short summary of what a sub-agent did.
func GenerateAgentSummary(transcriptLines, projectDir string, timeout time.Duration) (string, error) {
	prompt := fmt.Sprintf(`Summarize what this sub-agent did in 2-4 bullet points. Be concise and specific.
Focus on: what task it was given, what actions it took, what it produced or changed.
No preamble. Respond in plain text with bullet points starting with "- ".

AGENT TRANSCRIPT (last entries):
%s`, transcriptLines)

	cmdCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "claude", "-p", prompt)
	if projectDir != "" {
		cmd.Dir = projectDir
	}
	// CLAUDE_API_KEY is intentionally kept — claude -p needs it.
	cmd.Env = filterEnv(os.Environ(), "CLAUDECODE")
	out, err := runLimited(cmd, cmdCtx, maxClaudeOutputBytes)
	if err != nil {
		return "", fmt.Errorf("ctx: claude -p failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// runLimited starts cmd, reads at most maxBytes of stdout, drains the rest,
// and waits for the process to exit.
func runLimited(cmd *exec.Cmd, cmdCtx context.Context, maxBytes int64) ([]byte, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	out, readErr := io.ReadAll(io.LimitReader(stdout, maxBytes))
	_, _ = io.Copy(io.Discard, stdout)
	waitErr := cmd.Wait()
	if readErr != nil {
		return nil, readErr
	}
	return out, waitErr
}

// WriteAgentSnapshot writes an agent snapshot to disk atomically.
// Writing to a temp file then renaming prevents partial reads from concurrent agents.
func WriteAgentSnapshot(projectHash string, s AgentSnapshot) error {
	dir := AgentsDir(projectHash)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	content := formatAgentSnapshot(s)
	finalPath := AgentSnapshotPath(projectHash, s.Name)
	tmpPath := finalPath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0o600); err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	if err := os.Rename(tmpPath, finalPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("ctx: %w", err)
	}
	return nil
}

// ReadAgentSnapshots reads all non-hidden agent snapshot files for a project.
// Returns snapshots sorted by most recent first.
func ReadAgentSnapshots(projectHash string) ([]AgentSnapshot, error) {
	dir := AgentsDir(projectHash)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ctx: %w", err)
	}

	var snapshots []AgentSnapshot
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		snapshots = append(snapshots, parseAgentSnapshot(string(data)))
	}

	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].StoppedAt.After(snapshots[j].StoppedAt)
	})
	return snapshots, nil
}

// ClearAgentSnapshots removes all agent snapshots for a project.
func ClearAgentSnapshots(projectHash string) error {
	dir := AgentsDir(projectHash)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
				return fmt.Errorf("ctx: %w", err)
			}
		}
	}
	return nil
}

func formatAgentSnapshot(s AgentSnapshot) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Agent: %s\n", s.Name))
	b.WriteString(fmt.Sprintf("_Stopped: %s_\n", s.StoppedAt.UTC().Format("2006-01-02T15:04Z")))
	b.WriteString(fmt.Sprintf("_Type: %s_\n", s.Type))
	b.WriteString("\n")
	b.WriteString("## Summary\n")
	b.WriteString(s.FinalOutput)
	b.WriteString("\n")
	return b.String()
}

func parseAgentSnapshot(content string) AgentSnapshot {
	var s AgentSnapshot
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "# Agent: "):
			s.Name = strings.TrimPrefix(trimmed, "# Agent: ")
		case strings.HasPrefix(trimmed, "_Stopped: ") && strings.HasSuffix(trimmed, "_"):
			inner := strings.TrimPrefix(trimmed, "_Stopped: ")
			inner = strings.TrimSuffix(inner, "_")
			t, _ := time.Parse("2006-01-02T15:04Z", inner)
			s.StoppedAt = t
		case strings.HasPrefix(trimmed, "_Type: ") && strings.HasSuffix(trimmed, "_"):
			s.Type = strings.TrimPrefix(trimmed, "_Type: ")
			s.Type = strings.TrimSuffix(s.Type, "_")
		}
	}

	// Extract summary section — support both "## Summary" and legacy "## Final Output"
	inOutput := false
	var outputLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "## Summary" || trimmed == "## Final Output" {
			inOutput = true
			continue
		}
		if inOutput {
			if strings.HasPrefix(trimmed, "## ") {
				break
			}
			outputLines = append(outputLines, line)
		}
	}
	s.FinalOutput = strings.TrimSpace(strings.Join(outputLines, "\n"))

	if s.Type == "" {
		if strings.HasPrefix(s.Name, "agent-") {
			s.Type = "general"
		} else {
			s.Type = "custom"
		}
	}
	return s
}

// GitRoot returns the git repository root for the given directory.
// Falls back to dir itself if not a git repo or git is unavailable.
func GitRoot(dir string) string {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return dir
	}
	root := strings.TrimSpace(string(out))
	if root == "" {
		return dir
	}
	return root
}

// ReadAllAgentSnapshots reads all agent snapshots for a project including archived ones.
// If since is non-zero, only returns snapshots stopped after that time.
// Results are sorted most recent first.
func ReadAllAgentSnapshots(projectHash string, since time.Time) ([]AgentSnapshot, error) {
	current, err := ReadAgentSnapshots(projectHash)
	if err != nil {
		return nil, err
	}

	archiveBase := filepath.Join(config.DataDir(), projectHash, "agents", "archive")
	archiveDirs, err := os.ReadDir(archiveBase)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("ctx: %w", err)
	}
	for _, entry := range archiveDirs {
		if !entry.IsDir() {
			continue
		}
		slotDir := filepath.Join(archiveBase, entry.Name())
		files, _ := os.ReadDir(slotDir)
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".md") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(slotDir, f.Name()))
			if err != nil {
				continue
			}
			current = append(current, parseAgentSnapshot(string(data)))
		}
	}

	if !since.IsZero() {
		var filtered []AgentSnapshot
		for _, s := range current {
			if s.StoppedAt.After(since) {
				filtered = append(filtered, s)
			}
		}
		current = filtered
	}

	sort.Slice(current, func(i, j int) bool {
		return current[i].StoppedAt.After(current[j].StoppedAt)
	})
	return current, nil
}

// GenerateCombinedSummary calls claude -p to summarize work across multiple agent snapshots.
func GenerateCombinedSummary(snapshots []AgentSnapshot, projectDir string, timeout time.Duration) (string, error) {
	if len(snapshots) == 0 {
		return "", fmt.Errorf("ctx: no snapshots to summarize")
	}

	var b strings.Builder
	for _, s := range snapshots {
		b.WriteString(fmt.Sprintf("## Agent: %s (%s)\n", s.Name, s.StoppedAt.UTC().Format("2006-01-02T15:04Z")))
		b.WriteString(s.FinalOutput)
		b.WriteString("\n\n")
	}

	prompt := fmt.Sprintf(`Summarize what these sub-agents accomplished as a whole.
Group related work. Highlight what changed, what decisions were made, and any blockers.
Be concise. Respond in plain markdown with bullet points.

AGENT SUMMARIES:
%s`, b.String())

	cmdCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "claude", "-p", prompt)
	if projectDir != "" {
		cmd.Dir = projectDir
	}
	// CLAUDE_API_KEY is intentionally kept — claude -p needs it.
	cmd.Env = filterEnv(os.Environ(), "CLAUDECODE")
	out, err := runLimited(cmd, cmdCtx, maxClaudeOutputBytes)
	if err != nil {
		return "", fmt.Errorf("ctx: claude -p failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
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
