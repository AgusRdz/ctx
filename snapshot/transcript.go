package snapshot

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// compressLines groups consecutive lines with the same fingerprint, showing
// a representative line with a repeat count. Reduces token waste from
// repetitive tool calls before sending context to claude -p.
func compressLines(lines []string) []string {
	if len(lines) <= 5 {
		return lines
	}
	type run struct {
		line  string
		count int
		fp    string
	}
	var runs []run
	for _, line := range lines {
		fp := lineFingerprint(line)
		if len(runs) > 0 && runs[len(runs)-1].fp == fp {
			runs[len(runs)-1].count++
		} else {
			runs = append(runs, run{line, 1, fp})
		}
	}
	result := make([]string, 0, len(runs))
	for _, r := range runs {
		if r.count > 1 {
			result = append(result, fmt.Sprintf("%s (x%d)", r.line, r.count))
		} else {
			result = append(result, r.line)
		}
	}
	return result
}

// lineFingerprint returns a normalized key for grouping similar transcript lines.
// Bash tool calls are grouped by command prefix; other tools by name alone;
// text messages by their first 60 characters.
func lineFingerprint(line string) string {
	const toolPrefix = "[tool] "
	if idx := strings.Index(line, toolPrefix); idx != -1 {
		rest := line[idx+len(toolPrefix):]
		paren := strings.Index(rest, "(")
		if paren == -1 {
			return rest
		}
		toolName := rest[:paren]
		if toolName == "Bash" {
			arg := rest[paren:]
			if len(arg) > 30 {
				arg = arg[:30]
			}
			return toolPrefix + "Bash" + arg
		}
		return toolPrefix + toolName
	}
	if len(line) > 60 {
		return line[:60]
	}
	return line
}

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

// maxTranscriptReadBytes is the maximum number of bytes read from the tail of
// a transcript file. For large files we seek to the end and read backwards,
// so we never load the entire file into memory.
const maxTranscriptReadBytes = 10 * 1024 * 1024 // 10 MB

// ExtractTranscriptLines reads the last maxLines meaningful entries from a
// .jsonl transcript, returning parsed text instead of raw JSON.
// For large files only the last maxTranscriptReadBytes are examined.
func ExtractTranscriptLines(path string, maxLines int) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("ctx: %w", err)
	}
	defer f.Close()

	// For large transcripts seek to the tail to avoid reading the whole file.
	if info, err := f.Stat(); err == nil && info.Size() > maxTranscriptReadBytes {
		if _, err := f.Seek(-maxTranscriptReadBytes, io.SeekEnd); err != nil {
			return "", fmt.Errorf("ctx: %w", err)
		}
		// The seek may land mid-line; the first partial line will fail JSON
		// parsing and be silently skipped by parseTranscriptLine.
	}

	var rawLines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	for scanner.Scan() {
		rawLines = append(rawLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("ctx: %w", err)
	}

	// Extract up to 3x maxLines for better compression coverage.
	extractLimit := maxLines * 3
	var extracted []string
	for i := len(rawLines) - 1; i >= 0 && len(extracted) < extractLimit; i-- {
		if line := parseTranscriptLine(rawLines[i]); line != "" {
			extracted = append([]string{line}, extracted...)
		}
	}

	// Compress repetitive lines, then limit to maxLines.
	compressed := compressLines(extracted)
	if len(compressed) > maxLines {
		compressed = compressed[len(compressed)-maxLines:]
	}

	return strings.Join(compressed, "\n"), nil
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
