package projectstate

import (
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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
