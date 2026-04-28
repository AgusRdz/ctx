package snapshot

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/AgusRdz/ctx/config"
	"github.com/AgusRdz/ctx/logging"
)

// ProjectHash returns the sha256 hex of the absolute project path.
func ProjectHash(projectDir string) string {
	abs, err := filepath.Abs(projectDir)
	if err != nil {
		abs = projectDir
	}
	h := sha256.Sum256([]byte(abs))
	return fmt.Sprintf("%x", h)
}

// BranchForProject returns the current git branch for projectDir,
// or "_" if the directory is not a git repo, git is unavailable,
// or HEAD is detached.
func BranchForProject(projectDir string) string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = projectDir
	out, err := cmd.Output()
	if err != nil {
		return "_"
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" || branch == "HEAD" {
		return "_"
	}
	return sanitizeBranch(branch)
}

// sanitizeBranch replaces filesystem-unsafe characters in a branch name
// with "-" so it can be used as a directory name on all platforms.
func sanitizeBranch(branch string) string {
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "-",
		"?", "-",
		"\"", "-",
		"<", "-",
		">", "-",
		"|", "-",
	)
	s := replacer.Replace(branch)
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-")
}

// snapshotPathForBranch returns the branch-scoped snapshot path.
// Layout: DataDir / SHA256(abs_project_path) / {branch} / snapshot.md
func snapshotPathForBranch(projectDir, branch string) string {
	return filepath.Join(config.DataDir(), ProjectHash(projectDir), branch, "snapshot.md")
}

// legacySnapshotPath returns the pre-branch-aware snapshot path.
// Layout: DataDir / SHA256(abs_project_path) / snapshot.md
func legacySnapshotPath(projectDir string) string {
	return filepath.Join(config.DataDir(), ProjectHash(projectDir), "snapshot.md")
}

// Read returns the snapshot content for a project+branch.
// Falls back to the legacy flat snapshot.md if no branch-scoped snapshot exists
// (migrate-on-read for users upgrading from pre-branch versions).
// Returns empty string (no error) when nothing is found at either path.
func Read(projectDir, branch string) (string, error) {
	data, err := os.ReadFile(snapshotPathForBranch(projectDir, branch))
	if err == nil {
		return string(data), nil
	}
	if !os.IsNotExist(err) {
		return "", fmt.Errorf("ctx: %w", err)
	}

	// Fall back to legacy flat path
	data, err = os.ReadFile(legacySnapshotPath(projectDir))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("ctx: %w", err)
	}
	return string(data), nil
}

// Write saves a snapshot for a project+branch, creating directories as needed.
// path.txt is stored at the hash-dir level (not branch level) for use by List().
func Write(projectDir, branch, content string) error {
	p := snapshotPathForBranch(projectDir, branch)
	branchDir := filepath.Dir(p)
	hashDir := filepath.Dir(branchDir)

	if err := os.MkdirAll(branchDir, 0o700); err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	// path.txt lives at the hash-dir level so List() can find it with one ReadDir pass.
	pathFile := filepath.Join(hashDir, "path.txt")
	if err := os.WriteFile(pathFile, []byte(projectDir), 0o600); err != nil {
		logging.Log("snapshot | WARNING: failed to write path.txt for %s: %v", projectDir, err)
	}
	return nil
}

// Clear deletes the branch-scoped snapshot for a project.
// Other branches' snapshots and path.txt are preserved.
func Clear(projectDir, branch string) error {
	p := snapshotPathForBranch(projectDir, branch)
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("ctx: %w", err)
	}
	// Remove the now-empty branch directory (best-effort).
	_ = os.Remove(filepath.Dir(p))
	return nil
}

// ClearAll deletes the entire snapshot directory for a project (all branches).
func ClearAll(projectDir string) error {
	dir := filepath.Join(config.DataDir(), ProjectHash(projectDir))
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	return nil
}

// ClearStale deletes snapshots older than the given threshold.
// Returns the list of removed snapshots. If a project's hash directory ends
// up with no remaining branch snapshots, the directory and its path.txt are
// removed too.
func ClearStale(threshold time.Duration) ([]SnapshotInfo, error) {
	infos, _, err := List()
	if err != nil {
		return nil, err
	}
	var removed []SnapshotInfo
	for _, info := range infos {
		if info.CapturedAt.IsZero() || time.Since(info.CapturedAt) < threshold {
			continue
		}
		if err := Clear(info.ProjectDir, info.Branch); err != nil {
			return removed, err
		}
		removed = append(removed, info)
	}
	// Best-effort cleanup of now-empty hash directories
	dataDir := config.DataDir()
	entries, err := os.ReadDir(dataDir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			hashDir := filepath.Join(dataDir, entry.Name())
			if hasAnyBranch(hashDir) {
				continue
			}
			_ = os.Remove(filepath.Join(hashDir, "path.txt"))
			_ = os.Remove(hashDir)
		}
	}
	return removed, nil
}

// hasAnyBranch reports whether hashDir still contains a branch subdirectory
// with a snapshot.md inside.
func hasAnyBranch(hashDir string) bool {
	entries, err := os.ReadDir(hashDir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(hashDir, entry.Name(), "snapshot.md")); err == nil {
			return true
		}
	}
	return false
}

// SnapshotInfo holds metadata about a stored snapshot.
type SnapshotInfo struct {
	ProjectDir string
	Branch     string
	Goal       string
	CapturedAt time.Time
}

// List returns info about all stored snapshots across all projects and branches.
// The second return value is the count of legacy snapshots (pre-branch-aware)
// that exist on disk but cannot be listed because they lack path.txt.
func List() ([]SnapshotInfo, int, error) {
	dataDir := config.DataDir()
	entries, err := os.ReadDir(dataDir)
	if os.IsNotExist(err) {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, fmt.Errorf("ctx: %w", err)
	}

	var results []SnapshotInfo
	legacy := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		hashDir := filepath.Join(dataDir, entry.Name())

		// Read project path (at hash-dir level)
		pathData, err := os.ReadFile(filepath.Join(hashDir, "path.txt"))
		if err != nil {
			// Has snapshot.md but no path.txt — legacy entry
			if _, serr := os.Stat(filepath.Join(hashDir, "snapshot.md")); serr == nil {
				legacy++
			}
			continue
		}
		projectDir := strings.TrimSpace(string(pathData))

		// Enumerate branch subdirectories
		branchEntries, err := os.ReadDir(hashDir)
		if err != nil {
			continue
		}
		for _, branchEntry := range branchEntries {
			if !branchEntry.IsDir() {
				continue
			}
			snapFile := filepath.Join(hashDir, branchEntry.Name(), "snapshot.md")
			snapData, err := os.ReadFile(snapFile)
			if err != nil {
				continue
			}
			goal := goalFromSnapshot(string(snapData))
			capturedAt := time.Time{}
			if snapInfo, err := os.Stat(snapFile); err == nil {
				capturedAt = snapInfo.ModTime()
			}
			results = append(results, SnapshotInfo{
				ProjectDir: projectDir,
				Branch:     branchEntry.Name(),
				Goal:       goal,
				CapturedAt: capturedAt,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].CapturedAt.After(results[j].CapturedAt)
	})
	return results, legacy, nil
}

// goalFromSnapshot extracts the goal line from a formatted snapshot.
func goalFromSnapshot(content string) string {
	inGoal := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "## Goal" {
			inGoal = true
			continue
		}
		if inGoal {
			if trimmed == "" || strings.HasPrefix(trimmed, "_") {
				continue // skip blank lines and _Captured:_ lines
			}
			if strings.HasPrefix(trimmed, "##") {
				break // reached next section
			}
			return trimmed
		}
	}
	return "unknown"
}
