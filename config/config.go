package config

import (
	"os"
	"path/filepath"
	"runtime"
)

// DataDir returns the base directory for ctx data (~/.local/share/ctx).
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
