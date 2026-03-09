package snapshot

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ExtractTranscriptLines reads the last N relevant lines from a .jsonl transcript file.
func ExtractTranscriptLines(path string, maxLines int) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("ctx: %w", err)
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	// Increase buffer size for long JSONL lines
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("ctx: %w", err)
	}

	// Take the last maxLines
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}

	return strings.Join(lines, "\n"), nil
}
