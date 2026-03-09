# ctx

Preserve Claude Code working context across compactions.

ctx hooks into Claude Code's **PreCompact** and **SessionStart** events to automatically save and restore a structured snapshot of your session - so when context is compacted, you pick up right where you left off.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/AgusRdz/ctx/main/install.sh | sh
```

Or with a specific version:

```sh
CTX_VERSION=v0.1.0 curl -fsSL https://raw.githubusercontent.com/AgusRdz/ctx/main/install.sh | sh
```

## Setup

Register hooks in Claude Code:

```sh
ctx init
```

That's it. ctx will now automatically:
1. **On compaction** - generate a snapshot of your session (goal, decisions, in-progress work, next steps)
2. **On session start** - restore the snapshot as context for Claude

## Commands

```
ctx init              Install hooks in Claude Code
ctx init --remove     Remove hooks
ctx init --status     Check hook installation status
ctx show              Print current snapshot
ctx clear             Delete current snapshot
ctx update            Update to the latest version
ctx version           Show version
```

## How it works

When Claude Code compacts context (automatically or via `/compact`), ctx:

1. Reads the session transcript and git state
2. Calls `claude -p` to generate a semantic summary (goal, decisions, in-progress, next)
3. Saves the snapshot to `~/.local/share/ctx/{project-hash}/snapshot.md` (Linux/macOS) or `%LOCALAPPDATA%/ctx/{project-hash}/snapshot.md` (Windows)

When a new session starts, ctx checks for an existing snapshot and prints it to stdout - Claude Code automatically injects this as context.

If `claude -p` is unavailable, ctx falls back to a deterministic snapshot using git diff/log and CLAUDE.md.

## Snapshot format

```markdown
# Session Context

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

## Development

Requires Docker (no local Go installation needed):

```sh
make build        # Build binary
make test         # Run tests
make install      # Build + install to ~/bin/
make cross        # Cross-compile all platforms
```

## License

MIT
