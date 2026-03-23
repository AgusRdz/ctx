package projectstate

import (
	"fmt"
	"strings"
)

// Format renders a ProjectState as a ## Project State markdown section.
// maxDirtyFiles and maxErrors cap the number of items shown.
func Format(ps ProjectState, maxDirtyFiles, maxErrors int) string {
	var b strings.Builder
	b.WriteString("## Project State (at compaction)\n")
	formatGit(&b, ps.Git, maxDirtyFiles)
	if ps.TypeCheck.Tool != "" && ps.TypeCheck.Tool != "none" {
		formatTypeCheck(&b, ps.TypeCheck, maxErrors)
	}
	if ps.Tests.Tool != "" && ps.Tests.Tool != "none" {
		formatTests(&b, ps.Tests)
	}
	return b.String()
}

func formatGit(b *strings.Builder, g GitState, maxDirtyFiles int) {
	if g.Branch == "" {
		b.WriteString("Git:  (not a git repository)\n")
		return
	}
	line := "Git:  " + g.Branch
	if g.AheadBehind != "" {
		line += " | " + g.AheadBehind
	}
	if g.LastCommit != "" {
		line += " | last: " + g.LastCommit
	}
	b.WriteString(line + "\n")

	if len(g.DirtyFiles) == 0 {
		return
	}
	shown := g.DirtyFiles
	extra := 0
	if maxDirtyFiles > 0 && len(shown) > maxDirtyFiles {
		extra = len(shown) - maxDirtyFiles
		shown = shown[:maxDirtyFiles]
	}
	dirty := "Dirty: " + strings.Join(shown, ", ")
	if extra > 0 {
		dirty += fmt.Sprintf(" (+%d more)", extra)
	}
	b.WriteString(dirty + "\n")
}

func formatTypeCheck(b *strings.Builder, tc TypeCheckState, maxErrors int) {
	b.WriteString("\n")
	if tc.TimedOut {
		b.WriteString(fmt.Sprintf("TypeCheck: %s — timed out\n", tc.Tool))
		return
	}
	if tc.ErrorCount == 0 {
		b.WriteString(fmt.Sprintf("TypeCheck: %s — ok\n", tc.Tool))
		return
	}
	b.WriteString(fmt.Sprintf("TypeCheck: %s — %d error(s)\n", tc.Tool, tc.ErrorCount))
	shown := tc.Errors
	if maxErrors > 0 && len(shown) > maxErrors {
		shown = shown[:maxErrors]
	}
	for _, e := range shown {
		b.WriteString("  " + e + "\n")
	}
}

func formatTests(b *strings.Builder, ts TestState) {
	b.WriteString("\n")
	if ts.TimedOut {
		b.WriteString(fmt.Sprintf("Tests: %s — timed out\n", ts.Tool))
		return
	}
	b.WriteString(fmt.Sprintf("Tests: %s — %d pass | %d fail | %d skip\n",
		ts.Tool, ts.Pass, ts.Fail, ts.Skip))
	for _, name := range ts.FailedNames {
		b.WriteString("  x " + name + "\n")
	}
}
