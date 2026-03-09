package main

import (
	"fmt"
	"os"

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
		fmt.Fprintf(os.Stderr, "ctx: %v\n", err)
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
	case "":
		if err := install.Install(); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "ctx: hooks installed")
	default:
		return fmt.Errorf("unknown flag %q for init", flag)
	}
	return nil
}

func cmdHook() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: ctx hook <precompact|session>")
	}

	switch os.Args[2] {
	case "precompact":
		return hooks.RunPreCompact()
	case "session":
		return hooks.RunSession()
	default:
		return fmt.Errorf("unknown hook %q", os.Args[2])
	}
}

func cmdShow() error {
	dir, _ := os.Getwd()
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

func printUsage() {
	fmt.Fprintln(os.Stderr, `ctx — preserve Claude Code context across compactions

Usage:
  ctx init              Install hooks in Claude Code
  ctx init --remove     Remove hooks
  ctx init --status     Check hook installation status
  ctx show              Print current snapshot
  ctx clear             Delete current snapshot
  ctx uninstall         Remove ctx completely
  ctx update            Update to the latest version
  ctx version           Show version
  ctx hook precompact   (called by Claude Code PreCompact hook)
  ctx hook session      (called by Claude Code SessionStart hook)`)
}
