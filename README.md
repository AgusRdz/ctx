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
ctx init --local --agents v1      Create local config with agents mode preset
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
ctx agents --v1                   Enable v1 subagent capture
ctx agents --v2                   Enable v2 subagent capture (richer)
ctx agents --off                  Disable subagent capture
ctx agents --local --v1           Set mode in local project config
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
  mode: off           # off | v1 | v2
  inject_on_start: true
  max_inject: 5
  staleness_days: 7
```

**Local config** — `{project}/.ctx/config.yml` (optional, project-level overrides)

```yaml
# only include fields you want to override
agents:
  mode: v1
```

Create a local config with:

```sh
ctx init --local
# or with a preset mode:
ctx init --local --agents v1
```

`.ctx/` is automatically added to `.gitignore` — local config is a developer preference, not a team setting.

View effective configuration (with source per field):

```sh
ctx config

effective configuration
───────────────────────────────────────
core.debug              false      [global]
agents.mode             v1         [local]   ← override
agents.inject_on_start  true       [global]
agents.max_inject       5          [global]
agents.staleness_days   7          [global]

global:  ~/.config/ctx/config.yml
local:   /home/agus/projects/myapp/.ctx/config.yml
```

## Subagent capture (ctx agents)

When you use Claude Code with subagents (general-purpose agents or custom agents defined in `.claude/agents/`), ctx can capture their activity and inject a summary into the main agent's context at the start of each session.

Three modes:

| Mode | Behavior |
|------|----------|
| `off` | Default. Only the main agent's context is tracked. |
| `v1` | Captures the final output of each subagent when it stops. |
| `v2` | Captures internal state on `PreCompact` + final output on `SubagentStop`. Richer context. |

Enable globally:

```sh
ctx agents --v1
ctx init   # re-register hooks to include SubagentStop
```

Enable per project (without affecting other projects):

```sh
ctx init --local --agents v1
ctx init
```

When a session starts with agents mode enabled and previous subagent activity exists, ctx injects a block like:

```markdown
## Subagent Activity (2 agents)
- **refactor-agent** (custom): Extracted AuthService to /services/auth.go
- **agent-1741234567** (general): Wrote 14 unit tests, 2 failing in jwt_test.go
```

View captured agents for the current project:

```sh
ctx agents

mode:    v1  [local override]

captured agents (current project):
  refactor-agent     custom     stopped 14m ago
  agent-1741234567   general    stopped 1h ago
```

## How it works

1. **PreCompact hook** — Before Claude compacts, ctx reads the session transcript and git state, then calls `claude -p` to generate a semantic snapshot (with a 30s timeout). Transcript lines are pre-compressed before being sent — repetitive tool calls (e.g. 12 consecutive `Read` calls) are collapsed to a single entry with a repeat count, so the prompt covers more of the session history within the same token budget. If `claude -p` is unavailable, it falls back to a deterministic snapshot derived from git diff/log and CLAUDE.md.

2. **SessionStart hook** — When a session starts (or resumes after compaction), ctx checks for an existing snapshot and prints it to stdout. Claude Code automatically injects this as context. If the snapshot is more than 7 days old, a staleness warning is prepended. If agents mode is enabled, captured subagent activity is appended.

3. **SubagentStop hook** (v1/v2 only) — When a subagent finishes, ctx captures its output and stores it in the project's agents directory.

Snapshots are stored at:
- Linux/macOS: `~/.local/share/ctx/{project-hash}/snapshot.md`
- Windows: `%LOCALAPPDATA%\ctx\{project-hash}\snapshot.md`

Agent snapshots (v1/v2):
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

Agents mode captures subagent activity within the same session context — it does not persist agent history across sessions beyond the normal staleness window.

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
