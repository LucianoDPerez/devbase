package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/devbase/devbase/internal/adapters"
	"github.com/devbase/devbase/internal/ui"
)

var doctorDeps = []string{"git", "gh", "engram", "codebase-memory-mcp", "semgrep"}

func runDoctor(args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: devbase doctor [--dir PATH]")
		fmt.Fprintln(os.Stderr, "  Read-only: detected IDE configs, dependency binaries, optional API keys.")
		fs.PrintDefaults()
	}
	dir := fs.String("dir", ".", "project directory to inspect")
	if err := fs.Parse(args); err != nil {
		return err
	}
	out := os.Stdout
	fmt.Fprintln(out, ui.Section("IDEs"))
	for _, ide := range adapters.Supported() {
		if resolvePathFirst(ide, *dir) != "" {
			fmt.Fprintln(out, ui.Ok(ide.Name, "config found"))
		} else {
			fmt.Fprintln(out, ui.Warn(ide.Name, "not detected"))
		}
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, ui.Section("Dependencies"))
	for _, dep := range doctorDeps {
		if isInstalled(dep) {
			fmt.Fprintln(out, ui.Ok(dep, "ready"))
		} else {
			fmt.Fprintln(out, ui.Fail(dep, "missing — devbase setup installs it"))
		}
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, ui.Section("API keys (optional)"))
	if os.Getenv("CONTEXT7_API_KEY") != "" {
		fmt.Fprintln(out, ui.Ok("CONTEXT7_API_KEY", "higher rate limits"))
	} else {
		fmt.Fprintln(out, ui.Warn("CONTEXT7_API_KEY", "basic limits; free key at https://context7.com/dashboard"))
	}
	return nil
}
