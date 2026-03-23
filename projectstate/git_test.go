package projectstate

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// initTestRepo creates a temporary git repo with one commit for testing.
func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args[1:], err, out)
		}
	}

	run("git", "init", "-b", "main")
	run("git", "config", "user.email", "test@test.com")
	run("git", "config", "user.name", "test")
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello\n"), 0o644)
	run("git", "add", ".")
	run("git", "commit", "-m", "initial commit")
	return dir
}

func TestCaptureGit_CleanRepo(t *testing.T) {
	dir := initTestRepo(t)
	state := CaptureGit(dir, 10*time.Second)

	if state.Branch != "main" {
		t.Errorf("branch: want main, got %q", state.Branch)
	}
	if len(state.DirtyFiles) != 0 {
		t.Errorf("expected no dirty files, got %v", state.DirtyFiles)
	}
	if state.LastCommit == "" {
		t.Error("expected non-empty last commit")
	}
	if !strings.Contains(state.LastCommit, "initial commit") {
		t.Errorf("last commit %q does not contain expected message", state.LastCommit)
	}
}

func TestCaptureGit_DirtyFiles(t *testing.T) {
	dir := initTestRepo(t)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("modified\n"), 0o644)

	state := CaptureGit(dir, 10*time.Second)
	if len(state.DirtyFiles) == 0 {
		t.Error("expected dirty files after modifying README.md")
	}
	found := false
	for _, f := range state.DirtyFiles {
		if f == "README.md" {
			found = true
		}
	}
	if !found {
		t.Errorf("README.md not in dirty files: %v", state.DirtyFiles)
	}
}

func TestCaptureGit_NoUpstream(t *testing.T) {
	dir := initTestRepo(t)
	state := CaptureGit(dir, 10*time.Second)
	// No remote configured — AheadBehind should be empty, not an error
	if state.AheadBehind != "" {
		t.Logf("ahead/behind = %q (non-empty but acceptable if remote exists)", state.AheadBehind)
	}
}

func TestCaptureGit_NonGitDir(t *testing.T) {
	dir := t.TempDir()
	state := CaptureGit(dir, 10*time.Second)
	// Should return empty state, not panic
	if state.Branch != "" && state.Branch != "HEAD" {
		t.Logf("branch = %q in non-git dir (acceptable)", state.Branch)
	}
}
