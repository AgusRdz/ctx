package snapshot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractTranscriptLines_LastN(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "transcript.jsonl")

	content := ""
	for i := 0; i < 30; i++ {
		content += "{\"n\":" + itoa(i+1) + "}\n"
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	result, err := ExtractTranscriptLines(path, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultLines := strings.Split(result, "\n")
	if len(resultLines) != 5 {
		t.Fatalf("expected 5 lines, got %d", len(resultLines))
	}
	// Should contain lines 26-30
	if resultLines[0] != `{"n":26}` {
		t.Errorf("expected first line to be {\"n\":26}, got %s", resultLines[0])
	}
	if resultLines[4] != `{"n":30}` {
		t.Errorf("expected last line to be {\"n\":30}, got %s", resultLines[4])
	}
}

func TestExtractTranscriptLines_FewerThanMax(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "transcript.jsonl")

	content := "{\"a\":1}\n{\"a\":2}\n{\"a\":3}\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	result, err := ExtractTranscriptLines(path, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultLines := strings.Split(result, "\n")
	if len(resultLines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(resultLines))
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
