package snapshot

import (
	"strings"
	"testing"
)

func TestFormatSnapshot_AllSections(t *testing.T) {
	data := SnapshotData{
		Goal:       "Implement user authentication",
		Decisions:  []string{"Use JWT tokens", "Store refresh tokens in DB"},
		InProgress: "Writing middleware for token validation",
		Next:       "Add unit tests for auth middleware",
	}

	result := FormatSnapshot(data)

	// Verify all four section headers are present
	sections := []string{"# Session Context", "## Goal", "## Decisions", "## In Progress", "## Next"}
	for _, s := range sections {
		if !strings.Contains(result, s) {
			t.Errorf("missing section %q in output:\n%s", s, result)
		}
	}

	// Verify content appears under the correct sections
	if !strings.Contains(result, "Implement user authentication") {
		t.Error("goal content missing")
	}
	if !strings.Contains(result, "- Use JWT tokens\n") {
		t.Error("first decision missing or not formatted as list item")
	}
	if !strings.Contains(result, "- Store refresh tokens in DB\n") {
		t.Error("second decision missing or not formatted as list item")
	}
	if !strings.Contains(result, "Writing middleware for token validation") {
		t.Error("in_progress content missing")
	}
	if !strings.Contains(result, "Add unit tests for auth middleware") {
		t.Error("next content missing")
	}
}

func TestFormatSnapshot_SectionOrder(t *testing.T) {
	data := SnapshotData{
		Goal:       "goal-text",
		Decisions:  []string{"decision-text"},
		InProgress: "progress-text",
		Next:       "next-text",
	}

	result := FormatSnapshot(data)

	goalIdx := strings.Index(result, "## Goal")
	decisionsIdx := strings.Index(result, "## Decisions")
	progressIdx := strings.Index(result, "## In Progress")
	nextIdx := strings.Index(result, "## Next")

	if goalIdx >= decisionsIdx || decisionsIdx >= progressIdx || progressIdx >= nextIdx {
		t.Errorf("sections are not in expected order (Goal < Decisions < In Progress < Next):\n%s", result)
	}
}

func TestFormatSnapshot_EmptyDecisions(t *testing.T) {
	data := SnapshotData{
		Goal:       "Some goal",
		Decisions:  []string{},
		InProgress: "Some work",
		Next:       "Next step",
	}

	result := FormatSnapshot(data)

	// Decisions section should still exist but have no list items
	if !strings.Contains(result, "## Decisions\n") {
		t.Error("decisions section header missing")
	}
	if strings.Contains(result, "- ") {
		t.Error("unexpected list item found with empty decisions slice")
	}

	// Other sections should still render correctly
	if !strings.Contains(result, "Some goal") {
		t.Error("goal content missing with empty decisions")
	}
	if !strings.Contains(result, "Some work") {
		t.Error("in_progress content missing with empty decisions")
	}
	if !strings.Contains(result, "Next step") {
		t.Error("next content missing with empty decisions")
	}
}

func TestFormatSnapshot_NilDecisions(t *testing.T) {
	data := SnapshotData{
		Goal:       "Goal",
		Decisions:  nil,
		InProgress: "WIP",
		Next:       "Continue",
	}

	result := FormatSnapshot(data)

	if !strings.Contains(result, "## Decisions\n") {
		t.Error("decisions section header missing with nil decisions")
	}
	if strings.Contains(result, "- ") {
		t.Error("unexpected list item found with nil decisions")
	}
}

func TestFormatSnapshot_EndsWithNewline(t *testing.T) {
	data := SnapshotData{
		Goal:       "Goal",
		Decisions:  []string{"d1"},
		InProgress: "WIP",
		Next:       "Next",
	}

	result := FormatSnapshot(data)

	if !strings.HasSuffix(result, "\n") {
		t.Error("formatted snapshot should end with a newline")
	}
}

func TestGenerateFallback_ValidMarkdown(t *testing.T) {
	ctx := Context{
		DiffStat:   "file1.go | 10 +\nfile2.go | 5 -",
		ProjectDir: "/tmp/test",
		ProjectMD:  "Test project",
	}

	result := GenerateFallback(ctx)

	// Should contain a goal extracted from ProjectMD
	if !strings.Contains(result, "Test project") {
		t.Error("fallback goal should be extracted from ProjectMD")
	}

	// In Progress should contain the DiffStat
	if !strings.Contains(result, "file1.go | 10 +") {
		t.Error("DiffStat not included in fallback in_progress section")
	}

	// Next should contain the default fallback instruction
	if !strings.Contains(result, "Review modified files and continue") {
		t.Error("fallback next step missing")
	}

	// Should still have all four sections
	for _, s := range []string{"## Goal", "## Decisions", "## In Progress", "## Next"} {
		if !strings.Contains(result, s) {
			t.Errorf("missing section %q in fallback output", s)
		}
	}
}

func TestInferGoal_NoCommitsNoMD(t *testing.T) {
	ctx := Context{
		DiffStat:   "",
		RecentLog:  "",
		ProjectMD:  "",
		ProjectDir: "/tmp/myproject",
	}

	result := GenerateFallback(ctx)

	if !strings.Contains(result, "myproject") {
		t.Errorf("expected project dir name in goal, got:\n%s", result)
	}
	if strings.Contains(result, "Unable to determine") {
		t.Error("should not contain 'Unable to determine'")
	}
}

func TestInferGoal_NoCommitsWithMD(t *testing.T) {
	ctx := Context{
		DiffStat:   "",
		RecentLog:  "",
		ProjectMD:  "# myproject\n\nA CLI tool for doing things.\n",
		ProjectDir: "/tmp/myproject",
	}

	result := GenerateFallback(ctx)

	if !strings.Contains(result, "A CLI tool for doing things.") {
		t.Errorf("expected CLAUDE.md description as goal, got:\n%s", result)
	}
}

func TestInferGoal_CommitsNoMD(t *testing.T) {
	ctx := Context{
		DiffStat:   "",
		RecentLog:  "abc1234 feat: add login command\ndef5678 fix: handle empty input",
		ProjectMD:  "",
		ProjectDir: "/tmp/myproject",
	}

	result := GenerateFallback(ctx)

	if !strings.Contains(result, "feat: add login command") {
		t.Errorf("expected latest commit in goal, got:\n%s", result)
	}
}

func TestStripCodeFences(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain json unchanged",
			input: `{"goal":"test"}`,
			want:  `{"goal":"test"}`,
		},
		{
			name:  "json fenced with ```json",
			input: "```json\n{\"goal\":\"test\"}\n```",
			want:  `{"goal":"test"}`,
		},
		{
			name:  "json fenced with plain ```",
			input: "```\n{\"goal\":\"test\"}\n```",
			want:  `{"goal":"test"}`,
		},
		{
			name:  "whitespace trimmed",
			input: "```json\n  {\"goal\":\"test\"}  \n```",
			want:  `{"goal":"test"}`,
		},
		{
			name:  "sanity guard: non-json returned unchanged",
			input: "Sorry, I cannot answer that.",
			want:  "Sorry, I cannot answer that.",
		},
		{
			name:  "sanity guard: too short returned unchanged",
			input: "```json\nhello\n```",
			want:  "```json\nhello\n```",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := stripCodeFences(tc.input)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGenerateFallback_EmptyDiffStat(t *testing.T) {
	ctx := Context{
		DiffStat:   "",
		ProjectDir: "/tmp/test",
	}

	result := GenerateFallback(ctx)

	// Should still produce valid markdown even with empty diff
	if !strings.Contains(result, "## In Progress\n") {
		t.Error("in_progress section missing with empty DiffStat")
	}
	if !strings.Contains(result, "## Next\n") {
		t.Error("next section missing with empty DiffStat")
	}
}
