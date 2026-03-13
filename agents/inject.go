package agents

import (
	"fmt"
	"strings"
	"time"
)

// BuildInjectionBlock creates the Subagent Activity markdown block for SessionStart injection.
// staleness_days: filter out agents older than this many days (0 = no filter)
// maxInject: cap on agents shown (0 = no cap)
func BuildInjectionBlock(snapshots []AgentSnapshot, stalenessDays int, maxInject int) string {
	if len(snapshots) == 0 {
		return ""
	}

	// Filter stale snapshots
	var fresh []AgentSnapshot
	if stalenessDays > 0 {
		cutoff := time.Now().Add(-time.Duration(stalenessDays) * 24 * time.Hour)
		for _, s := range snapshots {
			if s.StoppedAt.After(cutoff) {
				fresh = append(fresh, s)
			}
		}
	} else {
		fresh = snapshots
	}

	if len(fresh) == 0 {
		return ""
	}

	extra := 0
	if maxInject > 0 && len(fresh) > maxInject {
		extra = len(fresh) - maxInject
		fresh = fresh[:maxInject]
	}

	total := len(fresh) + extra
	plural := "s"
	if total == 1 {
		plural = ""
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("## Subagent Activity (%d agent%s)\n", total, plural))

	for _, s := range fresh {
		// Use first line of final output, truncated for brevity
		output := strings.SplitN(s.FinalOutput, "\n", 2)[0]
		if len(output) > 120 {
			output = output[:117] + "..."
		}
		b.WriteString(fmt.Sprintf("- **%s** (%s): %s\n", s.Name, s.Type, output))
	}

	if extra > 0 {
		b.WriteString(fmt.Sprintf("(+%d more — run `ctx agents` to see all)\n", extra))
	}

	return b.String()
}
