package snapshot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildTranscriptLine builds a JSONL line for a user or assistant text message.
func buildTranscriptLine(role, text string) string {
	return `{"message":{"role":"` + role + `","content":[{"type":"text","text":"` + text + `"}]}}`
}

// buildToolUseLine builds a JSONL line for an assistant tool_use message.
func buildToolUseLine(toolName, command string) string {
	return `{"message":{"role":"assistant","content":[{"type":"tool_use","name":"` + toolName + `","input":{"command":"` + command + `"}}]}}`
}

// buildToolResultLine builds a JSONL line for a tool_result (should be skipped).
func buildToolResultLine() string {
	return `{"message":{"role":"tool","content":[{"type":"tool_result","content":"some output"}]}}`
}

func TestExtractTranscriptLines_LastN(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "transcript.jsonl")

	var lines []string
	for i := 1; i <= 10; i++ {
		lines = append(lines, buildTranscriptLine("user", "message "+itoa(i)))
	}

	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	result, err := ExtractTranscriptLines(path, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultLines := strings.Split(strings.TrimSpace(result), "\n")
	if len(resultLines) != 5 {
		t.Fatalf("expected 5 lines, got %d:\n%s", len(resultLines), result)
	}
	if !strings.Contains(resultLines[0], "message 6") {
		t.Errorf("expected first extracted line to contain 'message 6', got: %s", resultLines[0])
	}
	if !strings.Contains(resultLines[4], "message 10") {
		t.Errorf("expected last extracted line to contain 'message 10', got: %s", resultLines[4])
	}
}

func TestExtractTranscriptLines_FewerThanMax(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "transcript.jsonl")

	content := buildTranscriptLine("user", "hello") + "\n" +
		buildTranscriptLine("assistant", "world") + "\n" +
		buildTranscriptLine("user", "done") + "\n"

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	result, err := ExtractTranscriptLines(path, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultLines := strings.Split(strings.TrimSpace(result), "\n")
	if len(resultLines) != 3 {
		t.Fatalf("expected 3 lines, got %d:\n%s", len(resultLines), result)
	}
}

func TestExtractTranscriptLines_SkipsToolResults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "transcript.jsonl")

	content := buildTranscriptLine("user", "run it") + "\n" +
		buildToolUseLine("Bash", "go test ./...") + "\n" +
		buildToolResultLine() + "\n" + // should be skipped
		buildTranscriptLine("assistant", "tests passed") + "\n"

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	result, err := ExtractTranscriptLines(path, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(result, "tool_result") {
		t.Error("tool_result entries should be skipped")
	}
	if !strings.Contains(result, "go test ./...") {
		t.Error("tool_use command should appear in output")
	}
	if !strings.Contains(result, "tests passed") {
		t.Error("assistant text should appear in output")
	}
}

func TestExtractTranscriptLines_NonexistentFile(t *testing.T) {
	_, err := ExtractTranscriptLines("/nonexistent/path/file.jsonl", 10)
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
	if !strings.Contains(err.Error(), "ctx:") {
		t.Errorf("expected error wrapped with 'ctx:', got: %v", err)
	}
}

func TestExtractTranscriptLines_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.jsonl")

	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	result, err := ExtractTranscriptLines(path, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestExtractTranscriptLines_UnparseableLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mixed.jsonl")

	content := "not json at all\n" +
		buildTranscriptLine("user", "valid message") + "\n" +
		"{broken json\n"

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	result, err := ExtractTranscriptLines(path, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "valid message") {
		t.Error("valid message should be extracted")
	}
	// Unparseable lines should be silently skipped
	if strings.Contains(result, "not json") {
		t.Error("unparseable lines should be skipped")
	}
}

func TestParseTranscriptLine_ToolUse(t *testing.T) {
	line := buildToolUseLine("Bash", "git status")
	result := parseTranscriptLine(line)
	if !strings.Contains(result, "[tool] Bash") {
		t.Errorf("expected [tool] Bash in result, got: %s", result)
	}
	if !strings.Contains(result, "git status") {
		t.Errorf("expected command in result, got: %s", result)
	}
}

func TestParseTranscriptLine_Empty(t *testing.T) {
	result := parseTranscriptLine(`{"type":"other"}`)
	if result != "" {
		t.Errorf("expected empty string for entry without role, got: %s", result)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
