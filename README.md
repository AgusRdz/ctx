# ctx

Keep your Claude Code session context healthy across compactions.

## The problem

When Claude Code hits its context window limit, it compacts the conversation — summarizing everything into a shorter form. This is necessary, but it's lossy: architectural decisions, the current task goal, what file you were editing, what to do next — all of that can get diluted or lost.

This isn't a memory problem. You don't need Claude to recall past sessions from days ago. The problem is simpler: **your current session loses fidelity every time it compacts**. After two or three compactions, Claude may forget what you were building, repeat work, or contradict decisions you already made together.

## What ctx does

ctx hooks into Claude Code's **PreCompact** and **SessionStart** events to capture and restore a structured snapshot of your working context — automatically, every time.

Before compaction, ctx extracts:
- **Goal** — what you're building right now
- **Decisions** — technical choices already made in this session
- **In Progress** — what files are being modified
- **Next** — what to do when context resumes

When the session resumes after compaction (or when you start a new session in the same project), ctx restores the snapshot — so Claude picks up exactly where you left off.

No configuration. No database. One snapshot per project, always the most recent.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/AgusRdz/ctx/main/install.sh | sh
```

Then register the hooks:

```sh
ctx init
```

That's it. ctx works automatically from this point on.

## Commands

```
ctx init                       Install hooks in Claude Code
ctx init --remove              Remove hooks
ctx init --status              Check hook installation status
ctx show                       Print current snapshot
ctx show --project <path>      Print snapshot for a specific project
ctx clear                      Delete current snapshot
ctx list                       List all projects with snapshots
ctx config                     Show configuration (paths, debug status)
ctx config --debug true|false  Enable or disable verbose hook logging
ctx reset                      Clear snapshots (current directory or all)
ctx doctor                     Check installation health
ctx logs                       Show last 20 hook log entries
ctx logs -n <count>            Show last N entries
ctx logs --all                 Show all entries
ctx uninstall                  Remove ctx completely (hooks, data, binary)
ctx update                     Update to the latest version
ctx version                    Show version
```

## How it works

1. **PreCompact hook** — Before Claude compacts, ctx reads the session transcript and git state, then calls `claude -p` to generate a semantic snapshot (with a 30s timeout). If `claude -p` is unavailable, it falls back to a deterministic snapshot derived from git diff/log and CLAUDE.md.

2. **SessionStart hook** — When a session starts (or resumes after compaction), ctx checks for an existing snapshot and prints it to stdout. Claude Code automatically injects this as context. If the snapshot is more than 7 days old, a staleness warning is prepended.

Snapshots are stored at:
- Linux/macOS: `~/.local/share/ctx/{project-hash}/snapshot.md`
- Windows: `%LOCALAPPDATA%\ctx\{project-hash}\snapshot.md`

## Snapshot format

```markdown
# Session Context

_Captured: 2026-03-09T14:32Z_

## Goal
Building the authentication middleware

## Decisions
- Using JWT with RS256 for token signing
- Middleware applied at router level, not per-handler

## In Progress
 auth/middleware.go | 45 +++++++++
 auth/jwt.go       | 32 ++++++
 2 files changed, 77 insertions(+)

## Next
Add token refresh endpoint and write integration tests
```

## Debugging

```sh
ctx doctor                     Check hooks, binary path, claude availability
ctx logs                       Show last 20 hook log entries
ctx logs -n 50                 Show last 50 entries
ctx logs --all                 Show all entries
ctx config --debug true        Enable verbose logging (context sizes, timings)
```

Log file location:
- Linux/macOS: `~/.local/share/ctx/debug.log`
- Windows: `%LOCALAPPDATA%\ctx\debug.log`

## What ctx is NOT

ctx is not a memory tool. It doesn't accumulate knowledge across sessions, index conversations, or build a searchable history. It solves one specific problem: **keeping the current session coherent when context gets compacted**. One project, one snapshot, always overwritten.

## Development

Requires Docker (no local Go installation needed):

```sh
make build        # Build binary
make test         # Run tests
make install      # Build + install locally
make cross        # Cross-compile all platforms
```

## License

MIT
