# ctx

<p align="center">
  <img src="logo.png" alt="ctx logo" width="200" />
</p>

**Preserve Claude Code session context across compactions — automatically.**

When Claude Code hits its context window limit, it compacts the conversation. This is necessary, but lossy: the current goal, technical decisions, the file you were editing, what to do next — all of it can get diluted or lost. After two or three compactions, Claude may forget what you were building, repeat work, or contradict decisions you already made together.

ctx hooks into Claude Code's **PreCompact**, **PostCompact**, and **SessionStart** events to capture and restore a structured snapshot of your working context, every time. Snapshots are scoped per branch, so parallel sessions on different branches never collide.

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
│                  → writes snapshot (scoped to branch)        │
│                                                              │
│  PostCompact hook → ctx reads the snapshot just written      │
│                   → prints it to stdout                      │
│                   → Claude Code re-injects it immediately    │
│                                                              │
│  New session on same branch                                  │
│       │                                                      │
│       ▼                                                      │
│  SessionStart hook → ctx finds snapshot for project+branch   │
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
ctx init                  # register PreCompact + PostCompact + SessionStart hooks
ctx show                  # print the current snapshot (current branch)
ctx list                  # list all projects and branches with snapshots
```

---

## Commands

### Setup

```
ctx init                                 Install PreCompact, PostCompact, and SessionStart hooks
ctx init --remove                        Remove ctx hooks
ctx init --status                        Check hook installation status
ctx init --local                         Create local project config (.ctx/config.yml)
```

### Session

```
ctx show                                 Print snapshot for current branch
ctx show --project <path>                Print snapshot for a specific project
ctx state                                Capture and print current project state
ctx clear                                Delete snapshot for current branch
ctx clear --all                          Delete all branch snapshots for this project
ctx list                                 List all projects and branches with snapshots
```

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
```

**Local config** — `{project}/.ctx/config.yml` (optional, project-level overrides)

Create a local config with:

```sh
ctx init --local
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
    command: ""          # custom command overrides auto-detection (see below)
  tests:
    enabled: false       # opt-in — can be slow
    timeout_seconds: 60
    max_failed_names: 5
    command: ""          # custom command overrides auto-detection (see below)
  max_dirty_files: 10
  max_errors: 5
```

Auto-detection for **typecheck** covers `tsc` (tsconfig.json) and `go build` (go.mod). Auto-detection for **tests** covers jest, vitest, and go test. For everything else, set a custom command — ctx runs it, checks the exit code (0 = passed/ok, non-zero = failed/errors), and shows the last few output lines on failure:

```yaml
# phpstan (PHP)
project_state:
  typecheck:
    enabled: true
    command: "vendor/bin/phpstan analyse --no-progress"

# mypy (Python)
project_state:
  typecheck:
    enabled: true
    command: "mypy src --no-error-summary"

# clippy (Rust)
project_state:
  typecheck:
    enabled: true
    command: "cargo clippy --quiet 2>&1"

# dotnet build (C#)
project_state:
  typecheck:
    enabled: true
    command: "dotnet build --no-restore -v minimal"
```

```yaml
# pytest
project_state:
  tests:
    enabled: true
    command: "pytest -q --tb=no"

# cargo test
project_state:
  tests:
    enabled: true
    command: "cargo test --quiet 2>&1"

# dotnet test
project_state:
  tests:
    enabled: true
    command: "dotnet test --no-build --logger 'console;verbosity=minimal'"

# rspec
project_state:
  tests:
    enabled: true
    command: "bundle exec rspec --format progress"
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

global:  ~/.config/ctx/config.yml
local:   /home/agus/projects/myapp/.ctx/config.yml
```

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

Snapshots are stored per branch:
- Linux/macOS: `~/.local/share/ctx/{project-hash}/{branch}/snapshot.md`
- Windows: `%LOCALAPPDATA%\ctx\{project-hash}\{branch}\snapshot.md`

Two sessions on different branches in the same project write to separate snapshots and never overwrite each other. Non-git directories and detached HEAD use `_` as the branch name.

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
