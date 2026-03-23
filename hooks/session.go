package hooks

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/AgusRdz/ctx/logging"
	"github.com/AgusRdz/ctx/snapshot"
)

const staleAgeThreshold = 7 * 24 * time.Hour

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
// If agents mode is enabled, appends the agent activity block.
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
		if warning := stalenessWarning(content, projectDir); warning != "" {
			content = fmt.Sprintf("> %s\n\n", warning) + content
		}
		fmt.Print(content)
		logging.Log("session | project=%s | snapshot=found", projectDir)
	}

	return nil
}

// stalenessWarning returns a warning string if the snapshot appears stale, empty if fresh.
// Prefers commit-based detection; falls back to age-only if no Project State section exists.
func stalenessWarning(content, projectDir string) string {
	age := snapshotAge(content)
	snapshotHash := extractSnapshotCommit(content)

	if snapshotHash != "" {
		currentHash := gitShortHead(projectDir)
		if currentHash != "" && currentHash != snapshotHash {
			ahead := gitCommitsAhead(projectDir, snapshotHash)
			msg := fmt.Sprintf("⚠ Snapshot may be stale — captured at %s, current HEAD is %s", snapshotHash, currentHash)
			var details []string
			if ahead > 0 {
				if ahead == 1 {
					details = append(details, "1 commit ahead")
				} else {
					details = append(details, fmt.Sprintf("%d commits ahead", ahead))
				}
			}
			if age >= 24*time.Hour {
				details = append(details, formatAge(age))
			}
			if len(details) > 0 {
				msg += " (" + strings.Join(details, ", ") + ")"
			}
			return msg + "."
		}
	}

	// Fall back to age-only if no commit info available
	if age > staleAgeThreshold {
		days := int(age.Hours() / 24)
		return fmt.Sprintf("⚠ Snapshot is %d days old — run /compact to refresh.", days)
	}

	return ""
}

// snapshotAge parses the _Captured:_ timestamp from a snapshot and returns its age.
// Returns 0 if the timestamp cannot be parsed.
func snapshotAge(content string) time.Duration {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "_Captured:") && strings.HasSuffix(trimmed, "_") {
			inner := strings.TrimPrefix(trimmed, "_Captured: ")
			inner = strings.TrimSuffix(inner, "_")
			t, err := time.Parse("2006-01-02T15:04Z", inner)
			if err == nil {
				return time.Since(t)
			}
		}
	}
	return 0
}

// extractSnapshotCommit parses the last commit hash from the ## Project State section.
// Looks for the pattern "last: <hash> <message>" on the Git: line.
// Returns empty string if not found.
func extractSnapshotCommit(content string) string {
	for _, line := range strings.Split(content, "\n") {
		if idx := strings.Index(line, "last: "); idx >= 0 {
			rest := strings.TrimSpace(line[idx+6:])
			fields := strings.Fields(rest)
			if len(fields) > 0 {
				return fields[0]
			}
		}
	}
	return ""
}

// gitShortHead returns the short commit hash of HEAD for the given directory.
// Returns empty string on error (not a git repo, no commits, etc.).
func gitShortHead(projectDir string) string {
	out, err := exec.Command("git", "-C", projectDir, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// gitCommitsAhead returns the number of commits from fromHash to HEAD.
// Returns 0 on error or if fromHash is not found.
func gitCommitsAhead(projectDir, fromHash string) int {
	out, err := exec.Command("git", "-C", projectDir, "rev-list", "--count", fromHash+"..HEAD").Output()
	if err != nil {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return n
}

// formatAge returns a human-readable age string like "2 days ago" or "3 hours ago".
func formatAge(d time.Duration) string {
	if d >= 24*time.Hour {
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
	hours := int(d.Hours())
	if hours == 1 {
		return "1 hour ago"
	}
	return fmt.Sprintf("%d hours ago", hours)
}
