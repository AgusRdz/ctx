# Changelog

All notable changes to ctx are documented here.

## [0.3.0] - 2026-03-12

### Features
- XDG-compliant global config at ~/.config/ctx/config.yml
- Local project config at {project}/.ctx/config.yml with field-level override
- New: ctx agents — capture subagent activity and inject into main agent context
  - --v1: capture final output on SubagentStop
  - --v2: capture internal state (PreCompact) + final output
  - --off: default, main agent only (no behavior change)
  - --local flag for project-level override
- ctx config now shows effective config with [global]/[local] source per field
- ctx init --local creates project config template
- ctx clear --agents-only clears only agent snapshots
- Background auto-update check (every 24h, silent)
- ctx changelog command

### Bug Fixes
- Strip markdown code fences from claude -p JSON response

## [0.2.4] - 2026-03-10

### Bug Fixes
- Strip markdown code fences from claude -p JSON response

## [0.2.3] - 2026-03-09

### Features
- ctx logs -n / --all flags
- Sort ctx list by age, log total count

### Bug Fixes
- Eliminate double ctx: prefix
- Add missing tests, fix doctor spacing

## [0.2.2] - 2026-03-08

### Features
- Staleness warning for old snapshots
- Updated README with full documentation
