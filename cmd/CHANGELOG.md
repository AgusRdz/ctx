# Changelog

All notable changes to ctx are documented here.

## [1.1.1] - 2026-03-19

### Bug Fixes
- Enable color in Git Bash / MSYS2 via TERM and WT_SESSION fallback
([d281f38](https://github.com/AgusRdz/ctx/commit/d281f3884ccc9a06ee721f72d22e895b25a46c03))

### Miscellaneous
- Add .gitattributes to enforce LF line endings
([58be0fd](https://github.com/AgusRdz/ctx/commit/58be0fde14bf04ef15ae253fae483bdedacfcc9f))
## [1.1.0] - 2026-03-19

### Features
- Workspace scanning, color output, security hardening
([db693d4](https://github.com/AgusRdz/ctx/commit/db693d41957b3c1a7e6cb3b28a1cc63c3cb5bcbf))

### Miscellaneous
- Release v1.1.0
([9be95ab](https://github.com/AgusRdz/ctx/commit/9be95abf5f000a100d55da423801baf6cd1124cd))
## [1.0.0] - 2026-03-17

### Features
- V1.0.0 — summarize archive prompt, remove dead inject code
([ac48d11](https://github.com/AgusRdz/ctx/commit/ac48d113cb4afde55db98cfaeea3c47c6e27b102))
## [0.7.10] - 2026-03-17

### CI/CD
- Set run-name to Release <version> for cleaner Actions list
([6452231](https://github.com/AgusRdz/ctx/commit/6452231b1109bd13f444cc64a49b853ef4f58410))

### Features
- Ctx agents --global lists agents across all projects
([90cea27](https://github.com/AgusRdz/ctx/commit/90cea27094c52a9fc5530bbd71b8d260c76f1bf8))
## [0.7.9] - 2026-03-16

### Bug Fixes
- Ctx config --local now shows local config file
([5b63a88](https://github.com/AgusRdz/ctx/commit/5b63a888da8db78b255baa713e783e31f89e8a21))

### CI/CD
- Skip homebrew commit when formula is unchanged
([4eeee95](https://github.com/AgusRdz/ctx/commit/4eeee95e904595d0e91d73ea84d5784438e4647b))
## [0.7.8] - 2026-03-16

### Bug Fixes
- Never overwrite live binary until verification passes
([f5d2b1e](https://github.com/AgusRdz/ctx/commit/f5d2b1e4d6c839975f323bba3bf43a5bf972ef97))
## [0.7.7] - 2026-03-16

### Bug Fixes
- Switch signature encoding from xxd hex to base64
([570d7de](https://github.com/AgusRdz/ctx/commit/570d7de8288507092d56d0b35167e43388df54e2))
## [0.7.6] - 2026-03-16

### Miscellaneous
- Ignore signing_key.pem and coverage.out
([cffb34c](https://github.com/AgusRdz/ctx/commit/cffb34c2c71f8ebfb5b8ea0b6991dab80e1dc83f))

### Testing
- Add TestDownloadAndVerify_MissingSignature
([653c25a](https://github.com/AgusRdz/ctx/commit/653c25a8a0dfad8c204aa1f2383bf4bee3057039))
## [0.7.5] - 2026-03-16

### CI/CD
- Fix signing — base64-decode secret, rotate key pair
([b8c2360](https://github.com/AgusRdz/ctx/commit/b8c23601e8ba3dad8eb157c37cef16ebb1645c33))
## [0.7.4] - 2026-03-16

### CI/CD
- Fix signing step — pipefail, key validation, size check
([2212bc1](https://github.com/AgusRdz/ctx/commit/2212bc144dbeba6e2b227798c83a802f5470cfd9))
## [0.7.3] - 2026-03-16

### Features
- Verify checksums and signature on self-update
([cb04ccb](https://github.com/AgusRdz/ctx/commit/cb04ccb7b05e62510c707a54f2f0ac34462a1f8a))
## [0.7.2] - 2026-03-16

### CI/CD
- Stale-check workflow, signing, CONTRIBUTING, PR template, coverage
([8c43a41](https://github.com/AgusRdz/ctx/commit/8c43a41a01747552a527373e0f6bdc9e7b795bcb))
## [0.7.1] - 2026-03-16

### Bug Fixes
- Auto-register shell PATH and add logo to README
([72c7602](https://github.com/AgusRdz/ctx/commit/72c7602acee3289d0f0834fbdb30176e07fff23d))

### Features
- Agents redesign — human-readable dumps, git-root scoping, rm/summarize
([23b427d](https://github.com/AgusRdz/ctx/commit/23b427d4f1f006e217f65ce19944c487245dffed))

### Miscellaneous
- Add logo.png
([7b7c840](https://github.com/AgusRdz/ctx/commit/7b7c840b121f874d670b65d01acbd516c20e23b8))
- Remove dead inject config fields, update README for v0.7.0
([d53e74e](https://github.com/AgusRdz/ctx/commit/d53e74e8beb7414e3c99bfd637c799c475a6e8c1))
## [0.6.0] - 2026-03-13

### CI/CD
- Use git-cliff action for changelog and release notes
([3bfb4bb](https://github.com/AgusRdz/ctx/commit/3bfb4bb4b02d1c8d832939f6b99a9d65a5190470))

### Features
- Ctx agents inject — cross-repo context injection from agent snapshots
([56e5dba](https://github.com/AgusRdz/ctx/commit/56e5dbae6f32583eb8b75830d619616993978364))
## [0.5.0] - 2026-03-13

### Features
- Agents v2 — transcript capture, branch naming, archive on compaction
([ae0847d](https://github.com/AgusRdz/ctx/commit/ae0847d44c681c2fcf354aa66acae0d097a14f5c))
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

