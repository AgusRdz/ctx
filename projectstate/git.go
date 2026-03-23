package projectstate

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// CaptureGit runs git commands to collect branch, dirty files, last commit,
// and ahead/behind counts. Fails gracefully on non-git dirs or missing upstream.
func CaptureGit(projectDir string, timeout time.Duration) GitState {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	branch := strings.TrimSpace(runGit(ctx, projectDir, "rev-parse", "--abbrev-ref", "HEAD"))
	lastCommit := strings.TrimSpace(runGit(ctx, projectDir, "log", "-1", "--format=%h %s"))

	var dirtyFiles []string
	if raw := strings.TrimSpace(runGit(ctx, projectDir, "diff", "--name-only", "HEAD")); raw != "" {
		for _, f := range strings.Split(raw, "\n") {
			if f != "" {
				dirtyFiles = append(dirtyFiles, f)
			}
		}
	}

	// HEAD@{u} commands fail when branch has no upstream — treat as "↑0 ↓0"
	ahead := strings.TrimSpace(runGit(ctx, projectDir, "rev-list", "--count", "HEAD@{u}..HEAD"))
	behind := strings.TrimSpace(runGit(ctx, projectDir, "rev-list", "--count", "HEAD..HEAD@{u}"))

	return GitState{
		Branch:      branch,
		DirtyFiles:  dirtyFiles,
		LastCommit:  lastCommit,
		AheadBehind: formatAheadBehind(ahead, behind),
	}
}

// runGit runs a git subcommand and returns stdout. Returns "" on error.
func runGit(ctx context.Context, dir string, args ...string) string {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, _ := cmd.Output()
	return string(out)
}

// formatAheadBehind returns "↑A ↓B" if either count is non-zero, else "".
func formatAheadBehind(ahead, behind string) string {
	a := ahead
	b := behind
	if a == "" {
		a = "0"
	}
	if b == "" {
		b = "0"
	}
	if a == "0" && b == "0" {
		return ""
	}
	return "↑" + a + " ↓" + b
}
