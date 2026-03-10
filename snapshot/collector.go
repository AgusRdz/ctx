package snapshot

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// chopAvailable checks if chop is on PATH.
func chopAvailable() bool {
	_, err := exec.LookPath("chop")
	return err == nil
}

// runCommand executes a command, optionally prefixed with chop.
// Returns stdout as string. On error, returns empty string.
func runCommand(dir string, args ...string) string {
	if chopAvailable() {
		args = append([]string{"chop"}, args...)
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	out, _ := cmd.Output()
	return strings.TrimSpace(string(out))
}

// isGitRepo checks if the directory is inside a git repository.
func isGitRepo(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = dir
	out, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

// CollectContext gathers project context for snapshot generation.
type Context struct {
	DiffStat   string
	RecentLog  string
	ProjectMD  string
	ProjectDir string
}

// Collect gathers git and file info from the project directory.
func Collect(projectDir string) Context {
	ctx := Context{ProjectDir: projectDir}

	// Read CLAUDE.md — extract first two sections (up to 1000 chars)
	claudeMD := filepath.Join(projectDir, "CLAUDE.md")
	if data, err := os.ReadFile(claudeMD); err == nil {
		ctx.ProjectMD = extractFirstSections(string(data), 2, 1000)
	} else {
		ctx.ProjectMD = "Not available"
	}

	if isGitRepo(projectDir) {
		ctx.DiffStat = runCommand(projectDir, "git", "diff", "--stat")
		ctx.RecentLog = runCommand(projectDir, "git", "log", "-5", "--oneline")
	} else {
		// Fallback: list recently modified files
		ctx.DiffStat = listRecentFiles(projectDir)
		ctx.RecentLog = ""
	}

	return ctx
}

// extractFirstSections returns up to maxSections "##" sections from a markdown
// file, capped at maxChars. This captures the project overview without
// drowning the prompt in implementation details.
func extractFirstSections(content string, maxSections, maxChars int) string {
	var result []string
	sectionCount := 0
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "## ") {
			sectionCount++
			if sectionCount > maxSections {
				break
			}
		}
		result = append(result, line)
	}
	s := strings.TrimSpace(strings.Join(result, "\n"))
	if len(s) > maxChars {
		return s[:maxChars] + "..."
	}
	return s
}

// listRecentFiles returns a simple listing of files in the project root.
func listRecentFiles(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var lines []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		lines = append(lines, e.Name())
		if len(lines) >= 20 {
			break
		}
	}
	return strings.Join(lines, "\n")
}
