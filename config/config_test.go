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
}

func TestSaveLoad_Roundtrip(t *testing.T) {
	withTempDirs(t)
	path := filepath.Join(t.TempDir(), "config.yml")

	original := &Config{
		Core: CoreConfig{Debug: true},
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
}

func TestLoadPartial_Missing(t *testing.T) {
	pc, err := loadPartial("/nonexistent/path/config.yml")
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if pc.Core.Debug != nil {
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

func TestApplyPartial_NilPreservesDefault(t *testing.T) {
	base := DefaultConfig()
	pc := &partialConfig{}

	result := applyPartial(base, pc)
	if result.Core.Debug != false {
		t.Errorf("expected Debug=false (default), got %v", result.Core.Debug)
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

	// Write global config with debug=true
	globalPath := GlobalConfigPath()
	cfg := &Config{
		Core: CoreConfig{Debug: true},
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
	if sources.Debug != SourceGlobal {
		t.Errorf("Debug source: want global got %v", sources.Debug)
	}
}

func TestEffectiveConfigWithSources_LocalOverrides(t *testing.T) {
	withTempDirs(t)
	projectDir := t.TempDir()

	// Global: debug=false, Local: debug=true
	globalPath := GlobalConfigPath()
	Save(globalPath, &Config{
		Core: CoreConfig{Debug: false},
	})

	localPath := ProjectConfigPath(projectDir)
	os.MkdirAll(filepath.Dir(localPath), 0o755)
	os.WriteFile(localPath, []byte("core:\n  debug: true\n"), 0o644)

	effective, sources, err := EffectiveConfigWithSources(projectDir)
	if err != nil {
		t.Fatalf("EffectiveConfigWithSources: %v", err)
	}
	if effective.Core.Debug != true {
		t.Errorf("Debug: want true got %v", effective.Core.Debug)
	}
	if sources.Debug != SourceLocal {
		t.Errorf("Debug source: want local got %v", sources.Debug)
	}
}

func TestEffectiveConfig_NeitherExists(t *testing.T) {
	withTempDirs(t)
	cfg, err := EffectiveConfig("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return defaults
	if cfg.Core.Debug != false {
		t.Errorf("expected default debug=false, got %v", cfg.Core.Debug)
	}
}
