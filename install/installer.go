package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// settingsPath returns the path to Claude Code's settings.json.
func settingsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "settings.json")
}

// Hook represents a single hook entry.
type Hook struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// HookGroup represents a hook group with an optional matcher.
type HookGroup struct {
	Matcher string `json:"matcher,omitempty"`
	Hooks   []Hook `json:"hooks"`
}

// Settings represents the Claude Code settings.json structure.
// We only care about the hooks field; the rest is preserved as-is.
type Settings map[string]interface{}

// ctxBinaryPath returns the full path to the running ctx binary, with forward slashes.
func ctxBinaryPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "ctx"
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "ctx"
	}
	return filepath.ToSlash(exe)
}

var precompactAutoCmd = ""
var precompactManualCmd = ""
var sessionCmd = ""

func init() {
	bin := ctxBinaryPath()
	precompactAutoCmd = fmt.Sprintf(`"%s" hook precompact --trigger=auto`, bin)
	precompactManualCmd = fmt.Sprintf(`"%s" hook precompact --trigger=manual`, bin)
	sessionCmd = fmt.Sprintf(`"%s" hook session`, bin)
}

// Install adds ctx hooks to Claude Code settings.json.
func Install() error {
	settings, err := readSettings()
	if err != nil {
		return err
	}

	hooks := getOrCreateHooksMap(settings)

	// Add PreCompact hook (both auto and manual matchers)
	hooks["PreCompact"] = []interface{}{
		map[string]interface{}{
			"matcher": "auto",
			"hooks":   []interface{}{map[string]interface{}{"type": "command", "command": precompactAutoCmd}},
		},
		map[string]interface{}{
			"matcher": "manual",
			"hooks":   []interface{}{map[string]interface{}{"type": "command", "command": precompactManualCmd}},
		},
	}

	// Add SessionStart hook
	hooks["SessionStart"] = []interface{}{
		map[string]interface{}{
			"hooks": []interface{}{
				map[string]interface{}{"type": "command", "command": sessionCmd},
			},
		},
	}

	settings["hooks"] = hooks
	return writeSettings(settings)
}

// Remove deletes ctx hooks from Claude Code settings.json.
func Remove() error {
	settings, err := readSettings()
	if err != nil {
		return err
	}

	hooksRaw, ok := settings["hooks"]
	if !ok {
		return nil
	}
	hooks, ok := hooksRaw.(map[string]interface{})
	if !ok {
		return nil
	}

	delete(hooks, "PreCompact")
	delete(hooks, "SessionStart")

	if len(hooks) == 0 {
		delete(settings, "hooks")
	} else {
		settings["hooks"] = hooks
	}

	return writeSettings(settings)
}

// Status checks if ctx hooks are installed and returns a description.
func Status() string {
	settings, err := readSettings()
	if err != nil {
		return fmt.Sprintf("Cannot read settings: %v", err)
	}

	hooksRaw, ok := settings["hooks"]
	if !ok {
		return "Not installed (no hooks section)"
	}
	hooks, ok := hooksRaw.(map[string]interface{})
	if !ok {
		return "Not installed (invalid hooks section)"
	}

	hasPreCompact := hasCtxHook(hooks, "PreCompact", "hook precompact")
	hasSession := hasCtxHook(hooks, "SessionStart", "hook session")

	if hasPreCompact && hasSession {
		return "Installed (PreCompact + SessionStart)"
	}
	if hasPreCompact {
		return "Partially installed (PreCompact only)"
	}
	if hasSession {
		return "Partially installed (SessionStart only)"
	}
	return "Not installed"
}

func hasCtxHook(hooks map[string]interface{}, key string, cmd string) bool {
	raw, ok := hooks[key]
	if !ok {
		return false
	}
	data, _ := json.Marshal(raw)
	return len(data) > 0 && strings.Contains(string(data), cmd)
}

func getOrCreateHooksMap(settings Settings) map[string]interface{} {
	hooksRaw, ok := settings["hooks"]
	if !ok {
		return make(map[string]interface{})
	}
	hooks, ok := hooksRaw.(map[string]interface{})
	if !ok {
		return make(map[string]interface{})
	}
	return hooks
}

func readSettings() (Settings, error) {
	p := settingsPath()
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return make(Settings), nil
	}
	if err != nil {
		return nil, fmt.Errorf("ctx: %w", err)
	}
	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("ctx: parsing settings.json: %w", err)
	}
	return settings, nil
}

func writeSettings(settings Settings) error {
	p := settingsPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(p, data, 0o600); err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	return nil
}
