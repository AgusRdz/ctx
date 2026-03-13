package main

import (
	"bufio"
	_ "embed"
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
	case "agents":
		err = cmdAgents()
	case "changelog", "--changelog":
		runChangelog(os.Args[2:])
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
	agentsMode := ""
	action := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--local":
			local = true
		case "--agents":
			if i+1 >= len(args) {
				return fmt.Errorf("ctx: --agents requires a mode: v1|v2|off")
			}
			i++
			agentsMode = args[i]
			if agentsMode != "on" && agentsMode != "off" {
				return fmt.Errorf("ctx: --agents mode must be on or off")
			}
		case "--remove", "--status":
			if action != "" {
				return fmt.Errorf("ctx: conflicting flags")
			}
			action = args[i]
		case "--help", "-h":
			fmt.Fprintln(os.Stderr, "Usage: ctx init [--remove|--status|--local [--agents v1|v2|off]]")
			fmt.Fprintln(os.Stderr, "  (no flag)            Install PreCompact and SessionStart hooks")
			fmt.Fprintln(os.Stderr, "  --remove             Remove ctx hooks")
			fmt.Fprintln(os.Stderr, "  --status             Show installation status")
			fmt.Fprintln(os.Stderr, "  --local              Create .ctx/config.yml in current directory")
			fmt.Fprintln(os.Stderr, "  --local --agents v1  Create local config with agents mode preset")
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
			return cmdInitLocal(agentsMode)
		}
		if err := install.Install(); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "ctx: hooks installed")
	}
	return nil
}

// cmdInitLocal creates .ctx/config.yml in the current directory.
func cmdInitLocal(agentsMode string) error {
	dir, _ := os.Getwd()
	localPath := config.ProjectConfigPath(dir)
	localDir := config.ProjectConfigDir(dir)

	if err := os.MkdirAll(localDir, 0o755); err != nil {
		return fmt.Errorf("ctx: %w", err)
	}

	var content string
	if agentsMode != "" {
		content = fmt.Sprintf(
			"# Local ctx config — overrides ~/.config/ctx/config.yml\n"+
				"# Only include fields you want to override.\n"+
				"# This file should NOT be committed. Add .ctx/ to .gitignore.\n\n"+
				"agents:\n  mode: %s\n", agentsMode)
	} else {
		content = "# Local ctx config — overrides ~/.config/ctx/config.yml\n" +
			"# Only include fields you want to override.\n" +
			"# This file should NOT be committed. Add .ctx/ to .gitignore.\n\n" +
			"# agents:\n#   mode: v1\n"
	}

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
	path := projectDir + "/.gitignore"
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
		return fmt.Errorf("ctx: usage: ctx hook <precompact|session|subagent>")
	}

	switch os.Args[2] {
	case "--help", "-h":
		fmt.Fprintln(os.Stderr, "Usage: ctx hook <precompact|session|subagent>")
		fmt.Fprintln(os.Stderr, "  These commands are called by Claude Code hooks, not directly.")
		return nil
	case "precompact":
		return hooks.RunPreCompact()
	case "session":
		return hooks.RunSession()
	case "subagent":
		return hooks.RunSubagentStop()
	default:
		return fmt.Errorf("ctx: unknown hook %q", os.Args[2])
	}
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
	agentsOnly := false

	for _, arg := range os.Args[2:] {
		switch arg {
		case "--agents-only":
			agentsOnly = true
		case "--help", "-h":
			fmt.Fprintln(os.Stderr, "Usage: ctx clear [--agents-only]")
			return nil
		default:
			return fmt.Errorf("ctx: unknown flag %q for clear", arg)
		}
	}

	if agentsOnly {
		if err := snapshot.ClearAgents(dir); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "ctx: agent snapshots cleared")
		return nil
	}
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
	printField("agents.mode", effective.Agents.Mode, sources.Mode)
	printField("agents.inject_on_start", effective.Agents.InjectOnStart, sources.InjectOnStart)
	printField("agents.max_inject", effective.Agents.MaxInject, sources.MaxInject)
	printField("agents.staleness_days", effective.Agents.StalenessDays, sources.StalenessDays)

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

func cmdAgents() error {
	args := os.Args[2:]
	dir, _ := os.Getwd()

	local := false
	mode := ""

	// subcommands: ctx agents show <name>, ctx agents archive, ctx agents inject <name> [dir]
	if len(args) > 0 {
		switch args[0] {
		case "show":
			if len(args) < 2 {
				return fmt.Errorf("ctx: usage: ctx agents show <agent-name>")
			}
			return cmdAgentsShow(dir, args[1])
		case "archive":
			return cmdAgentsArchive(dir)
		case "inject":
			if len(args) < 2 {
				return fmt.Errorf("ctx: usage: ctx agents inject <agent-name> [dir]")
			}
			targetDir := dir
			if len(args) >= 3 {
				targetDir = args[2]
			}
			return cmdAgentsInject(dir, targetDir, args[1])
		}
	}

	for _, arg := range args {
		switch arg {
		case "--local":
			local = true
		case "--on":
			mode = "on"
		case "--off":
			mode = "off"
		case "--help", "-h":
			fmt.Fprintln(os.Stderr, "Usage: ctx agents [--on|--off] [--local]")
			fmt.Fprintln(os.Stderr, "  (no flags)        Show active mode and list captured agents")
			fmt.Fprintln(os.Stderr, "  show <name>       Print full snapshot for a captured agent")
			fmt.Fprintln(os.Stderr, "  archive           List archived agent sessions")
			fmt.Fprintln(os.Stderr, "  inject <name> [dir]  Inject agent context as session snapshot for dir")
			fmt.Fprintln(os.Stderr, "  --on              Enable agent capture")
			fmt.Fprintln(os.Stderr, "  --off             Disable agent capture")
			fmt.Fprintln(os.Stderr, "  --local           Write to local project config instead of global")
			return nil
		default:
			return fmt.Errorf("ctx: unknown flag %q for agents", arg)
		}
	}

	if mode != "" {
		if err := setConfigField(local, dir, func(cfg *config.Config) {
			cfg.Agents.Mode = mode
		}, fmt.Sprintf("agents mode set to %s", mode)); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "run ctx init to update hook registration")
		return nil
	}
	return showAgents(dir)
}

func cmdAgentsArchive(projectDir string) error {
	projectHash := snapshot.ProjectHash(projectDir)
	groups, err := snapshot.ListArchivedAgentGroups(projectHash)
	if err != nil {
		return err
	}
	if len(groups) == 0 {
		fmt.Fprintln(os.Stderr, "ctx: no archived agents")
		return nil
	}
	fmt.Println()
	fmt.Println("archived agent sessions (current project):")
	for _, g := range groups {
		label := g.DirName
		if !g.Timestamp.IsZero() {
			label = g.Timestamp.UTC().Format("2006-01-02T15:04Z")
		}
		plural := "agents"
		if len(g.Agents) == 1 {
			plural = "agent"
		}
		fmt.Printf("\n  %s  (%d %s)\n", label, len(g.Agents), plural)
		for _, a := range g.Agents {
			fmt.Printf("    %s\n", a.Name)
		}
	}
	return nil
}

func showAgents(projectDir string) error {
	effective, sources, err := config.EffectiveConfigWithSources(projectDir)
	if err != nil {
		return err
	}

	fmt.Println()
	if sources.Mode == config.SourceLocal {
		globalCfg, _ := config.EffectiveConfig("")
		fmt.Printf("mode:    %s  [local override]\n", effective.Agents.Mode)
		fmt.Printf("source:  global=%s, local=%s\n", globalCfg.Agents.Mode, effective.Agents.Mode)
	} else {
		fmt.Printf("mode:    %s  [%s]\n", effective.Agents.Mode, sources.Mode)
	}

	if effective.Agents.Mode == "off" || effective.Agents.Mode == "" {
		fmt.Println()
		fmt.Println("no agents captured yet — run ctx agents --on to enable")
		return nil
	}

	// List captured agents for current project
	projectHash := snapshot.ProjectHash(projectDir)
	agentList, err := snapshot.ListAgents(projectHash)
	if err != nil {
		return err
	}

	fmt.Println()
	if len(agentList) == 0 {
		fmt.Println("no agents captured yet")
		return nil
	}

	fmt.Println("captured agents (current project):")
	for _, a := range agentList {
		age := "unknown"
		if !a.StoppedAt.IsZero() {
			d := time.Since(a.StoppedAt).Round(time.Minute)
			age = fmt.Sprintf("%s ago", d)
		}
		fmt.Printf("  %-22s %-10s stopped %s\n", a.Name, a.Type, age)
	}
	return nil
}

func cmdAgentsShow(projectDir, agentID string) error {
	projectHash := snapshot.ProjectHash(projectDir)
	content, err := snapshot.ReadAgent(projectHash, agentID)
	if err != nil {
		return err
	}
	if content == "" {
		fmt.Fprintf(os.Stderr, "ctx: no snapshot found for agent %q\n", agentID)
		return nil
	}
	fmt.Print(content)
	return nil
}

func cmdAgentsInject(sourceProjectDir, targetDir, agentName string) error {
	projectHash := snapshot.ProjectHash(sourceProjectDir)
	content, err := snapshot.ReadAgent(projectHash, agentName)
	if err != nil {
		return err
	}
	if content == "" {
		fmt.Fprintf(os.Stderr, "ctx: agent %q not found\n", agentName)
		return nil
	}
	if err := snapshot.Write(targetDir, content); err != nil {
		return fmt.Errorf("ctx: %w", err)
	}
	fmt.Fprintf(os.Stderr, "ctx: injected agent %q context into %s\n", agentName, targetDir)
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
  ctx init --local      Create local project config (.ctx/config.yml)
  ctx show              Print current snapshot
  ctx show --project P  Print snapshot for project at path P
  ctx clear             Delete current snapshot
  ctx clear --agents-only  Clear only agent snapshots
  ctx list              List all projects with snapshots
  ctx config                     Show effective configuration with sources
  ctx config --global            Show only global config
  ctx config --local             Show only local config
  ctx config --debug true|false  Enable or disable verbose hook logging
  ctx agents                     Show agents mode and captured agents
  ctx agents show <name>         Print full snapshot for a captured agent
  ctx agents inject <name> [dir] Inject agent context as session snapshot for dir
  ctx agents archive             List archived agent sessions
  ctx agents --on                Enable agent capture
  ctx agents --off               Disable agent capture
  ctx agents --local --on        Set mode in local project config
  ctx reset             Clear snapshots (current directory or all projects)
  ctx doctor            Check installation health
  ctx logs              Show last 20 hook log entries
  ctx logs -n <count>   Show last N entries
  ctx logs --all        Show all entries
  ctx changelog              Show changes in the current version
  ctx changelog --full       Show full changelog history
  ctx uninstall         Remove ctx completely (hooks, data, binary)
  ctx update            Update to the latest version
  ctx version           Show version`)
}
