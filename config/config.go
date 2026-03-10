package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// DataDir returns the base directory for ctx data.
// Windows: %LOCALAPPDATA%\ctx  Linux/macOS: ~/.local/share/ctx
func DataDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "ctx")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "ctx")
}

// LogFile returns the path to the debug log.
func LogFile() string {
	return filepath.Join(DataDir(), "debug.log")
}

// ConfigFile returns the path to the ctx config file.
func ConfigFile() string {
	return filepath.Join(DataDir(), "config.json")
}

// Config holds ctx runtime configuration.
type Config struct {
	Debug bool `json:"debug"`
}

// Load reads the config file, returning defaults if not found or unreadable.
func Load() Config {
	data, err := os.ReadFile(ConfigFile())
	if err != nil {
		return Config{}
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return Config{}
	}
	return c
}

// Save writes the config to disk.
func Save(c Config) error {
	if err := os.MkdirAll(DataDir(), 0o755); err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	if err := os.WriteFile(ConfigFile(), data, 0o644); err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	return nil
}
