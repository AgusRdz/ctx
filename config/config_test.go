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
	if cfg.Agents.InjectOnStart != true {
		t.Errorf("expected InjectOnStart=true, got %v", cfg.Agents.InjectOnStart)
	}
	if cfg.Agents.MaxInject != 5 {
		t.Errorf("expected MaxInject=5, got %d", cfg.Agents.MaxInject)
	}
	if cfg.Agents.StalenessDays != 7 {
		t.Errorf("expected StalenessDays=7, got %d", cfg.Agents.StalenessDays)
	}
}

func TestSaveLoad_Roundtrip(t *testing.T) {
	withTempDirs(t)
	path := filepath.Join(t.TempDir(), "config.yml")

	original := &Config{
		Core:   CoreConfig{Debug: true},
		Agents: AgentsConfig{Mode: "v1", InjectOnStart: true, MaxInject: 3, StalenessDays: 14},
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
	if loaded.Agents.MaxInject != original.Agents.MaxInject {
		t.Errorf("MaxInject: want %d got %d", original.Agents.MaxInject, loaded.Agents.MaxInject)
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
	base := DefaultConfig() // InjectOnStart=true

	// Partial that explicitly sets InjectOnStart=false
	f := false
	pc := &partialConfig{}
	pc.Agents.InjectOnStart = &f

	result := applyPartial(base, pc)
	if result.Agents.InjectOnStart != false {
		t.Error("expected InjectOnStart=false after explicit override")
	}
	// Mode should still be default
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

	// Write global config with debug=true, mode=v1
	globalPath := GlobalConfigPath()
	cfg := &Config{
		Core:   CoreConfig{Debug: true},
		Agents: AgentsConfig{Mode: "v1", InjectOnStart: true, MaxInject: 5, StalenessDays: 7},
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
	if effective.Agents.Mode != "v1" {
		t.Errorf("Mode: want v1 got %q", effective.Agents.Mode)
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
	// Local: mode=v2
	globalPath := GlobalConfigPath()
	Save(globalPath, &Config{
		Core:   CoreConfig{},
		Agents: AgentsConfig{Mode: "off", InjectOnStart: true, MaxInject: 5, StalenessDays: 7},
	})

	localPath := ProjectConfigPath(projectDir)
	os.MkdirAll(filepath.Dir(localPath), 0o755)
	os.WriteFile(localPath, []byte("agents:\n  mode: v2\n"), 0o644)

	effective, sources, err := EffectiveConfigWithSources(projectDir)
	if err != nil {
		t.Fatalf("EffectiveConfigWithSources: %v", err)
	}
	if effective.Agents.Mode != "v2" {
		t.Errorf("Mode: want v2 got %q", effective.Agents.Mode)
	}
	if sources.Mode != SourceLocal {
		t.Errorf("Mode source: want local got %v", sources.Mode)
	}
	// InjectOnStart not in local → from global
	if sources.InjectOnStart != SourceGlobal {
		t.Errorf("InjectOnStart source: want global got %v", sources.InjectOnStart)
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
