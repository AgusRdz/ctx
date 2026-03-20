// Package tui provides minimal terminal color helpers for ctx output.
//
// Color is disabled automatically when:
//   - The NO_COLOR environment variable is set (https://no-color.org)
//   - The target stream is not an interactive terminal (piped, redirected, CI)
package tui

import (
	"fmt"
	"os"
	"sync"

	"github.com/mattn/go-isatty"
)

var (
	stdoutOnce    sync.Once
	stdoutEnabled bool
	stderrOnce    sync.Once
	stderrEnabled bool
)

// enabledFor returns true if color should be used when writing to f.
// Uses go-isatty which correctly handles MSYS2/Git Bash (IsCygwinTerminal)
// in addition to native terminals.
func enabledFor(f *os.File) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	fd := f.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

// stdoutEnabled_ reports whether color is enabled for stdout (memoized).
func stdoutColor() bool {
	stdoutOnce.Do(func() { stdoutEnabled = enabledFor(os.Stdout) })
	return stdoutEnabled
}

// stderrColor reports whether color is enabled for stderr (memoized).
func stderrColor() bool {
	stderrOnce.Do(func() { stderrEnabled = enabledFor(os.Stderr) })
	return stderrEnabled
}

func ansi(code, s string) string {
	if !stdoutColor() {
		return s
	}
	return "\033[" + code + "m" + s + "\033[0m"
}

func ansiErr(code, s string) string {
	if !stderrColor() {
		return s
	}
	return "\033[" + code + "m" + s + "\033[0m"
}

// Bold returns s in bold (stdout).
func Bold(s string) string { return ansi("1", s) }

// Dim returns s in dim/faint (stdout).
func Dim(s string) string { return ansi("2", s) }

// Green returns s in green (stdout).
func Green(s string) string { return ansi("32", s) }

// Yellow returns s in yellow (stdout).
func Yellow(s string) string { return ansi("33", s) }

// Cyan returns s in cyan (stdout).
func Cyan(s string) string { return ansi("36", s) }

// Red returns s in red (stdout).
func Red(s string) string { return ansi("31", s) }

// BoldErr returns s in bold, for writing to stderr.
func BoldErr(s string) string { return ansiErr("1", s) }

// ColorDebug prints terminal detection diagnostics to stdout.
func ColorDebug() {
	check := func(name string, f *os.File) {
		fd := f.Fd()
		fmt.Printf("%s (fd=%d):\n", name, fd)
		fmt.Printf("  IsTerminal:       %v\n", isatty.IsTerminal(fd))
		fmt.Printf("  IsCygwinTerminal: %v\n", isatty.IsCygwinTerminal(fd))
		fmt.Printf("  color enabled:    %v\n", enabledFor(f))
	}
	fmt.Printf("NO_COLOR=%q\n", os.Getenv("NO_COLOR"))
	fmt.Printf("TERM=%q\n", os.Getenv("TERM"))
	fmt.Printf("WT_SESSION=%q\n", os.Getenv("WT_SESSION"))
	fmt.Printf("COLORTERM=%q\n\n", os.Getenv("COLORTERM"))
	check("stdout", os.Stdout)
	check("stderr", os.Stderr)
	fmt.Printf("\nsample: %s %s %s %s\n", Bold("bold"), Green("green"), Yellow("yellow"), Cyan("cyan"))
}
