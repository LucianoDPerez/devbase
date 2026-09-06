package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/devbase/devbase/internal/mcpmerge"
	"github.com/devbase/devbase/internal/pm"
)

// context7Entry is the uniform stdio MCP entry (npx-based, works in every
// JSON-configured IDE). Codex uses TOML and gets a manual step instead.
var context7Entry = map[string]any{
	"command": "npx",
	"args":    []any{"-y", "@upstash/context7-mcp"},
}

func runSetup(args []string) error {
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: devbase setup [--dir PATH] [--ides IDE,...]")
		fmt.Fprintln(os.Stderr, "  Full bootstrap for this machine + project:")
		fmt.Fprintln(os.Stderr, "  install missing dependencies, write rules, wire IDEs, verify.")
		fs.PrintDefaults()
	}
	dir := fs.String("dir", ".", "project directory to set up")
	ides := fs.String("ides", "detected", "which IDEs to configure: detected, all, or comma list")
	skipInstall := fs.Bool("skip-install", false, "skip dependency installation (rules + wiring only)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	info := pm.Detect()
	fmt.Fprintf(os.Stdout, "os: %s, package manager: %s\n", info.OS, info.PM)

	// 1. Dependencies.
	if !*skipInstall {
		for _, d := range pm.Missing() {
			recipe := d.Recipe(info)
			if recipe == nil {
				fmt.Fprintf(os.Stdout, "install %-14s SKIP — manual: %s\n", d.Name, d.Manual)
				continue
			}
		fmt.Fprintf(os.Stdout, "install %-14s $ %s\n", d.Name, strings.Join(recipe, " "))
		// Safe: recipe argv comes only from the hardcoded tables in internal/pm
		// (fixed binaries and flags per OS/package-manager). No user input,
		// project content, or network data ever reaches this call.
		// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
		cmd := exec.Command(recipe[0], recipe[1:]...)
			cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
			if err := cmd.Run(); err != nil {
				fmt.Fprintf(os.Stdout, "install %-14s FAIL — manual: %s\n", d.Name, d.Manual)
			}
		}
	}
	if _, err := exec.LookPath("gh"); err == nil {
		if err := exec.Command("gh", "auth", "status").Run(); err != nil {
			fmt.Fprintln(os.Stdout, "gh found but not authenticated — run: gh auth login")
		}
	}

	// 2. Rules.
	stacks, secs, _, err := writeProject(*dir)
	if err != nil {
		return err
	}
	targets := pickIDEs(*dir, *ides)
	rendered, _, err := renderProject(*dir, secs, targets)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "stacks: %v, rendered %d native rule files\n", stacks, len(rendered))

	// 3. MCP entries (Context7; Engram and codebase-memory self-configure).
	for _, ide := range targets {
		cfg := resolvePathFirst(ide, *dir)
		if cfg == "" {
			fmt.Fprintf(os.Stdout, "mcp %-14s SKIP (no config found)\n", ide.Name)
			continue
		}
		changed, err := mcpmerge.EnsureEntry(cfg, ide.JSONKey, "context7", context7Entry)
		switch {
		case err != nil:
			fmt.Fprintf(os.Stdout, "mcp %-14s SKIP (%v)\n", ide.Name, err)
		case changed:
			fmt.Fprintf(os.Stdout, "mcp %-14s context7 added to %s\n", ide.Name, cfg)
		default:
			fmt.Fprintf(os.Stdout, "mcp %-14s context7 already present\n", ide.Name)
		}
	}

	// 4. External wiring.
	for _, ide := range targets {
		if resolvePathFirst(ide, *dir) == "" {
			continue
		}
		wireIDE(ide)
	}

	fmt.Fprintln(os.Stdout, "done — run `devbase doctor` to verify, then restart your IDE.")
	return nil
}
