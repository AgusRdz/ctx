package hooks

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsValidTranscriptPath_Valid(t *testing.T) {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".claude", "projects", "abc", "session.jsonl")
	if !isValidTranscriptPath(path) {
		t.Errorf("expected valid path to be accepted: %s", path)
	}
}

func TestIsValidTranscriptPath_OutsideClaude(t *testing.T) {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, "other", "session.jsonl")
	if isValidTranscriptPath(path) {
		t.Errorf("expected path outside ~/.claude to be rejected: %s", path)
	}
}

func TestIsValidTranscriptPath_TraversalAttempt(t *testing.T) {
	home, _ := os.UserHomeDir()
	// Attempt to escape via ../
	path := filepath.Join(home, ".claude", "..", "evil.jsonl")
	if isValidTranscriptPath(path) {
		t.Errorf("expected traversal path to be rejected: %s", path)
	}
}

func TestIsValidTranscriptPath_WrongExtension(t *testing.T) {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".claude", "session.json")
	if isValidTranscriptPath(path) {
		t.Errorf("expected non-.jsonl path to be rejected: %s", path)
	}
}

func TestIsValidTranscriptPath_ClaudePrefixBypass(t *testing.T) {
	home, _ := os.UserHomeDir()
	// ~/.claudeevil/... must not match ~/.claude/
	path := filepath.Join(home, ".claudeevil", "session.jsonl")
	if isValidTranscriptPath(path) {
		t.Errorf("expected ~/.claudeevil path to be rejected: %s", path)
	}
}

func TestIsValidTranscriptPath_Empty(t *testing.T) {
	if isValidTranscriptPath("") {
		t.Error("expected empty path to be rejected")
	}
}

func TestSanitizeForLog(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"normal", "normal"},
		{"has\nnewline", "has newline"},
		{"has\r\nwindows", "has  windows"},
		{"", ""},
	}
	for _, c := range cases {
		got := sanitizeForLog(c.input)
		if got != c.want {
			t.Errorf("sanitizeForLog(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}
