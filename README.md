# ctx

<p align="center">
  <img src="logo.png" alt="ctx logo" width="200" />
</p>

**Preserve Claude Code session context across compactions — automatically.**

When Claude Code hits its context window limit, it compacts the conversation. This is necessary, but lossy: the current goal, technical decisions, the file you were editing, what to do next — all of it can get diluted or lost. After two or three compactions, Claude may forget what you were building, repeat work, or contradict decisions you already made together.

ctx hooks into Claude Code's **PreCompact** and **SessionStart** events to capture and restore a structured snapshot of your working context, every time.

---

## How It Works

```
┌──────────────────────────────────────────────────────────────┐
│ Claude Code session                                          │
│                                                              │
│  /compact triggered                                          │
│       │                                                      │
│       ▼                                                      │
│  PreCompact hook → ctx reads transcript + git state          │
│                  → calls claude -p to extract semantics      │
│                  → writes snapshot.md (goal/decisions/next)  │
│                                                              │
│  Session resumes                                             │
│       │                                                      │
│       ▼                                                      │
│  SessionStart hook → ctx finds snapshot for this project     │
│                    → prints it to stdout                     │
│                    → Claude Code injects it as context       │
└──────────────────────────────────────────────────────────────┘
```

The snapshot is a structured markdown document with four fields:

| Field | What it captures |
|-------|-----------------|
| **Goal** | What you're building right now |
| **Decisions** | Technical choices already made this session |
| **In Progress** | Files being modified |
| **Next** | What to do when context resumes |

---

## Install

**macOS / Linux:**

```sh
curl -fsSL https://raw.githubusercontent.com/AgusRdz/ctx/main/install.sh | sh
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/AgusRdz/ctx/main/install.ps1 | iex
```

**Homebrew (macOS / Linux):**

```sh
brew install AgusRdz/tap/ctx
```

**With Go:**

```sh
go install github.com/AgusRdz/ctx@latest
```

Then register the hooks:

```sh
ctx init
```

That's it. ctx works automatically from this point on.

Update to latest:

```sh
ctx update
```

## Verification

All release binaries include [build provenance attestations](https://github.com/AgusRdz/ctx/attestations) verifiable with the GitHub CLI:

```sh
gh attestation verify <binary> --repo AgusRdz/ctx
```

---

## Quick Start

```sh
ctx init                  # register PreCompact + SessionStart hooks
ctx show                  # print the current snapshot
ctx list                  # list all projects with snapshots
ctx agents --on           # enable subagent capture
ctx agents                # show captured agents for this project
ctx agents workspace add ~/dev   # scan ~/dev for projects automatically
```

---

## Commands

### Setup

```
ctx init                                 Install PreCompact and SessionStart hooks
ctx init --remove                        Remove ctx hooks
ctx init --status                        Check hook installation status
ctx init --local                         Create local project config (.ctx/config.yml)
ctx init --local --agents on             Create local config with agents capture enabled
```

### Session

```
ctx show                                 Print current snapshot
ctx show --project <path>                Print snapshot for a specific project
ctx clear                                Delete current snapshot
ctx clear --agents-only                  Clear only agent snapshots
ctx list                                 List all projects with snapshots
```

### Agents

```
ctx agents                               Show mode and captured agents (current project)
ctx agents --global                      Show agents across all projects
ctx agents --on                          Enable agent capture
ctx agents --off                         Disable agent capture
ctx agents --local --on                  Set mode in local project config
ctx agents show <name>                   Print full snapshot for one agent
ctx agents show --all                    Print all agent snapshots
  [--project <path>] [--since Nd|Nw]     Filter by project or age (e.g. --since 7d)
ctx agents archive [--project <path>]    List archived agent sessions
ctx agents summarize                     AI summary of agent work via claude -p
  [--all] [--since Nd|Nw]                Include archived / filter by age
  [--project <path>]
ctx agents rm <name>                     Remove a specific agent snapshot
ctx agents rm --before Nd|Nw             Remove snapshots older than N days/weeks
ctx agents rm --session <id>             Remove an entire archived session
ctx agents rm --all                      Remove all agent snapshots
ctx agents --help                        Full agents command reference
```

### Workspace Scanning

```
ctx agents workspace list                Show configured workspaces, exclusions, markers
ctx agents workspace add <path>          Add a workspace directory to scan
ctx agents workspace rm <path>           Remove a workspace directory
ctx agents workspace exclude <path>      Always skip this path during scans
ctx agents workspace unexclude <path>    Remove a path from the exclusion list
ctx agents workspace marker add <glob>   Add a custom root marker (e.g. *.csproj)
ctx agents workspace marker rm <glob>    Remove a custom root marker
ctx agents workspace boundary add <dir>  Add a custom boundary dir (e.g. .terraform)
ctx agents workspace boundary rm <dir>   Remove a custom boundary dir
```

→ Full reference: [docs/workspace-scanning.md](docs/workspace-scanning.md)

### Configuration

```
ctx config                               Show effective configuration with sources
ctx config --global                      Show only global config file
ctx config --local                       Show only local config file
ctx config --debug true|false            Enable or disable verbose hook logging
```

### Diagnostics

```
ctx doctor                               Check hooks, binary path, claude availability
ctx logs                                 Show last 20 hook log entries
ctx logs -n <N>                          Show last N entries
ctx logs --all                           Show all entries
```

### Maintenance

```
ctx update                               Update to the latest version
ctx reset                                Interactively clear snapshots
ctx uninstall                            Remove ctx completely (hooks, data, binary)
ctx version                              Show version
ctx changelog                            Show changes in the current version
ctx changelog --full                     Show full changelog history
```

---

## Configuration

ctx uses two config layers that merge field-by-field. Local values win over global.

**Global config** — `~/.config/ctx/config.yml` (Windows: `%APPDATA%\ctx\config.yml`)

```yaml
core:
  debug: false
  claude_timeout: 30     # seconds; default 30

agents:
  mode: off              # off | on
  workspaces:
    - ~/dev
    - ~/projects
  scan:
    max_depth: 3         # default 3
    extra_root_markers:  # extend built-in markers
      - "*.csproj"
    extra_boundary_dirs: # extend built-in boundaries
      - ".terraform"
    exclude:             # always skip these paths
      - ~/dev/scratch
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
# or with a preset:
ctx init --local --agents on
```

`.ctx/` is automatically added to `.gitignore` — local config is a developer preference, not a team setting.

**Project state config** — controls what ctx captures at compaction time:

```yaml
project_state:
  enabled: true
  git: true
  typecheck:
    enabled: true
    timeout_seconds: 20
  tests:
    enabled: false       # opt-in — can be slow
    timeout_seconds: 60
    max_failed_names: 5
  max_dirty_files: 10
  max_errors: 5
```

> **Note for Docker-based Go projects:** ctx runs typecheck commands on the host. If your Go build runs inside Docker and Go is not installed locally, `go build ./...` will fail. Disable typecheck in your local config:
> ```yaml
> project_state:
>   typecheck:
>     enabled: false
> ```

View effective configuration (with source per field):

```sh
ctx config

effective configuration
───────────────────────────────────────
core.debug              false      [default]
agents.mode             on         [local]   ← override

global:  ~/.config/ctx/config.yml
local:   /home/agus/projects/myapp/.ctx/config.yml
```

---

## Subagent Capture

When you use Claude Code with subagents (general-purpose agents or custom agents defined in `.claude/agents/`), ctx can capture their activity as human-readable snapshots. These are **not** injected into Claude's context — they exist for you to read, review, and share.

Enable agent capture:

```sh
ctx agents --on
ctx init    # re-register hooks to include SubagentStop
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
ctx agents show feature-RES-219-20260313-150405

# all agents at once
ctx agents show --all

# filter to last 2 days
ctx agents show --all --since 2d
```

Get an AI-generated digest across all agents:

```sh
ctx agents summarize
ctx agents summarize --all --since 1w
```

By default, `summarize` only includes agents from the current (non-archived) set.
If archived agents exist, ctx will prompt you to include them. Use `--all` to always include them.

Manage old snapshots:

```sh
ctx agents rm feature-RES-219-20260313-150405   # specific agent
ctx agents rm --before 7d                        # older than 7 days
ctx agents rm --session 20260313-150405          # entire archived session
ctx agents rm --all                              # everything
```

Agent snapshots are grouped by session. When a compaction happens, current agents are archived under a timestamp slot — accessible later with `ctx agents archive`.

---

## Workspace Scanning

When you run `ctx agents` from a directory that isn't a recognized project root (e.g. a parent directory), ctx can scan configured workspace directories for projects automatically.

```sh
ctx agents workspace add ~/dev        # scan ~/dev for projects
ctx agents workspace add ~/projects
```

A directory is identified as a project root when it contains any of these files:

`.git` · `go.mod` · `package.json` · `Cargo.toml` · `pyproject.toml` · `setup.py` ·
`pom.xml` · `build.gradle` · `build.gradle.kts` · `composer.json` · `Gemfile` ·
`CMakeLists.txt` · `*.sln`

Scanning stops at boundary directories — it never descends into:

`vendor` · `node_modules` · `__pycache__` · `target` · `dist` · `.next` · `.gradle`

Add custom markers and boundaries for your stack:

```sh
ctx agents workspace marker add "*.csproj"        # .NET projects
ctx agents workspace boundary add ".terraform"    # Terraform workspaces
ctx agents workspace exclude ~/dev/scratch        # always skip this path
```

→ Full reference: [docs/workspace-scanning.md](docs/workspace-scanning.md)

---

## Snapshot Format

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

Snapshots are stored at:
- Linux/macOS: `~/.local/share/ctx/{project-hash}/snapshot.md`
- Windows: `%LOCALAPPDATA%\ctx\{project-hash}\snapshot.md`

---

## Debugging

```sh
ctx doctor                     Check hooks, binary path, claude availability, config
ctx logs                       Show last 20 hook log entries
ctx logs -n 50                 Show last 50 entries
ctx logs --all                 Show all entries
ctx config --debug true        Enable verbose logging (context sizes, timings)
```

Log file:
- Linux/macOS: `~/.local/share/ctx/debug.log`
- Windows: `%LOCALAPPDATA%\ctx\debug.log`

---

## Updates

ctx updates itself automatically in the background. Once every 24 hours it checks for a new release, downloads it silently, and applies it on the next invocation — you'll see a one-line notification when it does.

```sh
ctx update        # update immediately
ctx changelog     # see what changed
```

---

## What ctx is NOT

ctx is not a memory tool. It doesn't accumulate knowledge across sessions, index conversations, or build a searchable history. It solves one specific problem: **keeping the current session coherent when context gets compacted**. One project, one snapshot, always overwritten.

Agents mode captures subagent activity as human-readable snapshots for review and handoff. They are never injected into Claude's context — the main snapshot is the only thing Claude sees.

---

## Development

Requires Docker (no local Go installation needed):

```sh
make build              # Build binary
make test               # Run tests
make install            # Build + install locally
make cross              # Cross-compile all platforms
make changelog          # Regenerate CHANGELOG.md (requires git-cliff)
make release            # Auto-detect bump and release
make release-patch / release-minor / release-major
```

## License

MIT
