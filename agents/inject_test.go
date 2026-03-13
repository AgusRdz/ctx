package agents

import (
	"strings"
	"testing"
	"time"
)

func makeSnapshot(name, agentType, output string, minutesAgo int) AgentSnapshot {
	return AgentSnapshot{
		Name:        name,
		Type:        agentType,
		StoppedAt:   time.Now().Add(-time.Duration(minutesAgo) * time.Minute),
		FinalOutput: output,
	}
}

func TestBuildInjectionBlock_Empty(t *testing.T) {
	block := BuildInjectionBlock(nil, 7, 5)
	if block != "" {
		t.Errorf("expected empty block for nil snapshots, got %q", block)
	}
}

func TestBuildInjectionBlock_Basic(t *testing.T) {
	snapshots := []AgentSnapshot{
		makeSnapshot("refactor-agent", "custom", "Extracted AuthService", 10),
		makeSnapshot("agent-1234567890", "general", "Wrote 14 unit tests", 60),
	}
	block := BuildInjectionBlock(snapshots, 7, 5)

	if !strings.Contains(block, "## Subagent Activity") {
		t.Error("expected Subagent Activity header")
	}
	if !strings.Contains(block, "refactor-agent") {
		t.Error("expected refactor-agent in block")
	}
	if !strings.Contains(block, "Extracted AuthService") {
		t.Error("expected final output in block")
	}
}

func TestBuildInjectionBlock_MaxInject(t *testing.T) {
	snapshots := []AgentSnapshot{
		makeSnapshot("agent-1", "custom", "done 1", 1),
		makeSnapshot("agent-2", "custom", "done 2", 2),
		makeSnapshot("agent-3", "custom", "done 3", 3),
		makeSnapshot("agent-4", "custom", "done 4", 4),
	}
	block := BuildInjectionBlock(snapshots, 7, 2)

	if !strings.Contains(block, "+2 more") {
		t.Errorf("expected '+2 more' overflow note, got:\n%s", block)
	}
	if !strings.Contains(block, "agent-1") {
		t.Error("expected agent-1 (most recent) in block")
	}
}

func TestBuildInjectionBlock_StalenessFilter(t *testing.T) {
	snapshots := []AgentSnapshot{
		makeSnapshot("fresh-agent", "custom", "recent work", 60),        // 1 hour ago
		makeSnapshot("stale-agent", "custom", "old work", 24*60*10+1), // 10+ days ago
	}
	// stalenessDays=7 should filter out the stale agent
	block := BuildInjectionBlock(snapshots, 7, 5)

	if !strings.Contains(block, "fresh-agent") {
		t.Error("expected fresh-agent in block")
	}
	if strings.Contains(block, "stale-agent") {
		t.Error("stale-agent should be filtered out")
	}
}

func TestBuildInjectionBlock_AllStale(t *testing.T) {
	snapshots := []AgentSnapshot{
		makeSnapshot("stale-agent", "custom", "old work", 24*60*10), // 10 days ago
	}
	block := BuildInjectionBlock(snapshots, 7, 5)
	if block != "" {
		t.Errorf("expected empty block when all agents are stale, got %q", block)
	}
}
