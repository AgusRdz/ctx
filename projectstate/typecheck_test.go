package projectstate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectTypeChecker_TSConfig(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte("{}"), 0o644)
	if got := DetectTypeChecker(dir); got != "tsc" {
		t.Errorf("want tsc, got %q", got)
	}
}

func TestDetectTypeChecker_GoMod(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644)
	if got := DetectTypeChecker(dir); got != "go build" {
		t.Errorf("want go build, got %q", got)
	}
}

func TestDetectTypeChecker_TSConfigTakesPrecedence(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte("{}"), 0o644)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644)
	if got := DetectTypeChecker(dir); got != "tsc" {
		t.Errorf("want tsc (takes precedence over go.mod), got %q", got)
	}
}

func TestDetectTypeChecker_None(t *testing.T) {
	dir := t.TempDir()
	if got := DetectTypeChecker(dir); got != "none" {
		t.Errorf("want none, got %q", got)
	}
}

func TestDetectTestRunner_Jest(t *testing.T) {
	for _, name := range []string{"jest.config.js", "jest.config.ts", "jest.config.mjs"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			os.WriteFile(filepath.Join(dir, name), []byte(""), 0o644)
			if got := DetectTestRunner(dir); got != "jest" {
				t.Errorf("want jest, got %q", got)
			}
		})
	}
}

func TestDetectTestRunner_Vitest(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "vitest.config.ts"), []byte(""), 0o644)
	if got := DetectTestRunner(dir); got != "vitest" {
		t.Errorf("want vitest, got %q", got)
	}
}

func TestDetectTestRunner_GoTest(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644)
	if got := DetectTestRunner(dir); got != "go test" {
		t.Errorf("want go test, got %q", got)
	}
}

func TestDetectTestRunner_None(t *testing.T) {
	dir := t.TempDir()
	if got := DetectTestRunner(dir); got != "none" {
		t.Errorf("want none, got %q", got)
	}
}

func TestParseTscErrors(t *testing.T) {
	input := `src/services/auth.ts(23,5): error TS2322: Type 'string' is not assignable to type 'number'.
src/services/auth.ts(45,3): error TS2339: Property 'userId' does not exist on type 'Session'.
Found 2 errors.`

	errors := parseTscErrors(input)
	if len(errors) != 2 {
		t.Fatalf("want 2 errors, got %d: %v", len(errors), errors)
	}
	if errors[0] != "L23 src/services/auth.ts  Type 'string' is not assignable to type 'number'." {
		t.Errorf("unexpected error[0]: %q", errors[0])
	}
	if errors[1] != "L45 src/services/auth.ts  Property 'userId' does not exist on type 'Session'." {
		t.Errorf("unexpected error[1]: %q", errors[1])
	}
}

func TestParseTscErrors_NoErrors(t *testing.T) {
	if errors := parseTscErrors(""); len(errors) != 0 {
		t.Errorf("expected no errors, got %v", errors)
	}
}

func TestParseGoBuildErrors(t *testing.T) {
	input := `# example/pkg
./cmd/main.go:12:2: undefined: foo
./cmd/main.go:15:9: cannot use x (variable of type int) as type string`

	errors := parseGoBuildErrors(input)
	if len(errors) != 2 {
		t.Fatalf("want 2 errors, got %d: %v", len(errors), errors)
	}
	if errors[0] != "L12 cmd/main.go  undefined: foo" {
		t.Errorf("unexpected error[0]: %q", errors[0])
	}
	if errors[1] != "L15 cmd/main.go  cannot use x (variable of type int) as type string" {
		t.Errorf("unexpected error[1]: %q", errors[1])
	}
}

func TestParseGoBuildErrors_Dedup(t *testing.T) {
	input := `./cmd/main.go:12:2: undefined: foo
./cmd/main.go:12:2: undefined: foo`

	errors := parseGoBuildErrors(input)
	if len(errors) != 1 {
		t.Errorf("expected dedup to 1 error, got %d: %v", len(errors), errors)
	}
}

func TestParseGoBuildErrors_NoErrors(t *testing.T) {
	if errors := parseGoBuildErrors(""); len(errors) != 0 {
		t.Errorf("expected no errors, got %v", errors)
	}
}
