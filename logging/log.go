package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/AgusRdz/ctx/config"
)

// Log appends a formatted entry to the debug log file.
func Log(format string, args ...interface{}) {
	logFile := config.LogFile()
	if err := os.MkdirAll(filepath.Dir(logFile), 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()

	ts := time.Now().UTC().Format(time.RFC3339)
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(f, "[%s] %s\n", ts, msg)
}

// Debug appends a verbose entry to the log, but only when debug mode is enabled.
func Debug(format string, args ...interface{}) {
	if !config.Load().Debug {
		return
	}
	Log("DEBUG "+format, args...)
}
