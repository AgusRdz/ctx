package updater

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/AgusRdz/ctx/config"
)

const checkInterval = 24 * time.Hour

func lastCheckPath() string {
	return filepath.Join(config.DataDir(), "last-update-check")
}

func pendingUpdatePath() string {
	return filepath.Join(config.DataDir(), "pending-update")
}

// shouldCheck returns true if enough time has passed since the last update check.
func shouldCheck() bool {
	info, err := os.Stat(lastCheckPath())
	if err != nil {
		return true // never checked
	}
	return time.Since(info.ModTime()) > checkInterval
}

// touchLastCheck updates the timestamp of the last check file.
func touchLastCheck() {
	path := lastCheckPath()
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, []byte(time.Now().Format(time.RFC3339)), 0o644)
}

// ApplyPendingUpdate checks for a pending update downloaded in a previous run.
// If found, replaces the current binary. Silent on all errors — never disrupts the command.
func ApplyPendingUpdate(currentVersion string) {
	if IsDev(currentVersion) {
		return
	}

	pending := pendingUpdatePath()
	data, err := os.ReadFile(pending)
	if err != nil {
		return
	}

	// Format: "version\ntmpBinaryPath"
	parts := strings.SplitN(strings.TrimSpace(string(data)), "\n", 2)
	if len(parts) != 2 {
		os.Remove(pending)
		return
	}

	newVersion := parts[0]
	tmpBinary := parts[1]

	info, err := os.Stat(tmpBinary)
	if err != nil || info.Size() < 1024 {
		os.Remove(pending)
		os.Remove(tmpBinary)
		return
	}

	exe, err := os.Executable()
	if err != nil {
		os.Remove(pending)
		return
	}

	if err := replaceBinary(exe, tmpBinary); err != nil {
		os.Remove(pending)
		os.Remove(tmpBinary)
		return
	}

	os.Remove(pending)
	fmt.Fprintf(os.Stderr, "ctx: auto-updated %s -> %s\n", currentVersion, newVersion)
}

// replaceBinary atomically replaces the binary at destPath with srcPath.
func replaceBinary(destPath, srcPath string) error {
	if runtime.GOOS == "windows" {
		// Windows can't replace a running binary — rename dance
		oldPath := destPath + ".old"
		os.Remove(oldPath)
		if err := os.Rename(destPath, oldPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		if err := os.Rename(srcPath, destPath); err != nil {
			os.Rename(oldPath, destPath) // restore
			return err
		}
		os.Remove(oldPath)
		return nil
	}
	// Linux/macOS: rename works even on running binaries
	return os.Rename(srcPath, destPath)
}

// BackgroundCheck spawns a detached subprocess to check for updates and returns
// immediately. Silent on all errors — never disrupts command output.
func BackgroundCheck(currentVersion string) {
	if IsDev(currentVersion) {
		return
	}
	if !shouldCheck() {
		return
	}

	exe, err := os.Executable()
	if err != nil {
		return
	}

	cmd := exec.Command(exe, "--_bg-update", currentVersion)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	if cmd.Start() == nil {
		touchLastCheck()
	}
}

// RunBackgroundUpdate performs the version check and download.
// Called by the subprocess spawned from BackgroundCheck — runs after parent exits.
func RunBackgroundUpdate(currentVersion string) {
	latest, err := latestVersion()
	if err != nil || latest == currentVersion {
		return
	}

	exe, err := os.Executable()
	if err != nil {
		return
	}

	tmpPath := exe + ".new"
	binaryName := buildBinaryName()
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, latest, binaryName)

	if err := download(url, tmpPath); err != nil {
		os.Remove(tmpPath)
		return
	}

	pending := pendingUpdatePath()
	content := fmt.Sprintf("%s\n%s", latest, tmpPath)
	os.WriteFile(pending, []byte(content), 0o644)
}
