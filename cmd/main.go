package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/AgusRdz/ctx/config"
	"github.com/AgusRdz/ctx/hooks"
	"github.com/AgusRdz/ctx/install"
	"github.com/AgusRdz/ctx/snapshot"
	"github.com/AgusRdz/ctx/updater"
)

// version is set at build time via -ldflags "-X main.version=..."
var version = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var err error

	switch os.Args[1] {
	case "init":
		err = cmdInit()
	case "hook":
		err = cmdHook()
	case "show":
		err = cmdShow()
	case "clear":
		err = cmdClear()
	case "list":
		err = cmdList()
	case "config":
		cmdConfig()
	case "doctor":
		install.Doctor()
	case "logs":
		err = cmdLogs()
	case "reset":
		err = cmdReset()
	case "uninstall":
		if install.ConfirmUninstall(os.Args[2:]) {
			err = install.Uninstall()
		}
	case "update":
		updater.Run(version)
	case "version", "--version", "-v":
		fmt.Printf("ctx %s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "ctx: unknown command %q\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func cmdInit() error {
	flag := ""
	if len(os.Args) > 2 {
		flag = os.Args[2]
	}

	switch flag {
	case "--remove":
		if err := install.Remove(); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "ctx: hooks removed")
	case "--status":
		fmt.Println(install.Status())
	case "", "--help", "-h":
		if flag == "--help" || flag == "-h" {
			fmt.Fprintln(os.Stderr, "Usage: ctx init [--remove|--status]")
			fmt.Fprintln(os.Stderr, "  (no flag)   Install PreCompact and SessionStart hooks")
			fmt.Fprintln(os.Stderr, "  --remove    Remove ctx hooks")
			fmt.Fprintln(os.Stderr, "  --status    Show installation status")
			return nil
		}
		if err := install.Install(); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "ctx: hooks installed")
	default:
		return fmt.Errorf("ctx: unknown flag %q for init", flag)
	}
	return nil
}

func cmdHook() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("ctx: usage: ctx hook <precompact|session>")
	}

	switch os.Args[2] {
	case "--help", "-h":
		fmt.Fprintln(os.Stderr, "Usage: ctx hook <precompact|session>")
		fmt.Fprintln(os.Stderr, "  These commands are called by Claude Code hooks, not directly.")
		return nil
	case "precompact":
		return hooks.RunPreCompact()
	case "session":
		return hooks.RunSession()
	default:
		return fmt.Errorf("ctx: unknown hook %q", os.Args[2])
	}
}

func cmdShow() error {
	dir, _ := os.Getwd()

	// Support --project <path> or --project=<path>
	args := os.Args[2:]
	for i, arg := range args {
		if arg == "--project" && i+1 < len(args) {
			dir = args[i+1]
			break
		}
		if strings.HasPrefix(arg, "--project=") {
			dir = strings.TrimPrefix(arg, "--project=")
			break
		}
		if arg == "--help" || arg == "-h" {
			fmt.Fprintln(os.Stderr, "Usage: ctx show [--project <path>]")
			fmt.Fprintln(os.Stderr, "  Print the snapshot for the current or specified directory.")
			return nil
		}
	}

	content, err := snapshot.Read(dir)
	if err != nil {
		return err
	}
	if content == "" {
		fmt.Fprintln(os.Stderr, "ctx: no snapshot for this directory")
		return nil
	}
	fmt.Print(content)
	return nil
}

func cmdClear() error {
	dir, _ := os.Getwd()
	if err := snapshot.Clear(dir); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "ctx: snapshot cleared")
	return nil
}

func cmdList() error {
	infos, legacy, err := snapshot.List()
	if err != nil {
		return err
	}
	if len(infos) == 0 && legacy == 0 {
		fmt.Fprintln(os.Stderr, "ctx: no snapshots found")
		return nil
	}
	for _, info := range infos {
		age := ""
		if !info.CapturedAt.IsZero() {
			d := time.Since(info.CapturedAt).Round(time.Minute)
			age = fmt.Sprintf(" (%s ago)", d)
		}
		fmt.Printf("%s\n  %s%s\n\n", info.ProjectDir, info.Goal, age)
	}
	if legacy > 0 {
		fmt.Fprintf(os.Stderr, "ctx: %d legacy snapshot(s) not shown — trigger a compaction to refresh them\n", legacy)
	}
	return nil
}

func cmdConfig() {
	args := os.Args[2:]

	// Handle --debug true/false
	for i, arg := range args {
		if arg == "--debug" && i+1 < len(args) {
			val := strings.ToLower(args[i+1])
			if val != "true" && val != "false" {
				fmt.Fprintln(os.Stderr, "ctx: --debug value must be true or false")
				os.Exit(1)
			}
			c := config.Load()
			c.Debug = val == "true"
			if err := config.Save(c); err != nil {
				fmt.Fprintf(os.Stderr, "ctx: %v\n", err)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "ctx: debug=%s\n", val)
			return
		}
	}

	// Show current config
	c := config.Load()
	fmt.Printf("data dir:  %s\n", config.DataDir())
	fmt.Printf("log file:  %s\n", config.LogFile())
	fmt.Printf("debug:     %v\n", c.Debug)
}

func cmdLogs() error {
	n := 20
	all := false

	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--all":
			all = true
		case "-n":
			if i+1 >= len(args) {
				return fmt.Errorf("ctx: -n requires a number")
			}
			i++
			count, err := strconv.Atoi(args[i])
			if err != nil || count < 1 {
				return fmt.Errorf("ctx: -n must be a positive integer")
			}
			n = count
		default:
			return fmt.Errorf("ctx: unknown flag %q for logs", args[i])
		}
	}

	logPath := config.LogFile()
	f, err := os.Open(logPath)
	if os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "ctx: no log file yet")
		return nil
	}
	if err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if len(lines) == 0 {
		fmt.Fprintln(os.Stderr, "ctx: log is empty")
		return nil
	}
	total := len(lines)
	start := 0
	if !all && total > n {
		start = total - n
		fmt.Fprintf(os.Stderr, "ctx: showing last %d of %d entries\n", n, total)
	}
	for _, line := range lines[start:] {
		fmt.Println(line)
	}
	return nil
}

func cmdReset() error {
	dir, _ := os.Getwd()
	fmt.Fprint(os.Stderr, "ctx: clear snapshot for [c]urrent directory, [a]ll projects, or [n] cancel? ")
	var answer string
	fmt.Scanln(&answer)
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "c", "current":
		if err := snapshot.Clear(dir); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "ctx: snapshot cleared for current directory")
	case "a", "all":
		if err := snapshot.ClearAll(); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "ctx: all snapshots cleared")
	default:
		fmt.Fprintln(os.Stderr, "ctx: cancelled")
	}
	return nil
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `ctx — preserve Claude Code context across compactions

Usage:
  ctx init              Install hooks in Claude Code
  ctx init --remove     Remove hooks
  ctx init --status     Check hook installation status
  ctx show              Print current snapshot
  ctx show --project P  Print snapshot for project at path P
  ctx clear             Delete current snapshot
  ctx list              List all projects with snapshots
  ctx config                     Show configuration (paths, debug status)
  ctx config --debug true|false  Enable or disable verbose hook logging
  ctx reset             Clear snapshots (current directory or all projects)
  ctx doctor            Check installation health
  ctx logs              Show last 20 hook log entries
  ctx logs -n <count>   Show last N entries
  ctx logs --all        Show all entries
  ctx uninstall         Remove ctx completely (hooks, data, binary)
  ctx update            Update to the latest version
  ctx version           Show version
  ctx hook precompact   (called by Claude Code PreCompact hook)
  ctx hook session      (called by Claude Code SessionStart hook)`)
}
