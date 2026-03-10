package snapshot

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// transcriptMsg is a partial parse of a Claude Code .jsonl transcript line.
type transcriptMsg struct {
	Message struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

// contentItem represents one block within a message's content array.
type contentItem struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Name  string `json:"name"` // tool_use name
	Input struct {
		Command string `json:"command"`
		Pattern string `json:"pattern"`
		Path    string `json:"path"`
		Prompt  string `json:"prompt"`
	} `json:"input"`
}

// ExtractTranscriptLines reads the last maxLines meaningful entries from a
// .jsonl transcript, returning parsed text instead of raw JSON.
func ExtractTranscriptLines(path string, maxLines int) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("ctx: %w", err)
	}
	defer f.Close()

	var rawLines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	for scanner.Scan() {
		rawLines = append(rawLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("ctx: %w", err)
	}

	// Parse from the end, collect up to maxLines meaningful entries
	var extracted []string
	for i := len(rawLines) - 1; i >= 0 && len(extracted) < maxLines; i-- {
		if line := parseTranscriptLine(rawLines[i]); line != "" {
			extracted = append([]string{line}, extracted...)
		}
	}

	return strings.Join(extracted, "\n"), nil
}

// parseTranscriptLine extracts meaningful text from a single JSONL entry.
// Returns empty string if the entry should be skipped (tool results, noise).
func parseTranscriptLine(raw string) string {
	var msg transcriptMsg
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		return ""
	}
	if msg.Message.Role == "" {
		return ""
	}

	var items []contentItem
	if err := json.Unmarshal(msg.Message.Content, &items); err != nil {
		// Content might be a plain string
		var text string
		if err2 := json.Unmarshal(msg.Message.Content, &text); err2 == nil && text != "" {
			return msg.Message.Role + ": " + transcriptTruncate(text, 200)
		}
		return ""
	}

	var parts []string
	for _, item := range items {
		switch item.Type {
		case "text":
			if t := strings.TrimSpace(item.Text); t != "" {
				parts = append(parts, transcriptTruncate(t, 200))
			}
		case "tool_use":
			detail := item.Name
			switch {
			case item.Input.Command != "":
				detail += "(" + transcriptTruncate(item.Input.Command, 80) + ")"
			case item.Input.Pattern != "":
				detail += "(pattern=" + transcriptTruncate(item.Input.Pattern, 40) + ")"
			case item.Input.Path != "":
				detail += "(path=" + transcriptTruncate(item.Input.Path, 60) + ")"
			case item.Input.Prompt != "":
				detail += "(" + transcriptTruncate(item.Input.Prompt, 60) + ")"
			}
			parts = append(parts, "[tool] "+detail)
		}
		// tool_result and other types are intentionally skipped
	}

	if len(parts) == 0 {
		return ""
	}
	return msg.Message.Role + ": " + strings.Join(parts, " | ")
}

func transcriptTruncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
