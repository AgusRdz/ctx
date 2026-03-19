# Workspace Scanning

ctx can automatically discover projects across your workspace directories so `ctx agents` works from any directory — not just from inside a project root.

---

## The Problem

`ctx agents` normally operates on the current project (identified by its git root). If you run it from a parent directory like `~/dev`, it finds nothing — there's no project there.

Workspace scanning solves this: configure a list of workspace directories, and ctx will search them for project roots and show agents across all of them.

---

## Setup

```sh
ctx agents workspace add ~/dev
ctx agents workspace add ~/projects
```

Now running `ctx agents` from any directory will fall back to a workspace scan when no project is found at the current location.

View the current configuration:

```sh
ctx agents workspace list

workspaces:
  ~/dev
  ~/projects

excluded paths:
  ~/dev/scratch
```

---

## How Projects Are Identified

A directory is treated as a project root when it directly contains any of these files:

| Ecosystem | Markers |
|-----------|---------|
| Version control | `.git` |
| Go | `go.mod` |
| JavaScript / TypeScript | `package.json` |
| Rust | `Cargo.toml` |
| Python | `pyproject.toml`, `setup.py` |
| Java (Maven / Gradle) | `pom.xml`, `build.gradle`, `build.gradle.kts` |
| PHP | `composer.json` |
| Ruby | `Gemfile` |
| C / C++ | `CMakeLists.txt` |
| .NET | `*.sln` (glob) |

When a root marker is found, scanning **stops and records** that directory. It does not descend into project subdirectories.

---

## Boundary Directories

Scanning never descends into these directories — they signal you're already inside a project's dependency or build tree:

| Directory | Ecosystem |
|-----------|-----------|
| `vendor` | Go, PHP |
| `node_modules` | Node.js |
| `__pycache__` | Python |
| `target` | Rust, Java (Maven) |
| `dist` | Generic build output |
| `.next` | Next.js build |
| `.gradle` | Gradle cache |

Hidden directories (names starting with `.`) are also never descended into.

---

## Custom Markers and Boundaries

Extend the built-in lists for your stack without replacing them:

```sh
# Root markers
ctx agents workspace marker add "*.csproj"     # .NET projects
ctx agents workspace marker add ".hg"          # Mercurial repos
ctx agents workspace marker rm "*.csproj"      # remove a marker

# Boundary directories
ctx agents workspace boundary add ".terraform" # Terraform workspaces
ctx agents workspace boundary add "build"      # custom build output
ctx agents workspace boundary rm "dist"        # re-enable descent into dist
```

These are stored in `~/.config/ctx/config.yml` (Windows: `%APPDATA%\ctx\config.yml`):

```yaml
agents:
  scan:
    extra_root_markers:
      - "*.csproj"
      - ".hg"
    extra_boundary_dirs:
      - ".terraform"
      - "build"
```

---

## Excluding Paths

Always skip specific directories during scans, regardless of what markers they contain:

```sh
ctx agents workspace exclude ~/dev/scratch
ctx agents workspace exclude ~/dev/archived-projects
ctx agents workspace unexclude ~/dev/scratch
```

Excluded paths are stored as `~/...` when under your home directory, so they work portably across machines with different home paths.

---

## Scan Depth

By default, ctx scans up to **3 levels deep** below each workspace root. Override this globally:

```yaml
# ~/.config/ctx/config.yml
agents:
  scan:
    max_depth: 5
```

Or via the config file directly — there is no CLI flag for depth.

---

## Full Config Reference

```yaml
agents:
  workspaces:
    - ~/dev
    - ~/projects
  scan:
    max_depth: 3           # 0 = use default (3)
    extra_root_markers:    # extend built-in markers (globs supported)
      - "*.csproj"
    extra_boundary_dirs:   # extend built-in boundary dirs
      - ".terraform"
    exclude:               # always skip these paths (~ supported)
      - ~/dev/scratch
```

---

## How Scanning Works

```
~/dev/                          ← workspace root (depth 0)
├── myapp/
│   └── go.mod                  ← project root found → recorded, stop here
├── frontend/
│   └── package.json            ← project root found → recorded, stop here
├── infra/
│   ├── .terraform/             ← boundary dir → skip
│   └── modules/
│       └── vpc/
│           └── main.tf         ← no marker → continue
└── scratch/                    ← excluded → skip
```

Each discovered project is matched against its stored snapshot hash. Projects with no captured agents are silently skipped in the output — only projects with activity are shown.
