package config

import (
	"os"
	"path/filepath"
	"testing"
)

// withTempDirs sets up temporary HOME, LOCALAPPDATA, and APPDATA env vars
// so all config/data paths point to temp directories for the duration of the test.
func withTempDirs(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("LOCALAPPDATA", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("APPDATA", filepath.Join(tmp, "roaming"))
	return tmp
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Core.Debug != false {
		t.Errorf("expected Debug=false, got %v", cfg.Core.Debug)
	}
	if cfg.Agents.Mode != "off" {
		t.Errorf("expected Mode=off, got %q", cfg.Agents.Mode)
	}
}

func TestSaveLoad_Roundtrip(t *testing.T) {
	withTempDirs(t)
	path := filepath.Join(t.TempDir(), "config.yml")

	original := &Config{
		Core:   CoreConfig{Debug: true},
		Agents: AgentsConfig{Mode: "on"},
	}
	if err := Save(path, original); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	pc, err := loadPartial(path)
	if err != nil {
		t.Fatalf("loadPartial failed: %v", err)
	}

	loaded := applyPartial(DefaultConfig(), pc)
	if loaded.Core.Debug != original.Core.Debug {
		t.Errorf("Debug: want %v got %v", original.Core.Debug, loaded.Core.Debug)
	}
	if loaded.Agents.Mode != original.Agents.Mode {
		t.Errorf("Mode: want %q got %q", original.Agents.Mode, loaded.Agents.Mode)
	}
}

func TestLoadPartial_Missing(t *testing.T) {
	pc, err := loadPartial("/nonexistent/path/config.yml")
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if pc.Core.Debug != nil || pc.Agents.Mode != nil {
		t.Error("expected all fields nil for missing file")
	}
}

func TestLoadPartial_InvalidYAML(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "config.yml")
	os.WriteFile(tmp, []byte("not: valid: yaml: {{{"), 0o644)
	_, err := loadPartial(tmp)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestApplyPartial_OnlyOverridesNonNil(t *testing.T) {
	base := DefaultConfig()

	// Partial that explicitly sets Mode=on
	mode := "on"
	pc := &partialConfig{}
	pc.Agents.Mode = &mode

	result := applyPartial(base, pc)
	if result.Agents.Mode != "on" {
		t.Errorf("expected Mode=on after explicit override, got %q", result.Agents.Mode)
	}
}

func TestApplyPartial_NilModePreservesDefault(t *testing.T) {
	base := DefaultConfig()
	pc := &partialConfig{}

	result := applyPartial(base, pc)
	if result.Agents.Mode != "off" {
		t.Errorf("expected Mode=off (default), got %q", result.Agents.Mode)
	}
}

func TestSave_CreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "dir")
	path := filepath.Join(dir, "config.yml")
	if err := Save(path, DefaultConfig()); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("config.yml not created: %v", err)
	}
}

func TestEffectiveConfigWithSources_GlobalOnly(t *testing.T) {
	withTempDirs(t)

	// Write global config with debug=true, mode=on
	globalPath := GlobalConfigPath()
	cfg := &Config{
		Core:   CoreConfig{Debug: true},
		Agents: AgentsConfig{Mode: "on"},
	}
	if err := Save(globalPath, cfg); err != nil {
		t.Fatalf("Save global: %v", err)
	}

	effective, sources, err := EffectiveConfigWithSources("")
	if err != nil {
		t.Fatalf("EffectiveConfigWithSources: %v", err)
	}
	if effective.Core.Debug != true {
		t.Errorf("Debug: want true got %v", effective.Core.Debug)
	}
	if effective.Agents.Mode != "on" {
		t.Errorf("Mode: want on got %q", effective.Agents.Mode)
	}
	if sources.Debug != SourceGlobal {
		t.Errorf("Debug source: want global got %v", sources.Debug)
	}
	if sources.Mode != SourceGlobal {
		t.Errorf("Mode source: want global got %v", sources.Mode)
	}
}

func TestEffectiveConfigWithSources_LocalOverrides(t *testing.T) {
	withTempDirs(t)
	projectDir := t.TempDir()

	// Global: mode=off
	// Local: mode=on
	globalPath := GlobalConfigPath()
	Save(globalPath, &Config{
		Core:   CoreConfig{},
		Agents: AgentsConfig{Mode: "off"},
	})

	localPath := ProjectConfigPath(projectDir)
	os.MkdirAll(filepath.Dir(localPath), 0o755)
	os.WriteFile(localPath, []byte("agents:\n  mode: on\n"), 0o644)

	effective, sources, err := EffectiveConfigWithSources(projectDir)
	if err != nil {
		t.Fatalf("EffectiveConfigWithSources: %v", err)
	}
	if effective.Agents.Mode != "on" {
		t.Errorf("Mode: want on got %q", effective.Agents.Mode)
	}
	if sources.Mode != SourceLocal {
		t.Errorf("Mode source: want local got %v", sources.Mode)
	}
}

func TestEffectiveConfig_NeitherExists(t *testing.T) {
	withTempDirs(t)
	cfg, err := EffectiveConfig("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return defaults
	if cfg.Agents.Mode != "off" {
		t.Errorf("expected default mode=off, got %q", cfg.Agents.Mode)
	}
}
