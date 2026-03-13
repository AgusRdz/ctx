# Changelog

All notable changes to ctx are documented here.

## [0.5.0] - 2026-03-13

### Features
- Agent capture v2: read sub-agent transcript at SubagentStop, summarize via claude -p — no longer requires compaction to get rich output
- Agent naming: snapshots now use `{git-branch}-{YYYYMMDD-HHMMSS}` instead of raw session IDs
- Archive on compaction: PreCompact moves current agents to `agents/archive/YYYYMMDD-HHMMSS/` before writing new snapshot
- New commands: `ctx agents show <name>` (searches current + archive), `ctx agents archive` (lists archived sessions)
- Simplified agent mode: v1/v2 replaced by `on`/`off`; existing v1/v2 configs auto-migrate

### Breaking Changes
- `ctx agents --v1` and `ctx agents --v2` removed — use `ctx agents --on`

## [0.4.0] - 2026-03-13

### Documentation
- Add brew install option
([4d14fe0](https://github.com/AgusRdz/ctx/commit/4d14fe0e6e35f3adc2951e6110204295b52ee09c))
- Transcript compression, build attestations
([2b60e10](https://github.com/AgusRdz/ctx/commit/2b60e10548167b3c89b777b3b4fde35aacd30e87))

### Features
- Attestations, brew workflow, transcript compression, stripCodeFences guard
([acb6de0](https://github.com/AgusRdz/ctx/commit/acb6de02857064282fcc44e43ee7b57ed575cb70))
## [0.3.0] - 2026-03-13

### Features
- Ctx v0.3.0 — XDG config, local config, agents, changelog, auto-update
([b7a2ca0](https://github.com/AgusRdz/ctx/commit/b7a2ca0271c3e7689a0d1e06accd9105029f3120))
## [0.2.4] - 2026-03-10

### Bug Fixes
- Strip markdown code fences from claude -p JSON response
([502cb26](https://github.com/AgusRdz/ctx/commit/502cb26d6d5640a2860cb556169522f8dd4954bd))
## [0.2.3] - 2026-03-10

### Bug Fixes
- Sort ctx list by age, logs total count, windows/arm64, checksums, Makefile build target
([005cd0f](https://github.com/AgusRdz/ctx/commit/005cd0f76a2a0ac0fb6af2809c7cf4947d2fb70d))
## [0.2.2] - 2026-03-10

### Features
- Ctx logs -n / --all flags, fix install.sh next steps
([46d7962](https://github.com/AgusRdz/ctx/commit/46d7962262684b3c05834e86661c91bbad8048b4))
## [0.2.1] - 2026-03-10

### Bug Fixes
- Eliminate double ctx: prefix, add missing tests, fix doctor spacing
([5a27964](https://github.com/AgusRdz/ctx/commit/5a279646df04f37154b8721961f10d6657a8ff2b))
## [0.2.0] - 2026-03-10

### Features
- Staleness warning, updated README, tests for config and collector
([1cad13e](https://github.com/AgusRdz/ctx/commit/1cad13e705a6e81dbb353e0c84baa9b15c9d7b9f))
## [0.1.9] - 2026-03-10

### Bug Fixes
- Claude -p timeout, smart CLAUDE.md extraction, doctor debug status, legacy snapshot hint
([ebd5b35](https://github.com/AgusRdz/ctx/commit/ebd5b356f5eba5d4979be9f05638db362b148d75))
## [0.1.8] - 2026-03-10

### Bug Fixes
- Implement ctx config --debug, fix Clear() to remove full project dir
([8e140f0](https://github.com/AgusRdz/ctx/commit/8e140f0dcb396f4f069ec41e67188104e827d259))
## [0.1.7] - 2026-03-10

### Features
- Transcript parsing, timestamps, token budget, list/config/show --project commands
([592fea5](https://github.com/AgusRdz/ctx/commit/592fea5ff97c53bb08f0d6ce18ac82242cdc6494))
## [0.1.6] - 2026-03-09

### Documentation
- Rewrite README to clarify ctx is about session fidelity, not memory
([f80d27a](https://github.com/AgusRdz/ctx/commit/f80d27a3e1c53379166c3fcf0fc16c2e24fcedfe))

### Features
- Add doctor, logs, reset commands and fix fallback goal inference
([22eab57](https://github.com/AgusRdz/ctx/commit/22eab572743a61d12f2e1041352835d774173563))
## [0.1.5] - 2026-03-09

### Features
- Add ctx uninstall command
([4f66c08](https://github.com/AgusRdz/ctx/commit/4f66c08402259a31e41aaa1ece7205314664e047))
## [0.1.4] - 2026-03-09

### Bug Fixes
- Reliable Windows PATH update in install script
([40b46ea](https://github.com/AgusRdz/ctx/commit/40b46ea96794c858a71a8e2ccae3e80f06637327))
## [0.1.3] - 2026-03-09

### Bug Fixes
- Auto-add to Windows PATH via PowerShell during install
([90b75a0](https://github.com/AgusRdz/ctx/commit/90b75a031bbee32893d31387e9c1547ab41d266f))
## [0.1.2] - 2026-03-09

### Bug Fixes
- Install to AppData/Local/Programs/ctx on Windows
([febadd9](https://github.com/AgusRdz/ctx/commit/febadd9461bcc3003093850a7af622452778fa6d))
## [0.1.1] - 2026-03-09

### Bug Fixes
- Status check uses substring match instead of full path
([8d1188c](https://github.com/AgusRdz/ctx/commit/8d1188c4759b6e745cd40cb128f74ee92c03cbbf))
## [0.1.0] - 2026-03-09

### Bug Fixes
- Align hook structs with Claude Code's actual JSON contract
([fa1fe9e](https://github.com/AgusRdz/ctx/commit/fa1fe9eeb190b1955d74d7a4e9eebf89c18d7f2f))

### Features
- Initial scaffold for ctx CLI
([0357ee2](https://github.com/AgusRdz/ctx/commit/0357ee2059b0d8848cc3ac69ff9b48e6943e513c))
- Full e2e hook pipeline with version, Makefile, and claude -p fix
([f9fcd1a](https://github.com/AgusRdz/ctx/commit/f9fcd1a5286f34729037401389c18c20a1113ecc))
- Add CI, release workflow, self-updater, and install script
([266d7bb](https://github.com/AgusRdz/ctx/commit/266d7bbdc8d753a2721d4a579806db2f47d34d79))

### Testing
- Add unit tests for store, generator, transcript, and installer
([5429fbe](https://github.com/AgusRdz/ctx/commit/5429fbe9dd535b67eb05262ef37fd28f3ee4d35f))

