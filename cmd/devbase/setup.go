package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/devbase/devbase/internal/detect"
	"github.com/devbase/devbase/internal/mcpmerge"
	"github.com/devbase/devbase/internal/pm"
	"github.com/devbase/devbase/internal/ui"
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
	out := os.Stdout

	info := pm.Detect()
	fmt.Fprintln(out, ui.Section("Environment"))
	fmt.Fprintf(out, "  %s · package manager: %s\n\n", info.OS, info.PM)

	// 1. Dependencies.
	fmt.Fprintln(out, ui.Section("Dependencies"))
	if !*skipInstall {
		for _, d := range pm.Missing() {
			recipe := d.Recipe(info)
			if recipe == nil {
				fmt.Fprintln(out, ui.Warn(d.Name, "manual: "+d.Manual))
				continue
			}
			fmt.Fprintf(out, "  %s $ %s\n", d.Name, ui.Dim(strings.Join(recipe, " ")))
			// Safe: recipe argv comes only from the hardcoded tables in internal/pm
			// (fixed binaries and flags per OS/package-manager). No user input,
			// project content, or network data ever reaches this call.
			// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
			cmd := exec.Command(recipe[0], recipe[1:]...)
			cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
			if err := cmd.Run(); err != nil {
				fmt.Fprintln(out, ui.Fail(d.Name, "manual: "+d.Manual))
			} else {
				fmt.Fprintln(out, ui.Ok(d.Name, "installed"))
			}
		}
	}
	for _, d := range pm.Deps() {
		if isInstalled(d.Bin) {
			fmt.Fprintln(out, ui.Ok(d.Name, "ready"))
		}
	}
	if _, err := exec.LookPath("gh"); err == nil {
		if err := exec.Command("gh", "auth", "status").Run(); err != nil {
			fmt.Fprintln(out, ui.Warn("gh auth", "run: gh auth login"))
		}
	}
	fmt.Fprintln(out)

	// 1b. Playwright is conditional: suggest it for browser UIs, never install
	// browsers uninvited (hundreds of MB + system deps).
	if detect.WebFrontend(*dir) {
		fmt.Fprintln(out, ui.Section("E2E"))
		fmt.Fprintln(out, "  Web frontend detected — Playwright applies to this project.")
		fmt.Fprintln(out, ui.Dim("  gate runs specs automatically once a playwright.config exists."))
		fmt.Fprintln(out, ui.Dim("  to add it: npm init playwright@latest && npx playwright install --with-deps"))
		fmt.Fprintln(out)
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
	fmt.Fprintln(out, ui.Section("Rules"))
	fmt.Fprintf(out, "  stacks: %s\n", strings.Join(stacks, ", "))
	fmt.Fprintf(out, "  %d native rule files rendered\n\n", len(rendered))

	// 3. MCP entries (Context7; Engram and codebase-memory self-configure).
	fmt.Fprintln(out, ui.Section("MCP"))
	for _, ide := range targets {
		cfg := resolvePathFirst(ide, *dir)
		if cfg == "" {
			fmt.Fprintln(out, ui.Warn(ide.Name, "no config found"))
			continue
		}
		changed, err := mcpmerge.EnsureEntry(cfg, ide.JSONKey, "context7", context7Entry)
		switch {
		case err != nil:
			fmt.Fprintln(out, ui.Warn(ide.Name, err.Error()))
		case changed:
			fmt.Fprintln(out, ui.Ok(ide.Name, "context7 added"))
		default:
			fmt.Fprintln(out, ui.Ok(ide.Name, "context7 present"))
		}
	}
	fmt.Fprintln(out)

	// 4. External wiring.
	fmt.Fprintln(out, ui.Section("Wiring"))
	for _, ide := range targets {
		if resolvePathFirst(ide, *dir) == "" {
			continue
		}
		wireIDE(ide)
	}
	fmt.Fprintln(out)

	fmt.Fprintln(out, ui.Dim("done — run `devbase doctor` to verify, then restart your IDE."))
	return nil
}

func isInstalled(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}
