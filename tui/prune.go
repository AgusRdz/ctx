// Package tui — interactive prune picker for ctx snapshots.
//
// RunPrune launches a Bubble Tea program that lets the user multi-select
// snapshots to delete. Snapshots older than `threshold` are pre-selected.
package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/AgusRdz/ctx/snapshot"
)

// pruneItem wraps a snapshot with picker state.
type pruneItem struct {
	info     snapshot.SnapshotInfo
	selected bool
	stale    bool
}

// pruneModel is the Bubble Tea model for the prune picker.
type pruneModel struct {
	items     []pruneItem
	cursor    int // index into items (not visible — see visible())
	staleOnly bool
	threshold time.Duration
	confirm   bool
	width     int
	height    int
	deleted   []snapshot.SnapshotInfo // populated on commit
	quit      bool
}

// PruneResult is returned by RunPrune.
type PruneResult struct {
	Deleted []snapshot.SnapshotInfo
}

// RunPrune launches the picker. Returns the snapshots the user chose to
// delete (already removed from disk), or an empty slice if the user quit.
func RunPrune(infos []snapshot.SnapshotInfo, threshold time.Duration) (PruneResult, error) {
	items := make([]pruneItem, len(infos))
	for i, info := range infos {
		stale := false
		if threshold > 0 && !info.CapturedAt.IsZero() && time.Since(info.CapturedAt) >= threshold {
			stale = true
		}
		items[i] = pruneItem{info: info, selected: stale, stale: stale}
	}

	m := pruneModel{items: items, threshold: threshold}
	p := tea.NewProgram(m)
	final, err := p.Run()
	if err != nil {
		return PruneResult{}, fmt.Errorf("ctx: tui: %w", err)
	}
	out := final.(pruneModel)
	return PruneResult{Deleted: out.deleted}, nil
}

func (m pruneModel) Init() tea.Cmd { return nil }

// visible returns the indices of items that pass the current filter,
// in display order.
func (m pruneModel) visible() []int {
	out := make([]int, 0, len(m.items))
	for i, it := range m.items {
		if m.staleOnly && !it.stale {
			continue
		}
		out = append(out, i)
	}
	return out
}

func (m pruneModel) selectedCount() int {
	n := 0
	for _, it := range m.items {
		if it.selected {
			n++
		}
	}
	return n
}

func (m pruneModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.confirm {
			return m.updateConfirm(msg)
		}
		return m.updateList(msg)
	}
	return m, nil
}

func (m pruneModel) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	vis := m.visible()
	switch msg.String() {
	case "ctrl+c", "q", "esc":
		m.quit = true
		return m, tea.Quit
	case "up", "k":
		if len(vis) == 0 {
			return m, nil
		}
		pos := m.cursorPos(vis)
		if pos > 0 {
			m.cursor = vis[pos-1]
		}
	case "down", "j":
		if len(vis) == 0 {
			return m, nil
		}
		pos := m.cursorPos(vis)
		if pos < len(vis)-1 {
			m.cursor = vis[pos+1]
		}
	case "home", "g":
		if len(vis) > 0 {
			m.cursor = vis[0]
		}
	case "end", "G":
		if len(vis) > 0 {
			m.cursor = vis[len(vis)-1]
		}
	case " ":
		if len(vis) > 0 {
			m.items[m.cursor].selected = !m.items[m.cursor].selected
		}
	case "a":
		// Toggle: if any visible is unselected, select all visible; else clear all visible.
		anyUnselected := false
		for _, idx := range vis {
			if !m.items[idx].selected {
				anyUnselected = true
				break
			}
		}
		for _, idx := range vis {
			m.items[idx].selected = anyUnselected
		}
	case "s":
		m.staleOnly = !m.staleOnly
		// Reset cursor if it's now hidden
		newVis := m.visible()
		if len(newVis) == 0 {
			return m, nil
		}
		stillVisible := false
		for _, idx := range newVis {
			if idx == m.cursor {
				stillVisible = true
				break
			}
		}
		if !stillVisible {
			m.cursor = newVis[0]
		}
	case "enter":
		if m.selectedCount() == 0 {
			return m, nil
		}
		m.confirm = true
	}
	return m, nil
}

func (m pruneModel) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		// Commit deletions
		for _, it := range m.items {
			if !it.selected {
				continue
			}
			if err := snapshot.Clear(it.info.ProjectDir, it.info.Branch); err != nil {
				continue // best effort; surfaced count below reflects intent
			}
			m.deleted = append(m.deleted, it.info)
		}
		return m, tea.Quit
	case "n", "N", "esc", "ctrl+c", "q":
		m.confirm = false
		return m, nil
	}
	return m, nil
}

// cursorPos returns the position of m.cursor within the visible slice,
// or 0 if not present.
func (m pruneModel) cursorPos(vis []int) int {
	for pos, idx := range vis {
		if idx == m.cursor {
			return pos
		}
	}
	return 0
}

// ─── styles ─────────────────────────────────────────────────────────────

var (
	borderColor   = lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#4B5563"}
	titleColor    = lipgloss.AdaptiveColor{Light: "#111827", Dark: "#F9FAFB"}
	dimColor      = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}
	cursorBgColor = lipgloss.AdaptiveColor{Light: "#E5E7EB", Dark: "#1F2937"}
	staleColor    = lipgloss.AdaptiveColor{Light: "#B45309", Dark: "#F59E0B"}
	checkedColor  = lipgloss.AdaptiveColor{Light: "#047857", Dark: "#34D399"}
	dangerColor   = lipgloss.AdaptiveColor{Light: "#B91C1C", Dark: "#F87171"}

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(0, 1)

	titleStyle  = lipgloss.NewStyle().Foreground(titleColor).Bold(true)
	dimStyle    = lipgloss.NewStyle().Foreground(dimColor)
	cursorStyle = lipgloss.NewStyle().Background(cursorBgColor).Bold(true)
	staleTag    = lipgloss.NewStyle().Foreground(staleColor).Bold(true)
	checkStyle  = lipgloss.NewStyle().Foreground(checkedColor).Bold(true)
	dangerStyle = lipgloss.NewStyle().Foreground(dangerColor).Bold(true)
)

func (m pruneModel) View() string {
	if m.confirm {
		return m.viewConfirm()
	}
	return m.viewList()
}

func (m pruneModel) viewList() string {
	vis := m.visible()

	// Header
	header := titleStyle.Render("ctx prune")
	thresholdLabel := ""
	if m.threshold > 0 {
		days := int(m.threshold.Hours() / 24)
		thresholdLabel = dimStyle.Render(fmt.Sprintf("  %dd stale threshold", days))
	}
	filterLabel := ""
	if m.staleOnly {
		filterLabel = staleTag.Render("  stale only")
	}

	var rows []string
	rows = append(rows, header+thresholdLabel+filterLabel)
	rows = append(rows, dimStyle.Render(strings.Repeat("─", 60)))

	if len(vis) == 0 {
		rows = append(rows, dimStyle.Render("  (no snapshots match the current filter)"))
	}

	// Compute column widths from visible items only
	maxRepoW, maxBranchW, maxAgeW := 0, 0, 0
	for _, idx := range vis {
		it := m.items[idx]
		if w := lipgloss.Width(repoName(it.info.ProjectDir)); w > maxRepoW {
			maxRepoW = w
		}
		if w := lipgloss.Width(branchLabel(it.info.Branch)); w > maxBranchW {
			maxBranchW = w
		}
		if w := lipgloss.Width(ageLabel(it.info.CapturedAt)); w > maxAgeW {
			maxAgeW = w
		}
	}

	for _, idx := range vis {
		it := m.items[idx]
		check := "[ ]"
		if it.selected {
			check = checkStyle.Render("[x]")
		}
		repo := padRight(repoName(it.info.ProjectDir), maxRepoW)
		branch := padRight(branchLabel(it.info.Branch), maxBranchW)
		age := padRight(ageLabel(it.info.CapturedAt), maxAgeW)
		goal := truncate(it.info.Goal, 50)

		line := fmt.Sprintf(" %s  %s  %s  %s  %s", check, repo, branch, dimStyle.Render(age), goal)
		if it.stale {
			line += "  " + staleTag.Render("stale")
		}
		if idx == m.cursor {
			line = cursorStyle.Render(line)
		}
		rows = append(rows, line)
	}

	body := strings.Join(rows, "\n")
	box := boxStyle.Render(body)

	footer := dimStyle.Render(fmt.Sprintf("  %d of %d selected", m.selectedCount(), len(m.items)))
	keys := dimStyle.Render("  space toggle   a all   s stale-only   ↑↓ move   enter delete   q quit")

	return box + "\n" + footer + "\n" + keys + "\n"
}

func (m pruneModel) viewConfirm() string {
	n := m.selectedCount()
	msg := fmt.Sprintf("Delete %d snapshot", n)
	if n != 1 {
		msg += "s"
	}
	msg += "?"

	body := dangerStyle.Render(msg) + "\n\n" +
		dimStyle.Render("  y  yes, delete   n / esc  back to list")

	return boxStyle.Render(body) + "\n"
}

// ─── helpers ────────────────────────────────────────────────────────────

func repoName(projectDir string) string {
	name := filepath.Base(strings.TrimRight(projectDir, `/\`))
	if name == "" || name == "." {
		return projectDir
	}
	return name
}

func branchLabel(branch string) string {
	if branch == "" || branch == "_" {
		return "(no branch)"
	}
	return "[" + branch + "]"
}

func ageLabel(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "<1m ago"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func padRight(s string, w int) string {
	pad := w - lipgloss.Width(s)
	if pad <= 0 {
		return s
	}
	return s + strings.Repeat(" ", pad)
}

func truncate(s string, w int) string {
	if lipgloss.Width(s) <= w {
		return s
	}
	if w <= 1 {
		return "…"
	}
	return s[:w-1] + "…"
}
