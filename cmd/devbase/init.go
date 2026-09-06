package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/devbase/devbase/internal/adapters"
	"github.com/devbase/devbase/internal/detect"
	"github.com/devbase/devbase/internal/packs"
)

// manifest records what init wrote for a project.
type manifest struct {
	Version string   `json:"version"`
	Stacks  []string `json:"stacks"`
	Rules   []string `json:"rules"`
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
		fmt.Fprintln(os.Stderr, "Usage: devbase init [--dir PATH] [--wire]")
		fmt.Fprintln(os.Stderr, "  Detect the stack and write contextual rules to .devbase/.")
		fmt.Fprintln(os.Stderr, "  --wire also runs external tool setup (engram) for detected IDEs.")
		fs.PrintDefaults()
	}
	dir := fs.String("dir", ".", "project directory to initialize")
	wire := fs.Bool("wire", false, "also run external tool setup (engram) for detected IDEs")
	if err := fs.Parse(args); err != nil {
		return err
	}

	stacks := detect.Detect(*dir)
	written, err := packs.Write(*dir, stacks)
	if err != nil {
		return err
	}
	m := manifest{Version: version, Stacks: stacks, Rules: written}
	data, _ := json.MarshalIndent(m, "", "  ")
	manifestPath := filepath.Join(*dir, ".devbase", "devbase.json")
	if err := os.WriteFile(manifestPath, append(data, '\n'), 0o644); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "stacks: %v\n", stacks)
	fmt.Fprintf(os.Stdout, "wrote %d rule files to %s\n", len(written), filepath.Join(*dir, ".devbase", "rules"))

	for _, ide := range adapters.Supported() {
		if resolvePathFirst(ide, *dir) == "" {
			continue
		}
		target, ok := engramSetup[ide.Name]
		switch {
		case !*wire:
			fmt.Fprintf(os.Stdout, "wire %s: run with --wire to configure\n", ide.Name)
		case !ok || target == "":
			fmt.Fprintf(os.Stdout, "wire %s: manual step — engram docs for this agent\n", ide.Name)
		default:
			if _, err := exec.LookPath("engram"); err != nil {
				fmt.Fprintf(os.Stdout, "wire %s: SKIP (engram not installed)\n", ide.Name)
				continue
			}
			cmd := exec.Command("engram", "setup", target)
			if out, err := cmd.CombinedOutput(); err != nil {
				fmt.Fprintf(os.Stdout, "wire %s: FAIL (%v: %s)\n", ide.Name, err, string(out))
			} else {
				fmt.Fprintf(os.Stdout, "wire %s: OK\n", ide.Name)
			}
		}
	}
	return nil
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
