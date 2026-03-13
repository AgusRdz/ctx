package config

import (
	"os"
	"path/filepath"
	"runtime"
)

// GlobalConfigDir returns the global config directory.
// Windows: %APPDATA%\ctx   Linux/macOS: ~/.config/ctx
// Creates the directory if it doesn't exist.
func GlobalConfigDir() string {
	var dir string
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			home, _ := os.UserHomeDir()
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		dir = filepath.Join(appData, "ctx")
	} else {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config", "ctx")
	}
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

// GlobalConfigPath returns the path to the global config file.
func GlobalConfigPath() string {
	return filepath.Join(GlobalConfigDir(), "config.yml")
}

// ProjectConfigDir returns {projectRoot}/.ctx/
func ProjectConfigDir(projectRoot string) string {
	return filepath.Join(projectRoot, ".ctx")
}

// ProjectConfigPath returns {projectRoot}/.ctx/config.yml
func ProjectConfigPath(projectRoot string) string {
	return filepath.Join(ProjectConfigDir(projectRoot), "config.yml")
}

// EffectiveConfig returns the merged config for a project.
// Resolution order: defaults → global config → local project config.
// Pass empty string for projectRoot to use global config only.
func EffectiveConfig(projectRoot string) (*Config, error) {
	cfg, _, err := EffectiveConfigWithSources(projectRoot)
	return cfg, err
}

// FieldSource indicates where a config field value came from.
type FieldSource int

const (
	SourceDefault FieldSource = iota
	SourceGlobal
	SourceLocal
)

func (s FieldSource) String() string {
	switch s {
	case SourceGlobal:
		return "global"
	case SourceLocal:
		return "local"
	default:
		return "default"
	}
}

// ConfigSources tracks the source of each config field.
type ConfigSources struct {
	Debug         FieldSource
	Mode          FieldSource
	InjectOnStart FieldSource
	MaxInject     FieldSource
	StalenessDays FieldSource
}

// EffectiveConfigWithSources returns the effective config and the source of each field.
func EffectiveConfigWithSources(projectRoot string) (*Config, *ConfigSources, error) {
	cfg := DefaultConfig()
	sources := &ConfigSources{}

	// Apply global config
	globalPartial, err := loadPartial(GlobalConfigPath())
	if err != nil {
		return nil, nil, err
	}
	if globalPartial.Core.Debug != nil {
		cfg.Core.Debug = *globalPartial.Core.Debug
		sources.Debug = SourceGlobal
	}
	if globalPartial.Agents.Mode != nil {
		cfg.Agents.Mode = *globalPartial.Agents.Mode
		sources.Mode = SourceGlobal
	}
	if globalPartial.Agents.InjectOnStart != nil {
		cfg.Agents.InjectOnStart = *globalPartial.Agents.InjectOnStart
		sources.InjectOnStart = SourceGlobal
	}
	if globalPartial.Agents.MaxInject != nil {
		cfg.Agents.MaxInject = *globalPartial.Agents.MaxInject
		sources.MaxInject = SourceGlobal
	}
	if globalPartial.Agents.StalenessDays != nil {
		cfg.Agents.StalenessDays = *globalPartial.Agents.StalenessDays
		sources.StalenessDays = SourceGlobal
	}

	// Apply local project config if it exists
	if projectRoot != "" {
		localPartial, err := loadPartial(ProjectConfigPath(projectRoot))
		if err != nil {
			return nil, nil, err
		}
		if localPartial.Core.Debug != nil {
			cfg.Core.Debug = *localPartial.Core.Debug
			sources.Debug = SourceLocal
		}
		if localPartial.Agents.Mode != nil {
			cfg.Agents.Mode = *localPartial.Agents.Mode
			sources.Mode = SourceLocal
		}
		if localPartial.Agents.InjectOnStart != nil {
			cfg.Agents.InjectOnStart = *localPartial.Agents.InjectOnStart
			sources.InjectOnStart = SourceLocal
		}
		if localPartial.Agents.MaxInject != nil {
			cfg.Agents.MaxInject = *localPartial.Agents.MaxInject
			sources.MaxInject = SourceLocal
		}
		if localPartial.Agents.StalenessDays != nil {
			cfg.Agents.StalenessDays = *localPartial.Agents.StalenessDays
			sources.StalenessDays = SourceLocal
		}
	}

	// Migrate legacy v1/v2 mode values to "on"
	if cfg.Agents.Mode == "v1" || cfg.Agents.Mode == "v2" {
		cfg.Agents.Mode = "on"
	}

	return cfg, sources, nil
}
