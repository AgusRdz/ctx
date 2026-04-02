package main

import (
	"bufio"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/AgusRdz/ctx/config"
	"github.com/AgusRdz/ctx/hooks"
	"github.com/AgusRdz/ctx/install"
	"github.com/AgusRdz/ctx/projectstate"
	"github.com/AgusRdz/ctx/snapshot"
	"github.com/AgusRdz/ctx/tui"
	"github.com/AgusRdz/ctx/updater"
)

//go:embed CHANGELOG.md
var changelog string

// version is set at build time via -ldflags "-X main.version=..."
var version = "dev"

func main() {
	// Apply any pending auto-update from a previous run (silent, non-disruptive)
	updater.ApplyPendingUpdate(version)

	// Silently initialize global config on first run
	if _, err := os.Stat(config.GlobalConfigPath()); os.IsNotExist(err) {
		_ = config.Save(config.GlobalConfigPath(), config.DefaultConfig())
	}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Internal flags used by the auto-update subprocess
	if os.Args[1] == "--_bg-update" {
		if len(os.Args) >= 3 {
			updater.RunBackgroundUpdate(os.Args[2])
		}
		return
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
		err = cmdConfig()
	case "state":
		err = cmdState()
	case "changelog", "--changelog":
		runChangelog(os.Args[2:])
	case "doctor":
		install.Doctor()
	case "logs":
		err = cmdLogs()
	case "uninstall":
		if install.ConfirmUninstall(os.Args[2:]) {
			err = install.Uninstall()
		}
	case "update":
		updater.Run(version)
	case "version", "--version", "-v":
		fmt.Printf("ctx %s\n", version)
	case "--color-debug":
		tui.ColorDebug()
		return
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

	// Check for updates in the background (every 24h, downloads silently)
	updater.BackgroundCheck(version)
}

// runChangelog prints the changelog. Default: latest version only. --full: entire history.
func runChangelog(args []string) {
	if changelog == "" {
		fmt.Println("no changelog available")
		return
	}
	if len(args) > 0 && (args[0] == "--full" || args[0] == "-f") {
		fmt.Print(changelog)
		return
	}
	fmt.Print(extractLatestVersion(changelog))
}

// extractLatestVersion extracts the first version section from the changelog, skipping [Unreleased].
func extractLatestVersion(cl string) string {
	lines := strings.Split(cl, "\n")
	var result []string
	inSection := false

	for _, line := range lines {
		if strings.HasPrefix(line, "## [") {
			if inSection {
				break // hit the next version, stop
			}
			if strings.HasPrefix(line, "## [Unreleased]") {
				continue
			}
			inSection = true
		}
		if inSection {
			result = append(result, line)
		}
	}

	if len(result) == 0 {
		return cl
	}
	return strings.Join(result, "\n") + "\n"
}

func cmdInit() error {
	args := os.Args[2:]

	local := false
	action := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--local":
			local = true
		case "--remove", "--status":
			if action != "" {
				return fmt.Errorf("ctx: conflicting flags")
			}
			action = args[i]
		case "--help", "-h":
			fmt.Fprintln(os.Stderr, "Usage: ctx init [--remove|--status|--local]")
			fmt.Fprintln(os.Stderr, "  (no flag)  Install PreCompact and SessionStart hooks")
			fmt.Fprintln(os.Stderr, "  --remove   Remove ctx hooks")
			fmt.Fprintln(os.Stderr, "  --status   Show installation status")
			fmt.Fprintln(os.Stderr, "  --local    Create .ctx/config.yml in current directory")
			return nil
		default:
			return fmt.Errorf("ctx: unknown flag %q for init", args[i])
		}
	}

	switch action {
	case "--remove":
		if err := install.Remove(); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "ctx: hooks removed")
	case "--status":
		fmt.Println(install.Status())
	default:
		if local {
			return cmdInitLocal()
		}
		if err := install.Install(); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "ctx: hooks installed")
	}
	return nil
}

// cmdInitLocal creates .ctx/config.yml in the current directory.
func cmdInitLocal() error {
	dir, _ := os.Getwd()
	localPath := config.ProjectConfigPath(dir)
	localDir := config.ProjectConfigDir(dir)

	if err := os.MkdirAll(localDir, 0o755); err != nil {
		return fmt.Errorf("ctx: %w", err)
	}

	content := "# Local ctx config — overrides ~/.config/ctx/config.yml\n" +
		"# Only include fields you want to override.\n" +
		"# This file should NOT be committed. Add .ctx/ to .gitignore.\n"

	if err := os.WriteFile(localPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	fmt.Fprintf(os.Stderr, "ctx: created %s\n", localPath)
	addToGitignore(dir)
	return nil
}

// addToGitignore appends .ctx/ to the project's .gitignore if not already present.
func addToGitignore(projectDir string) {
	entry := ".ctx/"
	path := filepath.Join(projectDir, ".gitignore")
	data, err := os.ReadFile(path)
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.TrimSpace(line) == entry {
				return
			}
		}
		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return
		}
		defer f.Close()
		if len(data) > 0 && data[len(data)-1] != '\n' {
			f.WriteString("\n")
		}
		f.WriteString(entry + "\n")
		fmt.Fprintf(os.Stderr, "ctx: added .ctx/ to .gitignore\n")
	} else if os.IsNotExist(err) {
		os.WriteFile(path, []byte(entry+"\n"), 0o644)
		fmt.Fprintf(os.Stderr, "ctx: created .gitignore with .ctx/\n")
	}
}

func cmdHook() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("ctx: usage: ctx hook <precompact|session|postcompact>")
	}

	switch os.Args[2] {
	case "--help", "-h":
		fmt.Fprintln(os.Stderr, "Usage: ctx hook <precompact|session|postcompact>")
		fmt.Fprintln(os.Stderr, "  These commands are called by Claude Code hooks, not directly.")
		return nil
	case "precompact":
		return hooks.RunPreCompact()
	case "session":
		return hooks.RunSession()
	case "postcompact":
		return hooks.RunPostCompact()
	default:
		return fmt.Errorf("ctx: unknown hook %q", os.Args[2])
	}
}

func cmdState() error {
	dir, _ := os.Getwd()
	for _, arg := range os.Args[2:] {
		switch arg {
		case "--help", "-h":
			fmt.Fprintln(os.Stderr, "Usage: ctx state")
			fmt.Fprintln(os.Stderr, "  Capture and display the current project state without compacting.")
			return nil
		default:
			return fmt.Errorf("ctx: unknown flag %q for state", arg)
		}
	}

	cfg, err := config.EffectiveConfig(dir)
	if err != nil {
		return err
	}

	opts := projectstate.CaptureOptions{
		Git:                 cfg.ProjectState.Git,
		MaxDirtyFiles:       cfg.ProjectState.MaxDirtyFiles,
		MaxErrors:           cfg.ProjectState.MaxErrors,
		TypeCheck:           cfg.ProjectState.TypeCheck.Enabled,
		TypeCheckTimeout:    config.ClaudeTimeout(cfg.ProjectState.TypeCheck.TimeoutSeconds),
		TypeCheckCommand:    cfg.ProjectState.TypeCheck.Command,
		Tests:               cfg.ProjectState.Tests.Enabled,
		TestsTimeout:        config.ClaudeTimeout(cfg.ProjectState.Tests.TimeoutSeconds),
		TestsMaxFailedNames: cfg.ProjectState.Tests.MaxFailedNames,
		TestsCommand:        cfg.ProjectState.Tests.Command,
	}
	fmt.Print(projectstate.Format(projectstate.Capture(dir, opts), opts.MaxDirtyFiles, opts.MaxErrors))
	return nil
}

func cmdShow() error {
	dir, _ := os.Getwd()

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
			return nil
		}
	}

	branch := snapshot.BranchForProject(dir)
	content, err := snapshot.Read(dir, branch)
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
	allBranches := false

	for _, arg := range os.Args[2:] {
		switch arg {
		case "--all":
			allBranches = true
		case "--help", "-h":
			fmt.Fprintln(os.Stderr, "Usage: ctx clear [--all]")
			fmt.Fprintln(os.Stderr, "  (no flag)  Clear snapshot for current branch only")
			fmt.Fprintln(os.Stderr, "  --all      Clear all branch snapshots for this project")
			return nil
		default:
			return fmt.Errorf("ctx: unknown flag %q for clear", arg)
		}
	}

	if allBranches {
		if err := snapshot.ClearAll(dir); err != nil {
			return err
		}
	} else {
		branch := snapshot.BranchForProject(dir)
		if err := snapshot.Clear(dir, branch); err != nil {
			return err
		}
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
		branchLabel := ""
		if info.Branch != "" && info.Branch != "_" {
			branchLabel = fmt.Sprintf(" [%s]", info.Branch)
		}
		fmt.Printf("%s%s\n  %s%s\n\n", info.ProjectDir, branchLabel, info.Goal, age)
	}
	if legacy > 0 {
		fmt.Fprintf(os.Stderr, "ctx: %d legacy snapshot(s) not shown — trigger a compaction to refresh them\n", legacy)
	}
	return nil
}

func cmdConfig() error {
	args := os.Args[2:]
	dir, _ := os.Getwd()

	local := false
	showGlobal := false
	showLocal := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--local":
			local = true
			showLocal = true
		case "--global":
			showGlobal = true
		case "--debug":
			if i+1 >= len(args) {
				return fmt.Errorf("ctx: --debug requires true or false")
			}
			i++
			val := strings.ToLower(args[i])
			if val != "true" && val != "false" {
				return fmt.Errorf("ctx: --debug value must be true or false")
			}
			return setConfigField(local, dir, func(cfg *config.Config) {
				cfg.Core.Debug = val == "true"
			}, fmt.Sprintf("debug=%s", val))
		case "--help", "-h":
			fmt.Fprintln(os.Stderr, "Usage: ctx config [--global|--local] [--debug true|false]")
			return nil
		default:
			return fmt.Errorf("ctx: unknown flag %q for config", args[i])
		}
	}

	if showGlobal {
		return showRawConfigFile(config.GlobalConfigPath(), "global")
	}
	if showLocal {
		localPath := config.ProjectConfigPath(dir)
		if _, err := os.Stat(localPath); os.IsNotExist(err) {
			return fmt.Errorf("ctx: no local config at %s", localPath)
		}
		return showRawConfigFile(localPath, "local")
	}

	return showEffectiveConfig(dir)
}

// setConfigField loads a config file, applies a mutation, and saves it back.
func setConfigField(local bool, projectDir string, mutate func(*config.Config), label string) error {
	var path string
	if local {
		path = config.ProjectConfigPath(projectDir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("ctx: no local config — run 'ctx init --local' first")
		}
	} else {
		path = config.GlobalConfigPath()
	}

	cfg, err := config.LoadFull(path)
	if err != nil {
		return err
	}
	mutate(cfg)
	if err := config.Save(path, cfg); err != nil {
		return err
	}

	scope := "global"
	if local {
		scope = "local"
	}
	fmt.Fprintf(os.Stderr, "ctx: %s [%s]\n", label, scope)
	return nil
}

func showRawConfigFile(path, label string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "ctx: no %s config at %s\n", label, path)
		return nil
	}
	if err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	fmt.Printf("# %s: %s\n", label, path)
	fmt.Print(string(data))
	return nil
}

func showEffectiveConfig(projectDir string) error {
	effective, sources, err := config.EffectiveConfigWithSources(projectDir)
	if err != nil {
		return err
	}

	localPath := config.ProjectConfigPath(projectDir)
	hasLocal := false
	if _, err := os.Stat(localPath); err == nil {
		hasLocal = true
	}

	fmt.Println()
	fmt.Println("effective configuration")
	fmt.Println("───────────────────────────────────────")

	printField := func(name string, value interface{}, src config.FieldSource) {
		marker := ""
		if src == config.SourceLocal {
			marker = "  ← override"
		}
		fmt.Printf("%-24s %-10v [%s]%s\n", name, value, src, marker)
	}

	printField("core.debug", effective.Core.Debug, sources.Debug)
	printField("project_state.enabled", effective.ProjectState.Enabled, sources.ProjectStateEnabled)
	printField("project_state.git", effective.ProjectState.Git, sources.ProjectStateGit)
	printField("project_state.max_dirty_files", effective.ProjectState.MaxDirtyFiles, sources.ProjectStateMaxDirty)
	printField("project_state.max_errors", effective.ProjectState.MaxErrors, sources.ProjectStateMaxErrors)
	printField("project_state.typecheck.enabled", effective.ProjectState.TypeCheck.Enabled, sources.TypeCheckEnabled)
	printField("project_state.typecheck.timeout_seconds", effective.ProjectState.TypeCheck.TimeoutSeconds, sources.TypeCheckTimeout)
	tcCmd := effective.ProjectState.TypeCheck.Command
	if tcCmd == "" {
		tcCmd = "(auto-detect)"
	}
	printField("project_state.typecheck.command", tcCmd, sources.TypeCheckCommand)
	printField("project_state.tests.enabled", effective.ProjectState.Tests.Enabled, sources.TestsEnabled)
	printField("project_state.tests.timeout_seconds", effective.ProjectState.Tests.TimeoutSeconds, sources.TestsTimeout)
	printField("project_state.tests.max_failed_names", effective.ProjectState.Tests.MaxFailedNames, sources.TestsMaxFailedNames)
	testsCmd := effective.ProjectState.Tests.Command
	if testsCmd == "" {
		testsCmd = "(auto-detect)"
	}
	printField("project_state.tests.command", testsCmd, sources.TestsCommand)

	fmt.Println()
	fmt.Printf("global:  %s\n", config.GlobalConfigPath())
	if hasLocal {
		fmt.Printf("local:   %s\n", localPath)
	} else {
		fmt.Println("local:   none")
	}
	_ = hasLocal
	return nil
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

func printUsage() {
	section := func(s string) string { return tui.BoldErr(tui.CyanErr(s)) }
	flag := func(s string) string { return tui.YellowErr(s) }
	const colW = 42
	row := func(cmd, desc string) string {
		return fmt.Sprintf("  %-*s%s\n", colW, cmd, tui.DimErr(desc))
	}

	var b strings.Builder
	b.WriteString(tui.BoldErr("ctx") + " — preserve Claude Code context across compactions\n\n")

	b.WriteString(section("Setup") + "\n")
	b.WriteString(row("ctx init", "install PreCompact, PostCompact, and SessionStart hooks"))
	b.WriteString(row("ctx init "+flag("--remove"), "remove ctx hooks"))
	b.WriteString(row("ctx init "+flag("--status"), "check hook installation status"))
	b.WriteString(row("ctx init "+flag("--local"), "create local project config (.ctx/config.yml)"))
	b.WriteString("\n")

	b.WriteString(section("Session") + "\n")
	b.WriteString(row("ctx show", "print current snapshot"))
	b.WriteString(row("ctx show "+flag("--project")+" <path>", "print snapshot for a specific project"))
	b.WriteString(row("ctx state", "capture and print current project state"))
	b.WriteString(row("ctx clear", "delete snapshot for current branch"))
	b.WriteString(row("ctx clear "+flag("--all"), "delete all branch snapshots for this project"))
	b.WriteString(row("ctx list", "list all projects with snapshots"))
	b.WriteString("\n")

	b.WriteString(section("Configuration") + "\n")
	b.WriteString(row("ctx config", "show effective configuration with sources"))
	b.WriteString(row("ctx config "+flag("--global")+" | "+flag("--local"), "show a specific config file"))
	b.WriteString(row("ctx config "+flag("--debug")+" true|false", "toggle verbose hook logging"))
	b.WriteString("\n")

	b.WriteString(section("Diagnostics") + "\n")
	b.WriteString(row("ctx doctor", "check installation health"))
	b.WriteString(row("ctx logs", "show last 20 hook log entries"))
	b.WriteString(row("ctx logs "+flag("-n")+" N | "+flag("--all"), "show N or all entries"))
	b.WriteString("\n")

	b.WriteString(section("Maintenance") + "\n")
	b.WriteString(row("ctx update", "update to the latest version"))
	b.WriteString(row("ctx uninstall", "remove ctx completely"))
	b.WriteString(row("ctx version", "show version"))
	b.WriteString(row("ctx changelog ["+flag("--full")+"]", "show release notes"))

	fmt.Fprint(os.Stderr, b.String())
}
