package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/devbase/devbase/internal/adapters"
	"github.com/devbase/devbase/internal/detect"
	"github.com/devbase/devbase/internal/mcpmerge"
	"github.com/devbase/devbase/internal/pm"
	"github.com/devbase/devbase/internal/skills"
	"github.com/devbase/devbase/internal/tui"
	"github.com/devbase/devbase/internal/ui"
)

// context7Entry is the uniform stdio MCP entry (npx-based, works in every
// JSON-configured IDE). Codex uses TOML and gets a manual step instead.
var context7Entry = map[string]any{
	"command": "npx",
	"args":    []any{"-y", "@upstash/context7-mcp@4.0.5"},
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
	yes := fs.Bool("yes", false, "accept the full catalog without the interactive checklist")
	if err := fs.Parse(args); err != nil {
		return err
	}
	out := os.Stdout
	var pending []string
	addPending := func(s string) { pending = append(pending, s) }

	info := pm.Detect()
	allStacks := detect.Detect(*dir)

	// 0a. Which IDEs? Flag wins; otherwise ask (detected ones pre-checked).
	var targets []adapters.IDE
	if *ides != "detected" {
		targets = pickIDEs(*dir, *ides)
	} else {
		entries := ideEntries(*dir)
		if !*yes {
			var err error
			entries, err = tui.Select("DevBase — ¿en qué IDEs lo instalamos?", entries)
			if err != nil {
				return err
			}
		}
		targets = targetsFromSelection(adapters.Supported(), tui.SelectedIDs(entries))
		if len(targets) == 0 {
			return fmt.Errorf("no IDE selected — nothing to configure")
		}
	}

	// 0. Catalog checklist: everything on by default, space toggles.
	catalog := buildCatalog(info, allStacks)
	if !*yes {
		var err error
		catalog, err = tui.Select("DevBase — elegí tu arsenal (todo viene activado)", catalog)
		if err != nil {
			return err
		}
	}
	sel := tui.SelectedIDs(catalog)
	stacks := filterStacks(allStacks, sel)

	fmt.Fprintln(out, ui.Section("Environment"))
	fmt.Fprintf(out, "  %s · package manager: %s\n\n", info.OS, info.PM)

	// 1. Dependencies.
	fmt.Fprintln(out, ui.Section("Dependencies"))
	if !*skipInstall {
		for _, d := range pm.Missing() {
			if !sel["tool:"+d.Bin] {
				fmt.Fprintln(out, ui.Warn(d.Name, "deselected"))
				continue
			}
			if d.Bin == "engram" && !sel["mcp:engram"] {
				fmt.Fprintln(out, ui.Warn(d.Name, "deselected with its MCP"))
				continue
			}
			if d.Bin == "codebase-memory-mcp" && !sel["mcp:codebase-memory"] {
				fmt.Fprintln(out, ui.Warn(d.Name, "deselected with its MCP"))
				continue
			}
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
	secs, _, err := writeProject(*dir, stacks)
	if err != nil {
		return err
	}
	rendered, _, err := renderProject(*dir, secs, targets)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, ui.Section("Rules"))
	fmt.Fprintf(out, "  stacks: %s\n", strings.Join(stacks, ", "))
	fmt.Fprintf(out, "  %d native rule files rendered\n\n", len(rendered))

	// 2b. Starter workflow skills (only selected).
	keepSkills := map[string]bool{}
	for _, n := range skills.Names() {
		keepSkills[n] = sel["skill:"+n]
	}
	skillPaths, err := skills.Install(*dir, keepSkills)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, ui.Section("Skills"))
	fmt.Fprintf(out, "  %d workflow skills installed (.agents/skills, .claude/skills)\n\n", len(skills.Names()))

	// 2c. Local-only mode: ignore everything DevBase created.
	if sel["opt:gitignore"] {
		if err := writeGitignore(*dir, rendered, skillPaths); err != nil {
			return err
		}
		fmt.Fprintln(out, ui.Section("Git"))
		fmt.Fprintln(out, "  .gitignore actualizado — nada de DevBase se sube al repo")
		fmt.Fprintln(out)
	}

	// 3. MCP entries (Context7; Engram and codebase-memory self-configure).
	fmt.Fprintln(out, ui.Section("MCP"))
	mcpActive := map[string]bool{}
	wantCtx := sel["mcp:context7"]
	wantEngram := sel["mcp:engram"]
	for _, ide := range targets {
		cfg := resolvePathFirst(ide, *dir)
		if cfg == "" {
			fmt.Fprintln(out, ui.Warn(ide.Name, "no config found"))
			continue
		}
		if !wantCtx {
			fmt.Fprintln(out, ui.Warn(ide.Name, "context7 deselected"))
		} else {
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
	}
	fmt.Fprintln(out)

	// 4. External wiring.
	fmt.Fprintln(out, ui.Section("Wiring"))
	wiredEngram := false
	for _, ide := range targets {
		if resolvePathFirst(ide, *dir) == "" {
			continue
		}
		if !wantEngram {
			fmt.Fprintln(out, ui.Warn(ide.Name, "engram deselected"))
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
	if isInstalled("codebase-memory-mcp") && sel["mcp:codebase-memory"] {
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
		if !sel["skill:"+n] {
			continue
		}
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
		fmt.Fprintln(out, ui.Verdict("PASS", "READY — restart your IDE and work"))
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

// ideEntries builds the IDE picker rows: detected IDEs pre-checked,
// the rest offered unchecked.
func ideEntries(dir string) []tui.Entry {
	var out []tui.Entry
	for _, ide := range adapters.Supported() {
		note := "no detectado"
		on := false
		if resolvePathFirst(ide, dir) != "" {
			note, on = "config encontrada", true
		} else if ide.ProjectMarker != "" {
			if _, err := os.Stat(filepath.Join(dir, ide.ProjectMarker)); err == nil {
				note, on = "usado en este proyecto", true
			}
		}
		out = append(out, tui.Entry{
			Group: "IDEs", ID: "ide:" + ide.Name, Name: ide.Name, Use: "reglas + MCP + wiring", Note: note, On: on,
		})
	}
	return out
}

// targetsFromSelection maps checked IDE entries back to adapters.
func targetsFromSelection(all []adapters.IDE, sel map[string]bool) []adapters.IDE {
	var out []adapters.IDE
	for _, ide := range all {
		if sel["ide:"+ide.Name] {
			out = append(out, ide)
		}
	}
	return out
}

func isInstalled(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}

const gitignoreMarker = "# devbase:managed (solo uso local)"

// writeGitignore appends every DevBase-created path to .gitignore so local-only
// users never commit them. Idempotent: a second run changes nothing.
func writeGitignore(dir string, rendered, skillPaths []string) error {
	var entries []string
	entries = append(entries, ".devbase/")
	entries = append(entries, "*.pre-devbase.bak")
	toRel := func(abs string) string {
		rel, err := filepath.Rel(dir, abs)
		if err != nil {
			return ""
		}
		return filepath.ToSlash(rel)
	}
	for _, p := range append(append([]string{}, rendered...), skillPaths...) {
		if rel := toRel(p); rel != "" {
			entries = append(entries, rel)
		}
	}
	path := filepath.Join(dir, ".gitignore")
	var body string
	if data, err := os.ReadFile(path); err == nil {
		body = string(data)
		if strings.Contains(body, gitignoreMarker) {
			return nil
		}
		if body != "" && !strings.HasSuffix(body, "\n") {
			body += "\n"
		}
	}
	var b strings.Builder
	b.WriteString(body)
	b.WriteString(gitignoreMarker + "\n")
	for _, e := range entries {
		b.WriteString(e + "\n")
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// mcpCatalogUse explains the wired MCP servers.
var mcpCatalogUse = map[string]string{
	"mcp:context7":             "documentación viva de librerías",
	"mcp:engram":               "memoria persistente entre sesiones",
	"mcp:codebase-memory":      "grafo estructural del repo",
	"tool:git":                 "versionado del proyecto",
	"tool:gh":                  "PRs, issues y releases",
	"tool:node":                "servidores MCP vía npx",
	"tool:semgrep":             "seguridad SAST en el gate",
	"tool:engram":              "memoria persistente entre sesiones",
	"tool:codebase-memory-mcp": "grafo estructural del código",
}

// buildCatalog assembles the interactive catalog: MCPs, rules for the detected
// stacks, starter skills, and dependency tools. Everything defaults on.
func buildCatalog(info pm.Info, stacks []string) []tui.Entry {
	var out []tui.Entry
	add := func(group, id, name, use string) {
		out = append(out, tui.Entry{Group: group, ID: id, Name: name, Use: use, On: true})
	}
	add("MCPs", "mcp:context7", "context7", mcpCatalogUse["mcp:context7"])
	add("MCPs", "mcp:engram", "engram", mcpCatalogUse["mcp:engram"])
	add("MCPs", "mcp:codebase-memory", "codebase-memory", mcpCatalogUse["mcp:codebase-memory"])
	for _, s := range stacks {
		use := stackUse[s]
		if use == "" {
			use = "reglas contextuales"
		}
		add("Reglas", "stack:"+s, s, use)
	}
	for _, n := range skills.Names() {
		use := skillUse[n]
		if use == "" {
			use = "workflow del agente"
		}
		add("Skills", "skill:"+n, n, use)
	}
	missing := map[string]bool{}
	for _, d := range pm.Missing() {
		missing[d.Bin] = true
	}
	for _, d := range pm.Deps() {
		note := ""
		if !missing[d.Bin] {
			note = "ya instalado"
		}
		use := toolUse[d.Bin]
		if use == "" {
			use = d.Name
		}
		e := tui.Entry{Group: "Herramientas", ID: "tool:" + d.Bin, Name: d.Name, Use: use, Note: note, On: true}
		out = append(out, e)
	}
	out = append(out, tui.Entry{
		Group: "Opciones", ID: "opt:gitignore", Name: "solo uso local",
		Use: "ignora en git todo lo creado por DevBase (no se sube al repo)", On: false,
	})
	_ = info
	return out
}

// filterStacks keeps detected stacks in order, dropping deselected ones while
// always retaining core (the base everything builds on).
func filterStacks(all []string, sel map[string]bool) []string {
	var out []string
	for _, s := range all {
		if s == "core" || sel["stack:"+s] {
			out = append(out, s)
		}
	}
	return out
}
