package config

import (
	"os"
	"path/filepath"
	"testing"
)

// withTempDataDir overrides the LOCALAPPDATA / HOME env so DataDir() points
// to a temp directory for the duration of the test.
func withTempDataDir(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	// Both paths use env vars; point them to the temp dir.
	t.Setenv("LOCALAPPDATA", tmp)
	t.Setenv("HOME", tmp)
	return tmp
}

func TestLoad_Defaults(t *testing.T) {
	withTempDataDir(t)

	c := Load()
	if c.Debug != false {
		t.Errorf("expected Debug=false by default, got %v", c.Debug)
	}
}

func TestSaveLoad_Roundtrip(t *testing.T) {
	withTempDataDir(t)

	if err := os.MkdirAll(DataDir(), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	original := Config{Debug: true}
	if err := Save(original); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded := Load()
	if loaded.Debug != original.Debug {
		t.Errorf("expected Debug=%v, got %v", original.Debug, loaded.Debug)
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	withTempDataDir(t)

	if err := os.MkdirAll(DataDir(), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(ConfigFile(), []byte("not json"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Should return defaults, not panic
	c := Load()
	if c.Debug != false {
		t.Errorf("expected default on invalid JSON, got Debug=%v", c.Debug)
	}
}

func TestSave_CreatesDirectory(t *testing.T) {
	withTempDataDir(t)
	// DataDir does not exist yet — Save should create it

	if err := Save(Config{Debug: false}); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(DataDir(), "config.json")); err != nil {
		t.Errorf("config.json not created: %v", err)
	}
}

func TestConfigFile_InDataDir(t *testing.T) {
	withTempDataDir(t)
	cf := ConfigFile()
	dd := DataDir()
	if filepath.Dir(cf) != dd {
		t.Errorf("ConfigFile() %q should be inside DataDir() %q", cf, dd)
	}
}
