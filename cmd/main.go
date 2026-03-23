package main

import (
	"bufio"
	_ "embed"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/AgusRdz/ctx/agents"
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

// parseProjectFlag extracts --project <path> from args, resolving to the git root.
// Returns the resolved dir and remaining args with --project stripped out.
func parseProjectFlag(args []string, defaultDir string) (string, []string) {
	dir := defaultDir
	var remaining []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--project" && i+1 < len(args) {
			dir = agents.GitRoot(args[i+1])
			i++
		} else if strings.HasPrefix(args[i], "--project=") {
			dir = agents.GitRoot(strings.TrimPrefix(args[i], "--project="))
		} else {
			remaining = append(remaining, args[i])
		}
	}
	return dir, remaining
}

// parseSinceDuration parses a duration string like "7d" or "2w" and returns
// the time.Time representing that many days/weeks ago.
func parseSinceDuration(s string) (time.Time, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	var n int
	var unit string
	if _, err := fmt.Sscanf(s, "%d%s", &n, &unit); err != nil || n <= 0 {
		return time.Time{}, fmt.Errorf("ctx: invalid duration %q (use Nd or Nw, e.g. 7d or 2w)", s)
	}
	switch unit {
	case "d":
		return time.Now().Add(-time.Duration(n) * 24 * time.Hour), nil
	case "w":
		return time.Now().Add(-time.Duration(n) * 7 * 24 * time.Hour), nil
	default:
		return time.Time{}, fmt.Errorf("ctx: invalid duration unit %q (use d or w)", unit)
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

func cmdState() error {
	dir, _ := os.Getwd()
	jsonOut := false
	for _, arg := range os.Args[2:] {
		switch arg {
		case "--json":
			jsonOut = true
		case "--help", "-h":
			fmt.Fprintln(os.Stderr, "Usage: ctx state [--json]")
			fmt.Fprintln(os.Stderr, "  Capture and display the current project state without compacting.")
			fmt.Fprintln(os.Stderr, "  --json    Output as JSON instead of markdown")
			return nil
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
	ps := projectstate.Capture(dir, opts)

	if jsonOut {
		out, err := projectstate.FormatJSON(ps)
		if err != nil {
			return err
		}
		fmt.Print(out)
		return nil
	}
	fmt.Print(projectstate.Format(ps, opts.MaxDirtyFiles, opts.MaxErrors))
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
	printField("agents.mode", effective.Agents.Mode, sources.Mode)
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

func cmdAgents() error {
	args := os.Args[2:]
	cwd, _ := os.Getwd()

	if len(args) > 0 {
		switch args[0] {
		case "show":
			rest := args[1:]
			dir, rest := parseProjectFlag(rest, cwd)
			if len(rest) > 0 && rest[0] == "--all" {
				return cmdAgentsShowAll(dir, rest[1:])
			}
			if len(rest) == 0 {
				return fmt.Errorf("ctx: usage: ctx agents show <agent-name>|--all [--project <path>] [--since Nd]")
			}
			return cmdAgentsShow(dir, rest[0])
		case "archive":
			dir, _ := parseProjectFlag(args[1:], cwd)
			return cmdAgentsArchive(agents.GitRoot(dir))
		case "rm":
			return cmdAgentsRm(cwd, args[1:])
		case "summarize":
			return cmdAgentsSummarize(cwd, args[1:])
		case "workspace":
			return cmdAgentsWorkspace(args[1:])
		case "--help", "-h":
			printAgentsHelp()
			return nil
		}
	}

	local := false
	global := false
	mode := ""

	for _, arg := range args {
		switch arg {
		case "--local":
			local = true
		case "--global":
			global = true
		case "--on":
			mode = "on"
		case "--off":
			mode = "off"
		case "--help", "-h":
			printAgentsHelp()
			return nil
		default:
			return fmt.Errorf("ctx: unknown flag %q for agents", arg)
		}
	}

	if mode != "" {
		if err := setConfigField(local, cwd, func(cfg *config.Config) {
			cfg.Agents.Mode = mode
		}, fmt.Sprintf("agents mode set to %s", mode)); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "run ctx init to update hook registration")
		return nil
	}
	if global {
		return showAllAgents()
	}
	return showAgents(agents.GitRoot(cwd))
}

func printAgentsHelp() {
	section := func(s string) string { return tui.BoldErr(tui.CyanErr(s)) }
	flag := func(s string) string { return tui.YellowErr(s) }
	const colW = 44
	row := func(cmd, desc string) string {
		return fmt.Sprintf("  %-*s%s\n", colW, cmd, tui.DimErr(desc))
	}

	var b strings.Builder
	b.WriteString(tui.BoldErr("ctx agents") + " — subagent capture and workspace scanning\n\n")

	b.WriteString(section("Display") + "\n")
	b.WriteString(row("ctx agents", "show mode and captured agents (current project)"))
	b.WriteString(row("ctx agents "+flag("--global"), "show agents across all projects"))
	b.WriteString(row("ctx agents show <name>", "full snapshot for one agent"))
	b.WriteString(row("ctx agents show <name> "+flag("--project")+" <path>", "from a specific project"))
	b.WriteString(row("ctx agents show "+flag("--all"), "all agent snapshots"))
	b.WriteString(row("ctx agents show --all "+flag("--since")+" Nd|Nw", "filter by age (e.g. --since 7d)"))
	b.WriteString("\n")

	b.WriteString(section("Manage") + "\n")
	b.WriteString(row("ctx agents archive ["+flag("--project")+" <path>]", "list archived sessions"))
	b.WriteString(row("ctx agents summarize", "AI summary via claude -p"))
	b.WriteString(row("ctx agents summarize "+flag("--all")+" ["+flag("--since")+" Nd|Nw]", "include archived / filter by age"))
	b.WriteString(row("ctx agents rm <name>", "remove a specific agent snapshot"))
	b.WriteString(row("ctx agents rm "+flag("--before")+" Nd|Nw", "remove snapshots older than N days/weeks"))
	b.WriteString(row("ctx agents rm "+flag("--session")+" <id>", "remove an archived session"))
	b.WriteString(row("ctx agents rm "+flag("--all"), "remove all agent snapshots"))
	b.WriteString("\n")

	b.WriteString(section("Workspace Scanning") + "\n")
	b.WriteString(row("ctx agents workspace list", "show workspaces, exclusions, markers"))
	b.WriteString(row("ctx agents workspace add <path>", "add a workspace directory"))
	b.WriteString(row("ctx agents workspace rm <path>", "remove a workspace directory"))
	b.WriteString(row("ctx agents workspace exclude <path>", "always skip this path during scans"))
	b.WriteString(row("ctx agents workspace unexclude <path>", "remove from exclusion list"))
	b.WriteString(row("ctx agents workspace marker add|rm <pat>", "custom root markers (e.g. *.csproj)"))
	b.WriteString(row("ctx agents workspace boundary add|rm <dir>", "custom boundary dirs (e.g. .terraform)"))
	b.WriteString("\n")

	b.WriteString(section("Mode") + "\n")
	b.WriteString(row("ctx agents "+flag("--on"), "enable agent capture"))
	b.WriteString(row("ctx agents "+flag("--off"), "disable agent capture"))
	b.WriteString(row("ctx agents "+flag("--local")+" "+flag("--on"), "write to local project config"))

	fmt.Fprint(os.Stderr, b.String())
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
		fmt.Printf("mode:    %s  %s\n", colorMode(effective.Agents.Mode), tui.Yellow("[local override]"))
		fmt.Printf("source:  global=%s, local=%s\n", tui.Dim(globalCfg.Agents.Mode), tui.Dim(effective.Agents.Mode))
	} else {
		fmt.Printf("mode:    %s  %s\n", colorMode(effective.Agents.Mode), tui.Dim("["+sources.Mode.String()+"]"))
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
		// Fall back to workspace scan if workspaces are configured
		if len(effective.Agents.Workspaces) > 0 {
			fmt.Fprintln(os.Stderr, tui.Yellow("note:")+" not in a recognized project — scanning workspaces")
			return showWorkspaceAgents(&effective.Agents)
		}
		fmt.Println("no agents captured yet")
		return nil
	}

	fmt.Println(tui.Bold("captured agents") + " (current project):")
	for _, a := range agentList {
		fmt.Printf("  %-22s %s  stopped %s\n", a.Name, tui.Dim(fmt.Sprintf("%-10s", a.Type)), fmtAge(a.StoppedAt))
	}
	return nil
}

func showAllAgents() error {
	projects, err := snapshot.ListAllProjectAgents()
	if err != nil {
		return err
	}

	fmt.Println()
	if len(projects) == 0 {
		fmt.Println("no agents captured yet")
		return nil
	}

	home, _ := os.UserHomeDir()
	fmt.Println(tui.Bold("captured agents") + " (all projects):")
	for _, p := range projects {
		fmt.Printf("\n  %s\n", tui.Cyan(shortenHome(p.ProjectDir, home)))
		for _, a := range p.Agents {
			fmt.Printf("    %-22s %s  stopped %s\n", a.Name, tui.Dim(fmt.Sprintf("%-10s", a.Type)), fmtAge(a.StoppedAt))
		}
	}
	return nil
}

func showWorkspaceAgents(cfg *config.AgentsConfig) error {
	opts := snapshot.ScanOptions{
		MaxDepth:          cfg.Scan.MaxDepth,
		ExtraRootMarkers:  cfg.Scan.ExtraRootMarkers,
		ExtraBoundaryDirs: cfg.Scan.ExtraBoundaryDirs,
		Exclude:           cfg.Scan.Exclude,
	}
	projectDirs, err := snapshot.ScanWorkspaceProjects(cfg.Workspaces, opts)
	if err != nil {
		return err
	}

	home, _ := os.UserHomeDir()
	found := false
	for _, dir := range projectDirs {
		hash := snapshot.ProjectHash(dir)
		agentList, err := snapshot.ListAgents(hash)
		if err != nil || len(agentList) == 0 {
			continue
		}
		if !found {
			fmt.Println(tui.Bold("captured agents") + " (workspace scan):")
			found = true
		}
		fmt.Printf("\n  %s\n", tui.Cyan(shortenHome(dir, home)))
		for _, a := range agentList {
			fmt.Printf("    %-22s %s  stopped %s\n", a.Name, tui.Dim(fmt.Sprintf("%-10s", a.Type)), fmtAge(a.StoppedAt))
		}
	}
	if !found {
		fmt.Println("no agents captured yet")
	}
	return nil
}

// colorMode returns the mode string colored: green for "on", dim for anything else.
func colorMode(mode string) string {
	if mode == "on" {
		return tui.Green(mode)
	}
	return tui.Dim(mode)
}

// fmtAge formats a stopped-at timestamp as a human-readable age string,
// colored yellow if under 1 hour, dim otherwise.
func fmtAge(t time.Time) string {
	if t.IsZero() {
		return tui.Dim("unknown")
	}
	d := time.Since(t).Round(time.Minute)
	s := formatDuration(d) + " ago"
	if d < time.Hour {
		return tui.Yellow(s)
	}
	return tui.Dim(s)
}

// formatDuration formats a duration as a compact human string: "5m", "2h30m", "3h".
func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	switch {
	case h > 0 && m > 0:
		return fmt.Sprintf("%dh%dm", h, m)
	case h > 0:
		return fmt.Sprintf("%dh", h)
	case m <= 1:
		return "just now"
	default:
		return fmt.Sprintf("%dm", m)
	}
}

func cmdAgentsWorkspace(args []string) error {
	if len(args) == 0 || args[0] == "list" {
		return cmdAgentsWorkspaceList()
	}
	if len(args) < 2 {
		return fmt.Errorf("ctx: usage: ctx agents workspace add|rm|exclude|unexclude|marker|boundary <value>")
	}
	switch args[0] {
	case "add":
		return cmdAgentsWorkspaceAdd(args[1])
	case "rm":
		return cmdAgentsWorkspaceRm(args[1])
	case "exclude":
		return cmdAgentsWorkspaceExclude(args[1])
	case "unexclude":
		return cmdAgentsWorkspaceUnexclude(args[1])
	case "marker":
		return cmdAgentsWorkspaceMarker(args[1:])
	case "boundary":
		return cmdAgentsWorkspaceBoundary(args[1:])
	default:
		return fmt.Errorf("ctx: unknown workspace subcommand %q", args[0])
	}
}

func cmdAgentsWorkspaceList() error {
	cfg, err := config.LoadFull(config.GlobalConfigPath())
	if err != nil {
		return err
	}
	fmt.Println()
	home, _ := os.UserHomeDir()

	if len(cfg.Agents.Workspaces) == 0 {
		fmt.Println("no workspaces configured")
		fmt.Println("run: ctx agents workspace add <path>")
	} else {
		fmt.Println("workspaces:")
		for _, ws := range cfg.Agents.Workspaces {
			fmt.Printf("  %s\n", shortenHome(ws, home))
		}
	}

	if cfg.Agents.Scan.MaxDepth > 0 {
		fmt.Printf("\nmax depth: %d\n", cfg.Agents.Scan.MaxDepth)
	}
	if len(cfg.Agents.Scan.Exclude) > 0 {
		fmt.Println("\nexcluded paths:")
		for _, p := range cfg.Agents.Scan.Exclude {
			fmt.Printf("  %s\n", shortenHome(p, home))
		}
	}
	if len(cfg.Agents.Scan.ExtraRootMarkers) > 0 {
		fmt.Printf("\nextra root markers: %s\n", strings.Join(cfg.Agents.Scan.ExtraRootMarkers, ", "))
	}
	if len(cfg.Agents.Scan.ExtraBoundaryDirs) > 0 {
		fmt.Printf("extra boundary dirs: %s\n", strings.Join(cfg.Agents.Scan.ExtraBoundaryDirs, ", "))
	}
	return nil
}

func cmdAgentsWorkspaceAdd(path string) error {
	abs, err := resolveAbsPath(path)
	if err != nil {
		return err
	}
	if _, err := os.Stat(abs); os.IsNotExist(err) {
		return fmt.Errorf("ctx: directory not found: %s", abs)
	}
	home, _ := os.UserHomeDir()
	stored := snapshot.ShortenToHome(abs, home)
	return setConfigField(false, "", func(cfg *config.Config) {
		for _, ws := range cfg.Agents.Workspaces {
			if snapshot.AbsExpandHome(ws) == abs {
				return // already present (compare by expansion, not stored form)
			}
		}
		cfg.Agents.Workspaces = append(cfg.Agents.Workspaces, stored)
	}, fmt.Sprintf("workspace added: %s", stored))
}

func cmdAgentsWorkspaceRm(path string) error {
	abs, err := resolveAbsPath(path)
	if err != nil {
		return err
	}
	return setConfigField(false, "", func(cfg *config.Config) {
		filtered := cfg.Agents.Workspaces[:0]
		for _, ws := range cfg.Agents.Workspaces {
			if snapshot.AbsExpandHome(ws) != abs {
				filtered = append(filtered, ws)
			}
		}
		cfg.Agents.Workspaces = filtered
	}, fmt.Sprintf("workspace removed: %s", abs))
}

func cmdAgentsWorkspaceExclude(path string) error {
	abs, err := resolveAbsPath(path)
	if err != nil {
		return err
	}
	if _, err := os.Stat(abs); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "note: path does not exist (will be excluded if it is created later): %s\n", abs)
	}
	home, _ := os.UserHomeDir()
	stored := snapshot.ShortenToHome(abs, home)
	return setConfigField(false, "", func(cfg *config.Config) {
		for _, p := range cfg.Agents.Scan.Exclude {
			if snapshot.AbsExpandHome(p) == abs {
				return
			}
		}
		cfg.Agents.Scan.Exclude = append(cfg.Agents.Scan.Exclude, stored)
	}, fmt.Sprintf("excluded: %s", stored))
}

func cmdAgentsWorkspaceUnexclude(path string) error {
	abs, err := resolveAbsPath(path)
	if err != nil {
		return err
	}
	return setConfigField(false, "", func(cfg *config.Config) {
		filtered := cfg.Agents.Scan.Exclude[:0]
		for _, p := range cfg.Agents.Scan.Exclude {
			if snapshot.AbsExpandHome(p) != abs {
				filtered = append(filtered, p)
			}
		}
		cfg.Agents.Scan.Exclude = filtered
	}, fmt.Sprintf("unexcluded: %s", abs))
}

func cmdAgentsWorkspaceMarker(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("ctx: usage: ctx agents workspace marker add|rm <pattern>")
	}
	pattern := args[1]
	switch args[0] {
	case "add":
		return setConfigField(false, "", func(cfg *config.Config) {
			for _, m := range cfg.Agents.Scan.ExtraRootMarkers {
				if m == pattern {
					return
				}
			}
			cfg.Agents.Scan.ExtraRootMarkers = append(cfg.Agents.Scan.ExtraRootMarkers, pattern)
		}, fmt.Sprintf("root marker added: %s", pattern))
	case "rm":
		return setConfigField(false, "", func(cfg *config.Config) {
			filtered := cfg.Agents.Scan.ExtraRootMarkers[:0]
			for _, m := range cfg.Agents.Scan.ExtraRootMarkers {
				if m != pattern {
					filtered = append(filtered, m)
				}
			}
			cfg.Agents.Scan.ExtraRootMarkers = filtered
		}, fmt.Sprintf("root marker removed: %s", pattern))
	default:
		return fmt.Errorf("ctx: usage: ctx agents workspace marker add|rm <pattern>")
	}
}

func cmdAgentsWorkspaceBoundary(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("ctx: usage: ctx agents workspace boundary add|rm <dirname>")
	}
	name := args[1]
	switch args[0] {
	case "add":
		return setConfigField(false, "", func(cfg *config.Config) {
			for _, d := range cfg.Agents.Scan.ExtraBoundaryDirs {
				if d == name {
					return
				}
			}
			cfg.Agents.Scan.ExtraBoundaryDirs = append(cfg.Agents.Scan.ExtraBoundaryDirs, name)
		}, fmt.Sprintf("boundary dir added: %s", name))
	case "rm":
		return setConfigField(false, "", func(cfg *config.Config) {
			filtered := cfg.Agents.Scan.ExtraBoundaryDirs[:0]
			for _, d := range cfg.Agents.Scan.ExtraBoundaryDirs {
				if d != name {
					filtered = append(filtered, d)
				}
			}
			cfg.Agents.Scan.ExtraBoundaryDirs = filtered
		}, fmt.Sprintf("boundary dir removed: %s", name))
	default:
		return fmt.Errorf("ctx: usage: ctx agents workspace boundary add|rm <dirname>")
	}
}

// resolveAbsPath expands ~ and returns the absolute path.
func resolveAbsPath(path string) (string, error) {
	abs := snapshot.AbsExpandHome(path)
	if abs == "" {
		return "", fmt.Errorf("ctx: could not resolve path: %s", path)
	}
	return abs, nil
}

// shortenHome converts an absolute path to ~/... when it falls under home.
// Uses forward slashes after ~ so the result is portable across OSes.
func shortenHome(path, home string) string {
	return snapshot.ShortenToHome(path, home)
}

func cmdAgentsShow(projectDir, agentID string) error {
	projectDir = agents.GitRoot(projectDir)
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

func cmdAgentsShowAll(projectDir string, args []string) error {
	var since time.Time
	for i := 0; i < len(args); i++ {
		if args[i] == "--since" && i+1 < len(args) {
			t, err := parseSinceDuration(args[i+1])
			if err != nil {
				return err
			}
			since = t
			i++
		}
	}

	projectDir = agents.GitRoot(projectDir)
	projectHash := snapshot.ProjectHash(projectDir)
	snapshots, err := agents.ReadAllAgentSnapshots(projectHash, since)
	if err != nil {
		return err
	}
	if len(snapshots) == 0 {
		fmt.Fprintln(os.Stderr, "ctx: no agent snapshots found")
		return nil
	}
	for _, s := range snapshots {
		fmt.Printf("# Agent: %s\n", s.Name)
		fmt.Printf("_Stopped: %s_\n", s.StoppedAt.UTC().Format("2006-01-02T15:04Z"))
		fmt.Printf("_Type: %s_\n\n", s.Type)
		fmt.Println(s.FinalOutput)
		fmt.Println("---")
	}
	return nil
}

func cmdAgentsRm(cwd string, args []string) error {
	dir, args := parseProjectFlag(args, cwd)
	dir = agents.GitRoot(dir)
	projectHash := snapshot.ProjectHash(dir)

	if len(args) == 0 {
		return fmt.Errorf("ctx: usage: ctx agents rm <name>|--before Nd|--session <id>|--all [--project <path>]")
	}

	switch args[0] {
	case "--all":
		if err := snapshot.RemoveAllAgents(projectHash); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "ctx: all agent snapshots removed")
	case "--before":
		if len(args) < 2 {
			return fmt.Errorf("ctx: --before requires a duration (e.g. 7d)")
		}
		cutoff, err := parseSinceDuration(args[1])
		if err != nil {
			return err
		}
		n, err := snapshot.RemoveAgentsBefore(projectHash, cutoff)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "ctx: removed %d agent snapshot(s)\n", n)
	case "--session":
		if len(args) < 2 {
			return fmt.Errorf("ctx: --session requires a session ID")
		}
		if err := snapshot.RemoveAgentSession(projectHash, args[1]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "ctx: session %q removed\n", args[1])
	default:
		if err := snapshot.RemoveAgentSnapshot(projectHash, args[0]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "ctx: agent %q removed\n", args[0])
	}
	return nil
}

func cmdAgentsSummarize(cwd string, args []string) error {
	dir, args := parseProjectFlag(args, cwd)
	dir = agents.GitRoot(dir)
	projectHash := snapshot.ProjectHash(dir)

	includeAll := false
	var since time.Time
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--all":
			includeAll = true
		case "--since":
			if i+1 >= len(args) {
				return fmt.Errorf("ctx: --since requires a duration (e.g. 7d)")
			}
			t, err := parseSinceDuration(args[i+1])
			if err != nil {
				return err
			}
			since = t
			i++
		}
	}

	var snapshots []agents.AgentSnapshot
	var err error
	if includeAll {
		snapshots, err = agents.ReadAllAgentSnapshots(projectHash, since)
	} else {
		snapshots, err = agents.ReadAgentSnapshots(projectHash)
		if err == nil && !since.IsZero() {
			var filtered []agents.AgentSnapshot
			for _, s := range snapshots {
				if s.StoppedAt.After(since) {
					filtered = append(filtered, s)
				}
			}
			snapshots = filtered
		}
		if err == nil {
			all, archiveErr := agents.ReadAllAgentSnapshots(projectHash, since)
			if archiveErr == nil && len(all) > len(snapshots) {
				archivedCount := len(all) - len(snapshots)
				fmt.Fprintf(os.Stderr, "ctx: %d archived agent(s) not included — include them? [y/N] ", archivedCount)
				var answer string
				fmt.Scanln(&answer)
				if strings.ToLower(strings.TrimSpace(answer)) == "y" {
					snapshots = all
				}
			}
		}
	}
	if err != nil {
		return err
	}
	if len(snapshots) == 0 {
		fmt.Fprintln(os.Stderr, "ctx: no agent snapshots found")
		return nil
	}

	fmt.Fprintf(os.Stderr, "ctx: summarizing %d agent(s) via claude -p...\n", len(snapshots))
	cfg, _ := config.EffectiveConfig(dir)
	timeout := config.ClaudeTimeout(cfg.Core.ClaudeTimeoutSecs)
	summary, err := agents.GenerateCombinedSummary(snapshots, dir, timeout)
	if err != nil {
		return err
	}
	fmt.Println(summary)
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
	section := func(s string) string { return tui.BoldErr(tui.CyanErr(s)) }
	flag := func(s string) string { return tui.YellowErr(s) }
	const colW = 42
	row := func(cmd, desc string) string {
		return fmt.Sprintf("  %-*s%s\n", colW, cmd, tui.DimErr(desc))
	}

	var b strings.Builder
	b.WriteString(tui.BoldErr("ctx") + " — preserve Claude Code context across compactions\n\n")

	b.WriteString(section("Setup") + "\n")
	b.WriteString(row("ctx init", "install PreCompact and SessionStart hooks"))
	b.WriteString(row("ctx init "+flag("--remove"), "remove ctx hooks"))
	b.WriteString(row("ctx init "+flag("--status"), "check hook installation status"))
	b.WriteString(row("ctx init "+flag("--local"), "create local project config (.ctx/config.yml)"))
	b.WriteString(row("ctx init --local "+flag("--agents")+" on|off", "create local config with agents preset"))
	b.WriteString("\n")

	b.WriteString(section("Session") + "\n")
	b.WriteString(row("ctx show", "print current snapshot"))
	b.WriteString(row("ctx show "+flag("--project")+" <path>", "print snapshot for a specific project"))
	b.WriteString(row("ctx clear", "delete current snapshot"))
	b.WriteString(row("ctx clear "+flag("--agents-only"), "clear only agent snapshots"))
	b.WriteString(row("ctx list", "list all projects with snapshots"))
	b.WriteString("\n")

	b.WriteString(section("Agents") + "\n")
	b.WriteString(row("ctx agents", "show mode and captured agents"))
	b.WriteString(row("ctx agents "+flag("--on")+" | "+flag("--off"), "enable or disable capture"))
	b.WriteString(row("ctx agents "+flag("--local")+" "+flag("--on"), "set mode in local project config"))
	b.WriteString(row("ctx agents "+flag("--global"), "show agents across all projects"))
	b.WriteString(row("ctx agents show <name>", "print full agent snapshot"))
	b.WriteString(row("ctx agents show "+flag("--all")+" ["+flag("--since")+" Nd]", "print all snapshots"))
	b.WriteString(row("ctx agents archive", "list archived sessions"))
	b.WriteString(row("ctx agents rm <name|"+flag("--all")+"|"+flag("--before")+">", "remove agent snapshots"))
	b.WriteString(row("ctx agents summarize ["+flag("--all")+"]", "AI summary via claude -p"))
	b.WriteString(row("ctx agents workspace add|rm <path>", "manage workspace directories"))
	b.WriteString(row("ctx agents workspace list", "show workspaces and scan config"))
	b.WriteString(row("ctx agents "+flag("--help"), "full agents command reference"))
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
	b.WriteString(row("ctx reset", "clear snapshots interactively"))
	b.WriteString(row("ctx uninstall", "remove ctx completely"))
	b.WriteString(row("ctx version", "show version"))
	b.WriteString(row("ctx changelog ["+flag("--full")+"]", "show release notes"))

	fmt.Fprint(os.Stderr, b.String())
}
