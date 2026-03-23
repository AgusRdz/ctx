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
	Debug FieldSource
	Mode  FieldSource
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
	if globalPartial.Core.ClaudeTimeoutSecs != nil {
		cfg.Core.ClaudeTimeoutSecs = *globalPartial.Core.ClaudeTimeoutSecs
	}
	if globalPartial.Agents.Mode != nil {
		cfg.Agents.Mode = *globalPartial.Agents.Mode
		sources.Mode = SourceGlobal
	}
	if globalPartial.Agents.Scan.MaxDepth != nil {
		cfg.Agents.Scan.MaxDepth = *globalPartial.Agents.Scan.MaxDepth
	}
	if globalPartial.Agents.Scan.ExtraRootMarkers != nil {
		cfg.Agents.Scan.ExtraRootMarkers = *globalPartial.Agents.Scan.ExtraRootMarkers
	}
	if globalPartial.Agents.Scan.ExtraBoundaryDirs != nil {
		cfg.Agents.Scan.ExtraBoundaryDirs = *globalPartial.Agents.Scan.ExtraBoundaryDirs
	}
	if globalPartial.Agents.Scan.Exclude != nil {
		cfg.Agents.Scan.Exclude = *globalPartial.Agents.Scan.Exclude
	}
	if globalPartial.ProjectState.Enabled != nil {
		cfg.ProjectState.Enabled = *globalPartial.ProjectState.Enabled
	}
	if globalPartial.ProjectState.Git != nil {
		cfg.ProjectState.Git = *globalPartial.ProjectState.Git
	}
	if globalPartial.ProjectState.MaxDirtyFiles != nil {
		cfg.ProjectState.MaxDirtyFiles = *globalPartial.ProjectState.MaxDirtyFiles
	}
	if globalPartial.ProjectState.MaxErrors != nil {
		cfg.ProjectState.MaxErrors = *globalPartial.ProjectState.MaxErrors
	}
	if globalPartial.ProjectState.TypeCheck.Enabled != nil {
		cfg.ProjectState.TypeCheck.Enabled = *globalPartial.ProjectState.TypeCheck.Enabled
	}
	if globalPartial.ProjectState.TypeCheck.TimeoutSeconds != nil {
		cfg.ProjectState.TypeCheck.TimeoutSeconds = *globalPartial.ProjectState.TypeCheck.TimeoutSeconds
	}
	if globalPartial.ProjectState.Tests.Enabled != nil {
		cfg.ProjectState.Tests.Enabled = *globalPartial.ProjectState.Tests.Enabled
	}
	if globalPartial.ProjectState.Tests.TimeoutSeconds != nil {
		cfg.ProjectState.Tests.TimeoutSeconds = *globalPartial.ProjectState.Tests.TimeoutSeconds
	}
	if globalPartial.ProjectState.Tests.MaxFailedNames != nil {
		cfg.ProjectState.Tests.MaxFailedNames = *globalPartial.ProjectState.Tests.MaxFailedNames
	}
	if globalPartial.ProjectState.Tests.Command != nil {
		cfg.ProjectState.Tests.Command = *globalPartial.ProjectState.Tests.Command
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
		if localPartial.Core.ClaudeTimeoutSecs != nil {
			cfg.Core.ClaudeTimeoutSecs = *localPartial.Core.ClaudeTimeoutSecs
		}
		if localPartial.Agents.Mode != nil {
			cfg.Agents.Mode = *localPartial.Agents.Mode
			sources.Mode = SourceLocal
		}
		if localPartial.Agents.Scan.MaxDepth != nil {
			cfg.Agents.Scan.MaxDepth = *localPartial.Agents.Scan.MaxDepth
		}
		if localPartial.Agents.Scan.ExtraRootMarkers != nil {
			cfg.Agents.Scan.ExtraRootMarkers = *localPartial.Agents.Scan.ExtraRootMarkers
		}
		if localPartial.Agents.Scan.ExtraBoundaryDirs != nil {
			cfg.Agents.Scan.ExtraBoundaryDirs = *localPartial.Agents.Scan.ExtraBoundaryDirs
		}
		if localPartial.Agents.Scan.Exclude != nil {
			cfg.Agents.Scan.Exclude = *localPartial.Agents.Scan.Exclude
		}
		if localPartial.ProjectState.Enabled != nil {
			cfg.ProjectState.Enabled = *localPartial.ProjectState.Enabled
		}
		if localPartial.ProjectState.Git != nil {
			cfg.ProjectState.Git = *localPartial.ProjectState.Git
		}
		if localPartial.ProjectState.MaxDirtyFiles != nil {
			cfg.ProjectState.MaxDirtyFiles = *localPartial.ProjectState.MaxDirtyFiles
		}
		if localPartial.ProjectState.MaxErrors != nil {
			cfg.ProjectState.MaxErrors = *localPartial.ProjectState.MaxErrors
		}
		if localPartial.ProjectState.TypeCheck.Enabled != nil {
			cfg.ProjectState.TypeCheck.Enabled = *localPartial.ProjectState.TypeCheck.Enabled
		}
		if localPartial.ProjectState.TypeCheck.TimeoutSeconds != nil {
			cfg.ProjectState.TypeCheck.TimeoutSeconds = *localPartial.ProjectState.TypeCheck.TimeoutSeconds
		}
		if localPartial.ProjectState.Tests.Enabled != nil {
			cfg.ProjectState.Tests.Enabled = *localPartial.ProjectState.Tests.Enabled
		}
		if localPartial.ProjectState.Tests.TimeoutSeconds != nil {
			cfg.ProjectState.Tests.TimeoutSeconds = *localPartial.ProjectState.Tests.TimeoutSeconds
		}
		if localPartial.ProjectState.Tests.MaxFailedNames != nil {
			cfg.ProjectState.Tests.MaxFailedNames = *localPartial.ProjectState.Tests.MaxFailedNames
		}
		if localPartial.ProjectState.Tests.Command != nil {
			cfg.ProjectState.Tests.Command = *localPartial.ProjectState.Tests.Command
		}
	}

	// Migrate legacy v1/v2 mode values to "on"
	if cfg.Agents.Mode == "v1" || cfg.Agents.Mode == "v2" {
		cfg.Agents.Mode = "on"
	}

	return cfg, sources, nil
}
