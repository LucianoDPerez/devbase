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
	"github.com/devbase/devbase/internal/skills"
	"github.com/devbase/devbase/internal/ui"
)

// context7Entry is the uniform stdio MCP entry (npx-based, works in every
// JSON-configured IDE). Codex uses TOML and gets a manual step instead.
var context7Entry = map[string]any{
	"command": "npx",
	"args":    []any{"-y", "@upstash/context7-mcp"},
}

// toolUse explains what each dependency binary is for.
var toolUse = map[string]string{
	"git":                 "versionado del proyecto",
	"gh":                  "PRs, issues y releases de GitHub",
	"node":                "servidores MCP vía npx (p. ej. Context7)",
	"semgrep":             "seguridad SAST en el gate",
	"engram":              "memoria persistente entre sesiones",
	"codebase-memory-mcp": "grafo estructural del código",
}

// stackUse explains what each rule pack is for.
var stackUse = map[string]string{
	"core": "diseño limpio y SOLID", "security": "OWASP Top 10 y secretos",
	"php": "estándares PHP", "php/laravel": "convenciones Laravel",
	"js": "estándares JS/TS", "js/react": "patrones React", "js/nextjs": "App Router y server components",
	"python": "estándares Python", "python/django": "ORM, vistas y migraciones",
	"go": "build, vet y tests", "rust": "ownership, clippy y unsafe auditado",
	"ruby": "estilo y bundle audit", "ruby/rails": "MVC, strong params y jobs",
	"csharp": "async y nullables en .NET", "c": "C estricto y sanitizers",
	"cpp": "C++ moderno y clang-tidy", "sql": "queries parametrizadas e índices",
	"kotlin": "null safety y corrutinas", "swift": "optionals y concurrencia",
	"flutter": "widgets tontos y analyze limpio",
}

// skillUse explains what each starter skill is for.
var skillUse = map[string]string{
	"dev-review": "revisión pre-commit con evidencia",
	"dev-commit": "commits convencionales con gate previo",
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
	var pending []string
	addPending := func(s string) { pending = append(pending, s) }

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
				addPending("Install " + d.Name + " manually: " + d.Manual)
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
				addPending("Install " + d.Name + " manually: " + d.Manual)
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
			addPending("Authenticate GitHub: gh auth login")
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

	// 2b. Starter workflow skills.
	if _, err := skills.Install(*dir); err != nil {
		return err
	}
	fmt.Fprintln(out, ui.Section("Skills"))
	fmt.Fprintf(out, "  %d workflow skills installed (.agents/skills, .claude/skills)\n\n", len(skills.Names()))

	// 3. MCP entries (Context7; Engram and codebase-memory self-configure).
	fmt.Fprintln(out, ui.Section("MCP"))
	mcpActive := map[string]bool{}
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
			if ide.Name == "codex" {
				addPending("Codex: add the context7 server to " + cfg + " manually (TOML)")
			}
		case changed:
			fmt.Fprintln(out, ui.Ok(ide.Name, "context7 added"))
			mcpActive["context7"] = true
		default:
			fmt.Fprintln(out, ui.Ok(ide.Name, "context7 present"))
			mcpActive["context7"] = true
		}
	}
	fmt.Fprintln(out)

	// 4. External wiring.
	fmt.Fprintln(out, ui.Section("Wiring"))
	wiredEngram := false
	for _, ide := range targets {
		if resolvePathFirst(ide, *dir) == "" {
			continue
		}
		status, detail := printWireRow(ide)
		if status == "fail" {
			addPending("Retry wiring " + ide.Name + ": " + detail)
		} else if ide.Name == "claude-code" {
			addPending("Claude Code: claude plugin marketplace add Gentleman-Programming/engram && claude plugin install engram")
		}
		if status == "ok" {
			wiredEngram = true
		}
	}
	fmt.Fprintln(out)
	if isInstalled("codebase-memory-mcp") {
		mcpActive["codebase-memory"] = true
	}

	// 5. What you got, grouped by utility.
	fmt.Fprintln(out, ui.Section("Tu DevBase incluye"))
	fmt.Fprintln(out, "  MCPs (datos vivos para el agente):")
	if mcpActive["context7"] {
		fmt.Fprintln(out, "    · context7 — documentación actual de librerías")
	}
	if wiredEngram {
		mcpActive["engram"] = true
		fmt.Fprintln(out, "    · engram — memoria persistente entre sesiones")
	}
	if mcpActive["codebase-memory"] {
		fmt.Fprintln(out, "    · codebase-memory — grafo estructural del repo")
	}
	fmt.Fprintln(out, "  Skills (cómo trabaja el agente):")
	for _, n := range skills.Names() {
		use := skillUse[n]
		if use == "" {
			use = "workflow del agente"
		}
		fmt.Fprintf(out, "    · %s — %s\n", n, use)
	}
	fmt.Fprintln(out, "  Reglas (cómo debe salir el código):")
	for _, s := range stacks {
		use := stackUse[s]
		if use == "" {
			use = "reglas contextuales"
		}
		fmt.Fprintf(out, "    · %s — %s\n", s, use)
	}
	fmt.Fprintln(out, "  Herramientas (binarios en tu PATH):")
	for _, d := range pm.Deps() {
		if !isInstalled(d.Bin) {
			continue
		}
		use := toolUse[d.Bin]
		if use == "" {
			use = d.Name
		}
		fmt.Fprintf(out, "    · %s — %s\n", d.Bin, use)
	}
	fmt.Fprintln(out)

	if os.Getenv("CONTEXT7_API_KEY") == "" {
		fmt.Fprintln(out, ui.Dim("Optional: free Context7 key with higher limits at https://context7.com/dashboard"))
		fmt.Fprintln(out)
	}
	if len(pending) == 0 {
		fmt.Fprintln(out, ui.Verdict(true, "READY — restart your IDE and work"))
	} else {
		fmt.Fprintln(out, ui.Section("Pendiente manual"))
		for i, p := range pending {
			fmt.Fprintf(out, "  %d. %s\n", i+1, p)
		}
		fmt.Fprintln(out)
		fmt.Fprintln(out, ui.Dim("then restart your IDE."))
	}
	return nil
}

func isInstalled(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}
