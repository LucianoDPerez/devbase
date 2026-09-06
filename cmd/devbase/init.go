package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/devbase/devbase/internal/adapters"
	"github.com/devbase/devbase/internal/detect"
	"github.com/devbase/devbase/internal/packs"
	"github.com/devbase/devbase/internal/render"
	"github.com/devbase/devbase/internal/ui"
)

// manifest records what init wrote for a project.
type manifest struct {
	Version  string   `json:"version"`
	Stacks   []string `json:"stacks"`
	Rules    []string `json:"rules"`
	Rendered []string `json:"rendered"`
	IDEs     []string `json:"ides"`
}

// engramSetup maps a DevBase IDE name to its `engram setup` target.
// Empty means: no automated setup, print the manual command instead.
var engramSetup = map[string]string{
	"opencode":       "opencode",
	"cursor":         "cursor",
	"vscode-copilot": "vscode-copilot",
	"windsurf":       "windsurf",
	"codex":          "codex",
	"claude-code":    "",
}

func runInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: devbase init [--dir PATH] [--wire] [--ides IDE,...]")
		fmt.Fprintln(os.Stderr, "  Detect the stack, write .devbase/rules, and render each IDE's")
		fmt.Fprintln(os.Stderr, "  native rule files (CLAUDE.md, AGENTS.md, .cursor/rules, ...).")
		fmt.Fprintln(os.Stderr, "  --wire also runs external tool setup (engram) for detected IDEs.")
		fmt.Fprintln(os.Stderr, "  --ides: detected (default), all, or comma list (e.g. cursor,vscode-copilot).")
		fs.PrintDefaults()
	}
	dir := fs.String("dir", ".", "project directory to initialize")
	wire := fs.Bool("wire", false, "also run external tool setup (engram) for detected IDEs")
	ides := fs.String("ides", "detected", "which IDEs to render rules for")
	if err := fs.Parse(args); err != nil {
		return err
	}

	stacks, secs, written, err := writeProject(*dir)
	if err != nil {
		return err
	}
	targets := pickIDEs(*dir, *ides)
	rendered, owners, err := renderProject(*dir, secs, targets)
	if err != nil {
		return err
	}

	var ideNames []string
	for _, ide := range targets {
		ideNames = append(ideNames, ide.Name)
	}
	m := manifest{Version: version, Stacks: stacks, Rules: written, Rendered: rendered, IDEs: ideNames}
	data, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(filepath.Join(*dir, ".devbase", "devbase.json"), append(data, '\n'), 0o644); err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, ui.Section("Rules"))
	fmt.Fprintf(os.Stdout, "  stacks: %s\n", strings.Join(stacks, ", "))
	fmt.Fprintf(os.Stdout, "  %d rule files in %s\n", len(written), filepath.Join(*dir, ".devbase", "rules"))
	for rel, names := range owners {
		fmt.Fprintln(os.Stdout, ui.Ok(strings.Join(names, ","), rel))
	}
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, ui.Section("Wiring"))

	for _, ide := range targets {
		if resolvePathFirst(ide, *dir) == "" {
			continue
		}
		if *wire {
			wireIDE(ide)
		} else {
			fmt.Fprintln(os.Stdout, ui.Warn(ide.Name, "run with --wire to configure"))
		}
	}
	return nil
}

// writeProject detects the stack and writes .devbase/rules.
func writeProject(dir string) ([]string, []render.Section, []string, error) {
	stacks := detect.Detect(dir)
	written, err := packs.Write(dir, stacks)
	if err != nil {
		return nil, nil, nil, err
	}
	secs, err := loadSections(dir, written)
	if err != nil {
		return nil, nil, nil, err
	}
	return stacks, secs, written, nil
}

// renderProject generates native rule files for targets, deduping shared
// paths (e.g. AGENTS.md for opencode+codex).
func renderProject(dir string, secs []render.Section, targets []adapters.IDE) ([]string, map[string][]string, error) {
	merged := map[string]string{}
	owners := map[string][]string{}
	for _, ide := range targets {
		files, err := render.Files(ide.Name, secs)
		if err != nil {
			return nil, nil, err
		}
		for rel, content := range files {
			if _, dup := merged[rel]; !dup {
				merged[rel] = content
			}
			owners[rel] = append(owners[rel], ide.Name)
		}
	}
	rendered, err := render.WriteFiles(dir, merged)
	return rendered, owners, err
}

// wireIDE runs the external tool setup for one IDE.
func wireIDE(ide adapters.IDE) {
	target, ok := engramSetup[ide.Name]
	switch {
	case !ok || target == "":
		fmt.Fprintln(os.Stdout, ui.Warn(ide.Name, "manual step — engram docs for this agent"))
	default:
		if _, err := exec.LookPath("engram"); err != nil {
			fmt.Fprintln(os.Stdout, ui.Warn(ide.Name, "SKIP (engram not installed)"))
			return
		}
		cmd := exec.Command("engram", "setup", target)
		if out, err := cmd.CombinedOutput(); err != nil {
			fmt.Fprintln(os.Stdout, ui.Fail(ide.Name, fmt.Sprintf("%v: %s", err, firstLine(string(out)))))
		} else {
			fmt.Fprintln(os.Stdout, ui.Ok(ide.Name, "wired"))
		}
	}
}

// loadSections reads the written rule files back into stack-scoped sections.
func loadSections(dir string, written []string) ([]render.Section, error) {
	var secs []render.Section
	for _, dst := range written {
		rel, err := filepath.Rel(filepath.Join(dir, ".devbase", "rules"), dst)
		if err != nil {
			return nil, err
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		stack := parts[0]
		if len(parts) > 2 {
			stack = strings.Join(parts[:len(parts)-1], "/")
		}
		data, err := os.ReadFile(dst)
		if err != nil {
			return nil, err
		}
		secs = append(secs, render.Section{Stack: stack, Body: string(data)})
	}
	return secs, nil
}

// pickIDEs selects which IDEs to render for: "detected", "all", or a list.
func pickIDEs(dir, spec string) []adapters.IDE {
	all := adapters.Supported()
	if spec == "all" {
		return all
	}
	if spec != "detected" {
		want := map[string]bool{}
		for _, n := range strings.Split(spec, ",") {
			want[strings.TrimSpace(n)] = true
		}
		var out []adapters.IDE
		for _, ide := range all {
			if want[ide.Name] {
				out = append(out, ide)
			}
		}
		return out
	}
	var out []adapters.IDE
	for _, ide := range all {
		if resolvePathFirst(ide, dir) != "" {
			out = append(out, ide)
			continue
		}
		if ide.ProjectMarker != "" {
			if _, err := os.Stat(filepath.Join(dir, ide.ProjectMarker)); err == nil {
				out = append(out, ide)
			}
		}
	}
	return out
}

func resolvePathFirst(ide adapters.IDE, dir string) string {
	home, _ := os.UserHomeDir()
	for _, rel := range ide.ConfigRel {
		if p := resolvePath(rel, home, dir); p != "" {
			return p
		}
	}
	return ""
}

func resolvePath(rel, home, dir string) string {
	var p string
	switch {
	case strings.HasPrefix(rel, "HOME:"):
		p = filepath.Join(home, strings.TrimPrefix(rel, "HOME:"))
	case strings.HasPrefix(rel, "REPO:"):
		p = filepath.Join(dir, strings.TrimPrefix(rel, "REPO:"))
	default:
		return ""
	}
	if _, err := os.Stat(p); err != nil {
		return ""
	}
	return p
}
