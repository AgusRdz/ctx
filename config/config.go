package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"gopkg.in/yaml.v3"
)

// DataDir returns the base directory for ctx data.
// Windows: %LOCALAPPDATA%\ctx  Linux/macOS: ~/.local/share/ctx
func DataDir() string {
	if runtime.GOOS == "windows" {
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			home, _ := os.UserHomeDir()
			localAppData = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(localAppData, "ctx")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "ctx")
}

// LogFile returns the path to the debug log.
func LogFile() string {
	return filepath.Join(DataDir(), "debug.log")
}

// Config holds ctx runtime configuration.
type Config struct {
	Core         CoreConfig         `yaml:"core"`
	ProjectState ProjectStateConfig `yaml:"project_state"`
}

// ProjectStateConfig controls project state capture at PreCompact time.
type ProjectStateConfig struct {
	Enabled       bool              `yaml:"enabled"`
	Git           bool              `yaml:"git"`
	TypeCheck     TypeCheckConfig   `yaml:"typecheck"`
	Tests         TestsConfig       `yaml:"tests"`
	MaxDirtyFiles int               `yaml:"max_dirty_files"`
	MaxErrors     int               `yaml:"max_errors"`
}

// TypeCheckConfig controls typecheck capture (tsc / go build / custom command).
type TypeCheckConfig struct {
	Enabled        bool   `yaml:"enabled"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
	Command        string `yaml:"command"` // custom command; overrides auto-detection when set
}

// TestsConfig controls test capture (jest / vitest / go test).
type TestsConfig struct {
	Enabled        bool   `yaml:"enabled"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
	MaxFailedNames int    `yaml:"max_failed_names"`
	Command        string `yaml:"command"` // custom command; overrides auto-detection
}

// CoreConfig holds core settings.
type CoreConfig struct {
	Debug             bool `yaml:"debug"`
	ClaudeTimeoutSecs int  `yaml:"claude_timeout"`   // seconds; 0 = use default (30)
	StaleAfterDays    int  `yaml:"stale_after_days"` // snapshots older than this are flagged stale; 0 disables
}

// DefaultConfig returns a Config with all defaults populated.
func DefaultConfig() *Config {
	return &Config{
		Core: CoreConfig{
			Debug:          false,
			StaleAfterDays: 60,
		},
		ProjectState: ProjectStateConfig{
			Enabled:       true,
			Git:           true,
			TypeCheck:     TypeCheckConfig{Enabled: true, TimeoutSeconds: 20},
			Tests:         TestsConfig{Enabled: false, TimeoutSeconds: 60, MaxFailedNames: 5},
			MaxDirtyFiles: 10,
			MaxErrors:     5,
		},
	}
}

// partialConfig is used for loading config files where fields may be absent.
// Pointer types allow distinguishing "explicitly set to false/0" from "not present".
type partialConfig struct {
	Core struct {
		Debug             *bool `yaml:"debug"`
		ClaudeTimeoutSecs *int  `yaml:"claude_timeout"`
		StaleAfterDays    *int  `yaml:"stale_after_days"`
	} `yaml:"core"`
	ProjectState struct {
		Enabled       *bool `yaml:"enabled"`
		Git           *bool `yaml:"git"`
		MaxDirtyFiles *int  `yaml:"max_dirty_files"`
		MaxErrors     *int  `yaml:"max_errors"`
		TypeCheck     struct {
			Enabled        *bool   `yaml:"enabled"`
			TimeoutSeconds *int    `yaml:"timeout_seconds"`
			Command        *string `yaml:"command"`
		} `yaml:"typecheck"`
		Tests struct {
			Enabled        *bool   `yaml:"enabled"`
			TimeoutSeconds *int    `yaml:"timeout_seconds"`
			MaxFailedNames *int    `yaml:"max_failed_names"`
			Command        *string `yaml:"command"`
		} `yaml:"tests"`
	} `yaml:"project_state"`
}

// loadPartial reads a config file as a partial config.
// Returns empty partial if file doesn't exist, error on parse failure.
func loadPartial(path string) (*partialConfig, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &partialConfig{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ctx: %w", err)
	}
	var pc partialConfig
	if err := yaml.Unmarshal(data, &pc); err != nil {
		return nil, fmt.Errorf("ctx: parsing config %s: %w", path, err)
	}
	return &pc, nil
}

// applyPartial merges non-nil partial fields into a config copy and returns it.
func applyPartial(base *Config, pc *partialConfig) *Config {
	result := *base
	if pc.Core.Debug != nil {
		result.Core.Debug = *pc.Core.Debug
	}
	if pc.Core.ClaudeTimeoutSecs != nil {
		result.Core.ClaudeTimeoutSecs = *pc.Core.ClaudeTimeoutSecs
	}
	if pc.Core.StaleAfterDays != nil {
		result.Core.StaleAfterDays = *pc.Core.StaleAfterDays
	}
	if pc.ProjectState.Enabled != nil {
		result.ProjectState.Enabled = *pc.ProjectState.Enabled
	}
	if pc.ProjectState.Git != nil {
		result.ProjectState.Git = *pc.ProjectState.Git
	}
	if pc.ProjectState.MaxDirtyFiles != nil {
		result.ProjectState.MaxDirtyFiles = *pc.ProjectState.MaxDirtyFiles
	}
	if pc.ProjectState.MaxErrors != nil {
		result.ProjectState.MaxErrors = *pc.ProjectState.MaxErrors
	}
	if pc.ProjectState.TypeCheck.Enabled != nil {
		result.ProjectState.TypeCheck.Enabled = *pc.ProjectState.TypeCheck.Enabled
	}
	if pc.ProjectState.TypeCheck.TimeoutSeconds != nil {
		result.ProjectState.TypeCheck.TimeoutSeconds = *pc.ProjectState.TypeCheck.TimeoutSeconds
	}
	if pc.ProjectState.TypeCheck.Command != nil {
		result.ProjectState.TypeCheck.Command = *pc.ProjectState.TypeCheck.Command
	}
	if pc.ProjectState.Tests.Enabled != nil {
		result.ProjectState.Tests.Enabled = *pc.ProjectState.Tests.Enabled
	}
	if pc.ProjectState.Tests.TimeoutSeconds != nil {
		result.ProjectState.Tests.TimeoutSeconds = *pc.ProjectState.Tests.TimeoutSeconds
	}
	if pc.ProjectState.Tests.MaxFailedNames != nil {
		result.ProjectState.Tests.MaxFailedNames = *pc.ProjectState.Tests.MaxFailedNames
	}
	if pc.ProjectState.Tests.Command != nil {
		result.ProjectState.Tests.Command = *pc.ProjectState.Tests.Command
	}
	return &result
}

// LoadFull reads a config file and returns a complete Config with defaults for missing fields.
// If the file doesn't exist, returns defaults. If parsing fails, returns an error.
func LoadFull(path string) (*Config, error) {
	pc, err := loadPartial(path)
	if err != nil {
		return nil, err
	}
	return applyPartial(DefaultConfig(), pc), nil
}

// Save writes a config to disk in YAML format, creating directories as needed.
func Save(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	return nil
}

// ClaudeTimeout returns the configured claude -p timeout, or 30s if not set.
func ClaudeTimeout(secs int) time.Duration {
	if secs > 0 {
		return time.Duration(secs) * time.Second
	}
	return 30 * time.Second
}
