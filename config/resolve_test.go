package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEffectiveConfigWithSources_ProjectState(t *testing.T) {
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "global.yml")

	os.WriteFile(globalPath, []byte(`
project_state:
  enabled: false
  git: false
  max_dirty_files: 3
  max_errors: 2
  typecheck:
    enabled: false
    timeout_seconds: 10
    command: "mypy src"
  tests:
    enabled: true
    timeout_seconds: 30
    max_failed_names: 3
    command: "pytest -q"
`), 0o600)

	// Temporarily override GlobalConfigPath by loading directly
	pc, err := loadPartial(globalPath)
	if err != nil {
		t.Fatalf("loadPartial: %v", err)
	}
	cfg := applyPartial(DefaultConfig(), pc)

	if cfg.ProjectState.Enabled != false {
		t.Error("expected ProjectState.Enabled=false")
	}
	if cfg.ProjectState.Git != false {
		t.Error("expected ProjectState.Git=false")
	}
	if cfg.ProjectState.MaxDirtyFiles != 3 {
		t.Errorf("expected MaxDirtyFiles=3, got %d", cfg.ProjectState.MaxDirtyFiles)
	}
	if cfg.ProjectState.MaxErrors != 2 {
		t.Errorf("expected MaxErrors=2, got %d", cfg.ProjectState.MaxErrors)
	}
	if cfg.ProjectState.TypeCheck.Enabled != false {
		t.Error("expected TypeCheck.Enabled=false")
	}
	if cfg.ProjectState.TypeCheck.TimeoutSeconds != 10 {
		t.Errorf("expected TypeCheck.TimeoutSeconds=10, got %d", cfg.ProjectState.TypeCheck.TimeoutSeconds)
	}
	if cfg.ProjectState.TypeCheck.Command != "mypy src" {
		t.Errorf("expected TypeCheck.Command=mypy src, got %q", cfg.ProjectState.TypeCheck.Command)
	}
	if cfg.ProjectState.Tests.Enabled != true {
		t.Error("expected Tests.Enabled=true")
	}
	if cfg.ProjectState.Tests.TimeoutSeconds != 30 {
		t.Errorf("expected Tests.TimeoutSeconds=30, got %d", cfg.ProjectState.Tests.TimeoutSeconds)
	}
	if cfg.ProjectState.Tests.MaxFailedNames != 3 {
		t.Errorf("expected Tests.MaxFailedNames=3, got %d", cfg.ProjectState.Tests.MaxFailedNames)
	}
	if cfg.ProjectState.Tests.Command != "pytest -q" {
		t.Errorf("expected Tests.Command=pytest -q, got %q", cfg.ProjectState.Tests.Command)
	}
}
