# Contributing to ctx

Thanks for your interest in contributing!

## Keeping your fork up to date

PRs submitted from a stale branch conflict with recent changes and end up applied manually, showing as "Closed" instead of "Merged". Follow these steps every time before starting new work.

### Option A — upstream remote (recommended)

**One-time setup:**

```bash
git remote add upstream https://github.com/AgusRdz/ctx.git
```

**Before every PR:**

```bash
# 1. Fetch latest from upstream
git fetch upstream

# 2. Rebase your local main
git rebase upstream/main

# 3. Push to your fork
git push origin main

# 4. Create your feature branch from the updated main
git checkout -b feat/my-feature
```

### Option B — GitHub UI sync

If you use the **"Sync fork"** button on GitHub, you still need to pull locally before branching — otherwise your local copy is still on the old commit:

```bash
# 1. Click "Sync fork" on your fork's GitHub page

# 2. Pull the sync down locally
git pull origin main

# 3. Create your feature branch
git checkout -b feat/my-feature
```

## Development setup

ctx builds inside Docker — no local Go installation required.

```bash
# Run tests
make test

# Build binary
make build

# Run a single test
docker compose run --rm dev go test ./snapshot/ -run TestRead -v

# Coverage report
make coverage
```

## Project structure

```
cmd/              CLI entry point and all subcommand handlers
hooks/            PreCompact, SessionStart, SubagentStop hook handlers
snapshot/         Snapshot read/write, store, transcript parsing, generator
agents/           Subagent capture, git-root resolution, combined summaries
config/           Configuration loading (global + local, field-level merge)
install/          Hook registration in ~/.claude/settings.json
updater/          Self-update mechanism
logging/          Debug log writer
```

Each package has a single responsibility. When adding a feature, identify which package owns it — avoid putting logic in `cmd/` that belongs in a domain package.

## Adding a new command

1. Add the case to the `switch` in `cmd/main.go`
2. Implement the handler function in `cmd/main.go` (or a new file under `cmd/` if it's large)
3. Add the command to `printUsage()`
4. Write tests in the appropriate package (not in `cmd/`)

## Adding a new hook

1. Create `hooks/<hookname>.go` with a `Run<HookName>()` function
2. Register it in `cmd/main.go`'s `cmdHook()` switch
3. Register the hook type in `install/installer.go` if it requires a new Claude Code hook event

## Commit conventions

This project uses [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add ctx agents summarize command
fix: normalize subagent dir to git root before hashing
perf: compress transcript before sending to claude -p
test: add coverage for ReadAllAgentSnapshots with archive
docs: update README for v0.7.0 agents redesign
chore: remove dead inject_on_start config fields
ci: bump actions/checkout to v5
```

These feed directly into the changelog via [git-cliff](https://git-cliff.org/).

## Guidelines

- **One concern per package** — keep hook logic in `hooks/`, storage in `snapshot/`, etc.
- **Errors wrapped with context** — `fmt.Errorf("ctx: %w", err)` everywhere
- **No prints inside hooks** — hooks output only JSON or exit codes; use `logging.Log` for debug info
- **Fail gracefully** — hooks should never crash Claude Code; log the error and return `nil` for non-fatal issues
- **Keep dependencies minimal** — stdlib only where possible; the only external deps are yaml and the test suite
