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
	Debug                  FieldSource
	ProjectStateEnabled    FieldSource
	ProjectStateGit        FieldSource
	ProjectStateMaxDirty   FieldSource
	ProjectStateMaxErrors  FieldSource
	TypeCheckEnabled       FieldSource
	TypeCheckTimeout       FieldSource
	TypeCheckCommand       FieldSource
	TestsEnabled           FieldSource
	TestsTimeout           FieldSource
	TestsMaxFailedNames    FieldSource
	TestsCommand           FieldSource
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
	if globalPartial.ProjectState.Enabled != nil {
		cfg.ProjectState.Enabled = *globalPartial.ProjectState.Enabled
		sources.ProjectStateEnabled = SourceGlobal
	}
	if globalPartial.ProjectState.Git != nil {
		cfg.ProjectState.Git = *globalPartial.ProjectState.Git
		sources.ProjectStateGit = SourceGlobal
	}
	if globalPartial.ProjectState.MaxDirtyFiles != nil {
		cfg.ProjectState.MaxDirtyFiles = *globalPartial.ProjectState.MaxDirtyFiles
		sources.ProjectStateMaxDirty = SourceGlobal
	}
	if globalPartial.ProjectState.MaxErrors != nil {
		cfg.ProjectState.MaxErrors = *globalPartial.ProjectState.MaxErrors
		sources.ProjectStateMaxErrors = SourceGlobal
	}
	if globalPartial.ProjectState.TypeCheck.Enabled != nil {
		cfg.ProjectState.TypeCheck.Enabled = *globalPartial.ProjectState.TypeCheck.Enabled
		sources.TypeCheckEnabled = SourceGlobal
	}
	if globalPartial.ProjectState.TypeCheck.TimeoutSeconds != nil {
		cfg.ProjectState.TypeCheck.TimeoutSeconds = *globalPartial.ProjectState.TypeCheck.TimeoutSeconds
		sources.TypeCheckTimeout = SourceGlobal
	}
	if globalPartial.ProjectState.TypeCheck.Command != nil {
		cfg.ProjectState.TypeCheck.Command = *globalPartial.ProjectState.TypeCheck.Command
		sources.TypeCheckCommand = SourceGlobal
	}
	if globalPartial.ProjectState.Tests.Enabled != nil {
		cfg.ProjectState.Tests.Enabled = *globalPartial.ProjectState.Tests.Enabled
		sources.TestsEnabled = SourceGlobal
	}
	if globalPartial.ProjectState.Tests.TimeoutSeconds != nil {
		cfg.ProjectState.Tests.TimeoutSeconds = *globalPartial.ProjectState.Tests.TimeoutSeconds
		sources.TestsTimeout = SourceGlobal
	}
	if globalPartial.ProjectState.Tests.MaxFailedNames != nil {
		cfg.ProjectState.Tests.MaxFailedNames = *globalPartial.ProjectState.Tests.MaxFailedNames
		sources.TestsMaxFailedNames = SourceGlobal
	}
	if globalPartial.ProjectState.Tests.Command != nil {
		cfg.ProjectState.Tests.Command = *globalPartial.ProjectState.Tests.Command
		sources.TestsCommand = SourceGlobal
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
		if localPartial.ProjectState.Enabled != nil {
			cfg.ProjectState.Enabled = *localPartial.ProjectState.Enabled
			sources.ProjectStateEnabled = SourceLocal
		}
		if localPartial.ProjectState.Git != nil {
			cfg.ProjectState.Git = *localPartial.ProjectState.Git
			sources.ProjectStateGit = SourceLocal
		}
		if localPartial.ProjectState.MaxDirtyFiles != nil {
			cfg.ProjectState.MaxDirtyFiles = *localPartial.ProjectState.MaxDirtyFiles
			sources.ProjectStateMaxDirty = SourceLocal
		}
		if localPartial.ProjectState.MaxErrors != nil {
			cfg.ProjectState.MaxErrors = *localPartial.ProjectState.MaxErrors
			sources.ProjectStateMaxErrors = SourceLocal
		}
		if localPartial.ProjectState.TypeCheck.Enabled != nil {
			cfg.ProjectState.TypeCheck.Enabled = *localPartial.ProjectState.TypeCheck.Enabled
			sources.TypeCheckEnabled = SourceLocal
		}
		if localPartial.ProjectState.TypeCheck.TimeoutSeconds != nil {
			cfg.ProjectState.TypeCheck.TimeoutSeconds = *localPartial.ProjectState.TypeCheck.TimeoutSeconds
			sources.TypeCheckTimeout = SourceLocal
		}
		if localPartial.ProjectState.TypeCheck.Command != nil {
			cfg.ProjectState.TypeCheck.Command = *localPartial.ProjectState.TypeCheck.Command
			sources.TypeCheckCommand = SourceLocal
		}
		if localPartial.ProjectState.Tests.Enabled != nil {
			cfg.ProjectState.Tests.Enabled = *localPartial.ProjectState.Tests.Enabled
			sources.TestsEnabled = SourceLocal
		}
		if localPartial.ProjectState.Tests.TimeoutSeconds != nil {
			cfg.ProjectState.Tests.TimeoutSeconds = *localPartial.ProjectState.Tests.TimeoutSeconds
			sources.TestsTimeout = SourceLocal
		}
		if localPartial.ProjectState.Tests.MaxFailedNames != nil {
			cfg.ProjectState.Tests.MaxFailedNames = *localPartial.ProjectState.Tests.MaxFailedNames
			sources.TestsMaxFailedNames = SourceLocal
		}
		if localPartial.ProjectState.Tests.Command != nil {
			cfg.ProjectState.Tests.Command = *localPartial.ProjectState.Tests.Command
			sources.TestsCommand = SourceLocal
		}
	}

	return cfg, sources, nil
}
