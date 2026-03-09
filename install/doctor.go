package install

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/AgusRdz/ctx/config"
)

// Doctor checks the ctx installation and reports issues.
func Doctor() {
	issues := 0

	// 1. Check hooks are installed
	settings, err := readSettings()
	if err != nil {
		fmt.Printf("[!] cannot read Claude Code settings: %v\n", err)
		fmt.Println("    fix: ctx init")
		issues++
	} else {
		hooksRaw, _ := settings["hooks"]
		hooks, _ := hooksRaw.(map[string]interface{})

		hasPC := hasCtxHook(hooks, "PreCompact", "hook precompact")
		hasSS := hasCtxHook(hooks, "SessionStart", "hook session")

		if hasPC && hasSS {
			fmt.Println("[ok] hooks installed (PreCompact + SessionStart)")

			// 2. Check binary path in hook matches current binary
			currentBin := ctxBinaryPath()
			if !hookContainsBinary(hooks, currentBin) {
				fmt.Println("[!] hook points to a different binary path")
				fmt.Printf("    current binary: %s\n", currentBin)
				fmt.Println("    fix: ctx init")
				issues++
			} else {
				fmt.Println("[ok] hook binary path matches current binary")
			}
		} else {
			if !hasPC {
				fmt.Println("[!] PreCompact hook not installed")
			}
			if !hasSS {
				fmt.Println("[!] SessionStart hook not installed")
			}
			fmt.Println("    fix: ctx init")
			issues++
		}
	}

	// 3. Check claude binary is available
	if _, err := exec.LookPath("claude"); err != nil {
		fmt.Println("[!] claude binary not found on PATH")
		fmt.Println("    ctx requires the Claude Code CLI to generate snapshots")
		issues++
	} else {
		fmt.Println("[ok] claude binary found")
	}

	// 4. Check data directory is accessible
	dataDir := config.DataDir()
	fmt.Printf("[ok] data directory: %s\n", dataDir)

	if issues == 0 {
		fmt.Println("\nall good!")
	} else {
		fmt.Printf("\n%d issue(s) found\n", issues)
	}
}

// hookContainsBinary returns true if any hook command in settings contains the binary path.
func hookContainsBinary(hooks map[string]interface{}, binPath string) bool {
	if hooks == nil {
		return false
	}
	data, _ := json.Marshal(hooks)
	return searchString(string(data), binPath)
}
