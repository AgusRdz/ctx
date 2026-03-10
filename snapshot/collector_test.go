package snapshot

import (
	"strings"
	"testing"
)

func TestExtractFirstSections_TwoSections(t *testing.T) {
	md := `# Project

## What it does
Preserves context across compactions.

## Architecture
- PreCompact hook
- SessionStart hook

## File Structure
- cmd/main.go
- hooks/
`
	result := extractFirstSections(md, 2, 1000)

	if !strings.Contains(result, "What it does") {
		t.Error("first section header missing")
	}
	if !strings.Contains(result, "Architecture") {
		t.Error("second section header missing")
	}
	if strings.Contains(result, "File Structure") {
		t.Error("third section should be excluded")
	}
}

func TestExtractFirstSections_NoSections(t *testing.T) {
	md := "Just some text\nno headings here\nand more text"
	result := extractFirstSections(md, 2, 1000)
	if result != md {
		t.Errorf("expected full content when no ## sections, got: %q", result)
	}
}

func TestExtractFirstSections_OnlyOneSectionExists(t *testing.T) {
	md := `# Title

## Only Section
Some content here.
`
	result := extractFirstSections(md, 2, 1000)
	if !strings.Contains(result, "Only Section") {
		t.Error("only section should be included")
	}
	if !strings.Contains(result, "Some content here.") {
		t.Error("section content should be included")
	}
}

func TestExtractFirstSections_TruncatesAtMaxChars(t *testing.T) {
	md := strings.Repeat("a", 2000)
	result := extractFirstSections(md, 2, 100)
	if len(result) > 103 { // 100 chars + "..."
		t.Errorf("expected truncation at ~100 chars, got %d", len(result))
	}
	if !strings.HasSuffix(result, "...") {
		t.Error("truncated result should end with ...")
	}
}

func TestExtractFirstSections_ShortContentNotTruncated(t *testing.T) {
	md := "## Short\nContent"
	result := extractFirstSections(md, 2, 1000)
	if strings.HasSuffix(result, "...") {
		t.Error("short content should not be truncated")
	}
	if result != strings.TrimSpace(md) {
		t.Errorf("expected %q, got %q", strings.TrimSpace(md), result)
	}
}

func TestExtractFirstSections_ExactlyAtLimit(t *testing.T) {
	md := strings.Repeat("b", 1000)
	result := extractFirstSections(md, 2, 1000)
	if strings.HasSuffix(result, "...") {
		t.Error("content exactly at limit should not be truncated")
	}
}
