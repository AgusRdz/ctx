package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
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

// Config holds ctx runtime configuration.
type Config struct {
	Core   CoreConfig   `yaml:"core"`
	Agents AgentsConfig `yaml:"agents"`
}

// CoreConfig holds core settings.
type CoreConfig struct {
	Debug bool `yaml:"debug"`
}

// AgentsConfig holds subagent capture settings.
type AgentsConfig struct {
	Mode          string `yaml:"mode"`            // off | v1 | v2
	InjectOnStart bool   `yaml:"inject_on_start"`
	MaxInject     int    `yaml:"max_inject"`
	StalenessDays int    `yaml:"staleness_days"`
}

// DefaultConfig returns a Config with all defaults populated.
func DefaultConfig() *Config {
	return &Config{
		Core: CoreConfig{
			Debug: false,
		},
		Agents: AgentsConfig{
			Mode:          "off",
			InjectOnStart: true,
			MaxInject:     5,
			StalenessDays: 7,
		},
	}
}

// partialConfig is used for loading config files where fields may be absent.
// Pointer types allow distinguishing "explicitly set to false/0" from "not present".
type partialConfig struct {
	Core struct {
		Debug *bool `yaml:"debug"`
	} `yaml:"core"`
	Agents struct {
		Mode          *string `yaml:"mode"`
		InjectOnStart *bool   `yaml:"inject_on_start"`
		MaxInject     *int    `yaml:"max_inject"`
		StalenessDays *int    `yaml:"staleness_days"`
	} `yaml:"agents"`
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
	if pc.Agents.Mode != nil {
		result.Agents.Mode = *pc.Agents.Mode
	}
	if pc.Agents.InjectOnStart != nil {
		result.Agents.InjectOnStart = *pc.Agents.InjectOnStart
	}
	if pc.Agents.MaxInject != nil {
		result.Agents.MaxInject = *pc.Agents.MaxInject
	}
	if pc.Agents.StalenessDays != nil {
		result.Agents.StalenessDays = *pc.Agents.StalenessDays
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
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	return nil
}
