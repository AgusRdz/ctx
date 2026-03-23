package projectstate

import (
	"fmt"
	"os"
	"path/filepath"
)

// DetectTypeChecker returns the typecheck tool for the project root.
// Uses root-level config files only (no subdirectory scanning).
// Returns "tsc", "go build", or "none".
func DetectTypeChecker(projectDir string) string {
	if fileExists(filepath.Join(projectDir, "tsconfig.json")) {
		return "tsc"
	}
	if fileExists(filepath.Join(projectDir, "go.mod")) {
		return "go build"
	}
	return "none"
}

// DetectTestRunner returns the test runner for the project root.
// Returns "jest", "vitest", "go test", or "none".
func DetectTestRunner(projectDir string) string {
	for _, name := range []string{
		"jest.config.js", "jest.config.ts", "jest.config.mjs", "jest.config.cjs",
	} {
		if fileExists(filepath.Join(projectDir, name)) {
			return "jest"
		}
	}
	for _, name := range []string{
		"vitest.config.js", "vitest.config.ts", "vitest.config.mjs",
	} {
		if fileExists(filepath.Join(projectDir, name)) {
			return "vitest"
		}
	}
	if fileExists(filepath.Join(projectDir, "go.mod")) {
		return "go test"
	}
	return "none"
}

// MonorepoNote returns a non-empty string if subdirectories contain additional
// config files of the same type, indicating a monorepo. ctx uses root-level
// config only — the note informs users that subdirectory configs are ignored.
func MonorepoNote(projectDir, tool string) string {
	if tool == "none" {
		return ""
	}
	var marker string
	switch tool {
	case "tsc":
		marker = "tsconfig.json"
	case "go build", "go test":
		marker = "go.mod"
	default:
		return ""
	}

	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return ""
	}
	extra := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "node_modules" || name == "vendor" || name[0] == '.' {
			continue
		}
		if fileExists(filepath.Join(projectDir, name, marker)) {
			extra++
		}
	}
	if extra == 0 {
		return ""
	}
	return fmt.Sprintf("root only (%d more %s in subdirs — ignored)", extra, marker)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
