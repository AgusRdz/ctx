package snapshot

import (
	"os"
	"path/filepath"
	"strings"
)

// defaultRootMarkers are filenames (exact or glob) whose presence in a directory
// identifies it as a project root. Scanning stops and records the directory.
var defaultRootMarkers = []string{
	// Version control
	".git",
	// Go
	"go.mod",
	// JavaScript / TypeScript
	"package.json",
	// Rust
	"Cargo.toml",
	// Python
	"pyproject.toml",
	"setup.py",
	// JVM (Maven / Gradle)
	"pom.xml",
	"build.gradle",
	"build.gradle.kts",
	// PHP
	"composer.json",
	// Ruby
	"Gemfile",
	// C/C++
	"CMakeLists.txt",
	// .NET — glob patterns are supported
	"*.sln",
}

// defaultBoundaryDirs are directory names to never descend into.
// They indicate you are already inside a project's dependency or build tree.
var defaultBoundaryDirs = []string{
	"vendor",       // Go, PHP
	"node_modules", // Node
	"__pycache__",  // Python
	"target",       // Rust, Java (Maven)
	"dist",         // generic build output
	".next",        // Next.js build
	".gradle",      // Gradle cache
}

const defaultMaxDepth = 3

// ScanOptions carries user-defined additions and exclusions for workspace scanning.
type ScanOptions struct {
	// MaxDepth overrides the default scan depth (3). 0 means use the default.
	MaxDepth int
	// ExtraRootMarkers are appended to defaultRootMarkers (exact names or globs).
	ExtraRootMarkers []string
	// ExtraBoundaryDirs are appended to defaultBoundaryDirs.
	ExtraBoundaryDirs []string
	// Exclude is a list of paths (~ supported) to always skip.
	Exclude []string
}

// ScanWorkspaceProjects walks each workspace directory up to MaxDepth levels,
// identifies project roots using root markers, and returns their absolute paths.
//
// A directory is a project root when any of its direct entries matches a root
// marker (exact filename or glob pattern such as "*.sln").
//
// Scanning never descends into boundary dirs or hidden dirs, and skips any
// path listed in opts.Exclude.
func ScanWorkspaceProjects(workspaces []string, opts ScanOptions) ([]string, error) {
	maxDepth := defaultMaxDepth
	if opts.MaxDepth > 0 {
		maxDepth = opts.MaxDepth
	}

	allMarkers := append(defaultRootMarkers, opts.ExtraRootMarkers...)
	boundarySet := toBoundarySet(opts.ExtraBoundaryDirs)
	excludeSet := toExcludeSet(opts.Exclude)

	seen := make(map[string]bool)
	var results []string
	for _, ws := range workspaces {
		abs := AbsExpandHome(ws)
		if abs == "" {
			continue
		}
		scanDir(abs, 0, maxDepth, allMarkers, boundarySet, excludeSet, seen, &results)
	}
	return results, nil
}

func scanDir(
	dir string,
	depth, maxDepth int,
	rootMarkers []string,
	boundarySet map[string]bool,
	excludeSet map[string]bool,
	seen map[string]bool,
	results *[]string,
) {
	if seen[dir] || excludeSet[dir] {
		return
	}
	seen[dir] = true

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	// Check whether this directory is a project root.
	for _, e := range entries {
		if matchesAnyMarker(e.Name(), rootMarkers) {
			*results = append(*results, dir)
			return // don't recurse into project subdirectories
		}
	}

	if depth >= maxDepth {
		return
	}

	// Recurse into subdirectories, skipping hidden and boundary dirs.
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") || boundarySet[name] {
			continue
		}
		sub := filepath.Join(dir, name)
		if excludeSet[sub] {
			continue
		}
		scanDir(sub, depth+1, maxDepth, rootMarkers, boundarySet, excludeSet, seen, results)
	}
}

// matchesAnyMarker returns true if name equals any marker exactly,
// or matches any marker that contains a glob wildcard.
func matchesAnyMarker(name string, markers []string) bool {
	for _, m := range markers {
		if strings.ContainsAny(m, "*?[") {
			if ok, _ := filepath.Match(m, name); ok {
				return true
			}
		} else if name == m {
			return true
		}
	}
	return false
}

func toBoundarySet(extra []string) map[string]bool {
	set := make(map[string]bool, len(defaultBoundaryDirs)+len(extra))
	for _, d := range defaultBoundaryDirs {
		set[d] = true
	}
	for _, d := range extra {
		set[d] = true
	}
	return set
}

func toExcludeSet(paths []string) map[string]bool {
	set := make(map[string]bool, len(paths))
	for _, p := range paths {
		abs := AbsExpandHome(p)
		if abs != "" {
			set[abs] = true
		}
	}
	return set
}

// AbsExpandHome expands a leading ~ to the home directory and returns the
// absolute path. Handles both ~/... and ~\... (Windows) safely.
// Returns "" on error.
func AbsExpandHome(path string) string {
	// Normalize to forward slashes so we handle both ~/... and ~\... uniformly.
	normalized := filepath.ToSlash(path)
	if normalized == "~" || strings.HasPrefix(normalized, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		if normalized == "~" {
			path = home
		} else {
			// filepath.Join produces OS-native separators from the forward-slash suffix.
			path = filepath.Join(home, normalized[2:])
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	return abs
}

// ShortenToHome converts an absolute path to a ~/... form when it falls under
// the given home directory. Always uses forward slashes after ~ so the result
// is portable across OSes and safe to store in YAML.
// Returns the original path unchanged when it is not under home.
func ShortenToHome(abs, home string) string {
	if home == "" {
		return abs
	}
	sep := string(filepath.Separator)
	switch {
	case abs == home:
		return "~"
	case strings.HasPrefix(abs, home+sep):
		rel := abs[len(home)+len(sep):]
		return "~/" + filepath.ToSlash(rel)
	}
	return abs
}
