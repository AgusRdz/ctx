# ctx

<p align="center">
  <img src="logo.png" alt="ctx logo" width="200" />
</p>

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

## Install

**Homebrew (macOS/Linux):**

```sh
brew install AgusRdz/tap/ctx
```

**curl installer:**

```sh
curl -fsSL https://raw.githubusercontent.com/AgusRdz/ctx/main/install.sh | sh
```

Then register the hooks:

```sh
ctx init
```

That's it. ctx works automatically from this point on.

All release binaries include [build provenance attestations](https://github.com/AgusRdz/ctx/attestations) verifiable with the GitHub CLI:

```sh
gh attestation verify <binary> --repo AgusRdz/ctx
```

## Commands

```
ctx init                          Install hooks in Claude Code
ctx init --remove                 Remove hooks
ctx init --status                 Check hook installation status
ctx init --local                  Create local project config (.ctx/config.yml)
ctx init --local --agents on      Create local config with agents capture enabled
ctx show                          Print current snapshot
ctx show --project <path>         Print snapshot for a specific project
ctx clear                         Delete current snapshot
ctx clear --agents-only           Clear only agent snapshots
ctx list                          List all projects with snapshots
ctx config                        Show effective configuration with sources
ctx config --global               Show only global config
ctx config --local                Show only local config
ctx config --debug true|false     Enable or disable verbose hook logging
ctx agents                        Show agents mode and captured agents
ctx agents show <name>            Print full snapshot for a captured agent
ctx agents show --all             Print all agent snapshots
  [--project <path>] [--since Nd] Filter by project or age (e.g. --since 7d)
ctx agents archive                List archived agent sessions
ctx agents rm <name>              Remove a specific agent snapshot
ctx agents rm --before Nd         Remove snapshots older than N days/weeks
ctx agents rm --session <id>      Remove an archived session
ctx agents rm --all               Remove all agent snapshots
ctx agents summarize              AI summary of agent work via claude -p
  [--all] [--since Nd] [--project <path>]
ctx agents --on                   Enable agent capture
ctx agents --off                  Disable agent capture
ctx agents --local --on           Set mode in local project config
ctx reset                         Clear snapshots (current directory or all)
ctx doctor                        Check installation health
ctx logs                          Show last 20 hook log entries
ctx logs -n <count>               Show last N entries
ctx logs --all                    Show all entries
ctx changelog                     Show changes in the current version
ctx changelog --full              Show full changelog history
ctx uninstall                     Remove ctx completely (hooks, data, binary)
ctx update                        Update to the latest version
ctx version                       Show version
```

## Configuration

ctx uses two config layers that merge field-by-field. Local values win over global.

**Global config** — `~/.config/ctx/config.yml` (Windows: `%APPDATA%\ctx\config.yml`)

```yaml
core:
  debug: false

agents:
  mode: off           # off | on
```

**Local config** — `{project}/.ctx/config.yml` (optional, project-level overrides)

```yaml
# only include fields you want to override
agents:
  mode: on
```

Create a local config with:

```sh
ctx init --local
# or with a preset mode:
ctx init --local --agents on
```

`.ctx/` is automatically added to `.gitignore` — local config is a developer preference, not a team setting.

View effective configuration (with source per field):

```sh
ctx config

effective configuration
───────────────────────────────────────
core.debug              false      [global]
agents.mode             on         [local]   ← override

global:  ~/.config/ctx/config.yml
local:   /home/agus/projects/myapp/.ctx/config.yml
```

## Subagent capture (ctx agents)

When you use Claude Code with subagents (general-purpose agents or custom agents defined in `.claude/agents/`), ctx can capture their activity as human-readable snapshots. These are **not** injected into Claude's context — they exist purely for you to read, review, and share.

Agent snapshots capture what each subagent did: what task it was given, what actions it took, and what it produced. They're useful for writing tickets, explaining work to teammates, or just reviewing what happened in a long session.

Enable agent capture:

```sh
ctx agents --on
ctx init   # re-register hooks to include SubagentStop
```

Enable per project only:

```sh
ctx init --local --agents on
ctx init
```

View captured agents for the current project:

```sh
ctx agents

mode:    on  [global]

captured agents (current project):
  feature-RES-219-20260313-150405   custom    stopped 14m ago
  feature-RES-219-20260313-143201   general   stopped 1h ago
```

Read what an agent did:

```sh
# one agent
ctx agents show feature-RES-219-20260313-150405

# all agents at once (works from any directory)
ctx agents show --all --project /path/to/project

# filter to last 2 days
ctx agents show --all --since 2d
```

Get an AI-generated digest across all agents (useful for writing tickets):

```sh
ctx agents summarize
ctx agents summarize --all --since 1w --project /path/to/project
```

Manage old snapshots:

```sh
ctx agents rm feature-RES-219-20260313-150405   # specific agent
ctx agents rm --before 7d                        # older than 7 days
ctx agents rm --session 20260313-150405          # entire archived session
ctx agents rm --all                              # everything
```

Agent snapshots are grouped by session. When a compaction happens, current agents are archived under a timestamp slot — so you can still access them later with `ctx agents archive`.

## How it works

1. **PreCompact hook** — Before Claude compacts, ctx reads the session transcript and git state, then calls `claude -p` to generate a semantic snapshot (with a 30s timeout). Transcript lines are pre-compressed before being sent — repetitive tool calls (e.g. 12 consecutive `Read` calls) are collapsed to a single entry with a repeat count, so the prompt covers more of the session history within the same token budget. If `claude -p` is unavailable, it falls back to a deterministic snapshot derived from git diff/log and CLAUDE.md.

2. **SessionStart hook** — When a session starts (or resumes after compaction), ctx checks for an existing snapshot and prints it to stdout. Claude Code automatically injects this as context. If the snapshot is more than 7 days old, a staleness warning is prepended.

3. **SubagentStop hook** (agents on only) — When a subagent finishes, ctx captures its output and stores it in the project's agents directory.

Snapshots are stored at:
- Linux/macOS: `~/.local/share/ctx/{project-hash}/snapshot.md`
- Windows: `%LOCALAPPDATA%\ctx\{project-hash}\snapshot.md`

Agent snapshots:
- Linux/macOS: `~/.local/share/ctx/{project-hash}/agents/{name}.md`
- Windows: `%LOCALAPPDATA%\ctx\{project-hash}\agents\{name}.md`

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
ctx doctor                     Check hooks, binary path, claude availability, config
ctx logs                       Show last 20 hook log entries
ctx logs -n 50                 Show last 50 entries
ctx logs --all                 Show all entries
ctx config --debug true        Enable verbose logging (context sizes, timings)
```

Log file location:
- Linux/macOS: `~/.local/share/ctx/debug.log`
- Windows: `%LOCALAPPDATA%\ctx\debug.log`

## Updates

ctx updates itself automatically in the background. Once every 24 hours it checks for a new release, downloads it silently, and applies it on the next invocation — you'll see a one-line notification when it does.

To update immediately:

```sh
ctx update
```

To see what changed:

```sh
ctx changelog
```

## What ctx is NOT

ctx is not a memory tool. It doesn't accumulate knowledge across sessions, index conversations, or build a searchable history. It solves one specific problem: **keeping the current session coherent when context gets compacted**. One project, one snapshot, always overwritten.

Agents mode captures subagent activity as human-readable snapshots for review and handoff. They are never injected into Claude's context — the main snapshot is the only thing Claude sees.

## Development

Requires Docker (no local Go installation needed):

```sh
make build        # Build binary
make test         # Run tests
make install      # Build + install locally
make cross        # Cross-compile all platforms
make changelog    # Regenerate CHANGELOG.md (requires git-cliff)
make release      # Auto-detect bump and release
make release-patch / release-minor / release-major
```

## License

MIT
