package snapshot

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/AgusRdz/ctx/config"
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

// snapshotPath returns the full path to a project's snapshot file.
func snapshotPath(projectDir string) string {
	return filepath.Join(config.DataDir(), ProjectHash(projectDir), "snapshot.md")
}

// Read returns the snapshot content for a project, or empty string if none exists.
func Read(projectDir string) (string, error) {
	data, err := os.ReadFile(snapshotPath(projectDir))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("ctx: %w", err)
	}
	return string(data), nil
}

// Write saves a snapshot for a project, creating directories as needed.
func Write(projectDir string, content string) error {
	p := snapshotPath(projectDir)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	return nil
}

// Clear deletes the snapshot for a project.
func Clear(projectDir string) error {
	p := snapshotPath(projectDir)
	err := os.Remove(p)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	return nil
}
