package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/AgusRdz/ctx/config"
)

// Uninstall removes ctx completely: hooks, data, binary, and PATH entry.
func Uninstall() error {
	// 1. Remove hooks from settings.json
	if err := Remove(); err != nil {
		fmt.Fprintf(os.Stderr, "ctx: warning: failed to remove hooks: %v\n", err)
	} else {
		fmt.Fprintln(os.Stderr, "ctx: hooks removed")
	}

	// 2. Delete all snapshot data and logs
	dataDir := config.DataDir()
	if err := os.RemoveAll(dataDir); err != nil {
		fmt.Fprintf(os.Stderr, "ctx: warning: failed to remove data dir: %v\n", err)
	} else {
		fmt.Fprintln(os.Stderr, "ctx: data removed")
	}

	// 3. Remove PATH entry on Windows
	if runtime.GOOS == "windows" {
		removeFromWindowsPath()
	}

	// 4. Remove the binary (schedule self-deletion)
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	exe, _ = filepath.EvalSymlinks(exe)
	binDir := filepath.Dir(exe)

	if runtime.GOOS == "windows" {
		// On Windows, can't delete a running binary directly.
		// Schedule deletion via a background cmd process.
		cmd := exec.Command("cmd.exe", "/C",
			fmt.Sprintf("timeout /t 1 /nobreak >nul & del /f /q \"%s\" & rmdir \"%s\" 2>nul", exe, binDir))
		cmd.Start()
		fmt.Fprintln(os.Stderr, "ctx: binary will be removed shortly")
	} else {
		os.Remove(exe)
		os.Remove(binDir) // remove dir if empty
		fmt.Fprintln(os.Stderr, "ctx: binary removed")
	}

	fmt.Fprintln(os.Stderr, "ctx: uninstalled")
	return nil
}

// removeFromWindowsPath removes the ctx install dir from the Windows user PATH.
func removeFromWindowsPath() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	exe, _ = filepath.EvalSymlinks(exe)
	binDir := filepath.Dir(exe)
	winDir := filepath.ToSlash(binDir)
	// Also check with backslashes
	winDirBack := filepath.FromSlash(winDir)

	cmd := exec.Command("powershell.exe", "-NoProfile", "-Command",
		fmt.Sprintf(
			"$cur = [Environment]::GetEnvironmentVariable('Path','User'); "+
				"$parts = $cur -split ';' | Where-Object { $_ -ne '%s' -and $_ -ne '%s' -and $_ -ne '' }; "+
				"[Environment]::SetEnvironmentVariable('Path', ($parts -join ';'), 'User')",
			winDir, winDirBack))
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "ctx: warning: failed to update PATH: %v\n", err)
	} else {
		fmt.Fprintln(os.Stderr, "ctx: removed from PATH")
	}
}

// ConfirmUninstall prompts for confirmation unless --force is passed.
func ConfirmUninstall(args []string) bool {
	for _, a := range args {
		if a == "--force" || a == "-f" {
			return true
		}
	}
	fmt.Fprint(os.Stderr, "ctx: this will remove all hooks, snapshots, and the binary. Continue? [y/N] ")
	var answer string
	fmt.Scanln(&answer)
	return strings.ToLower(answer) == "y" || strings.ToLower(answer) == "yes"
}
