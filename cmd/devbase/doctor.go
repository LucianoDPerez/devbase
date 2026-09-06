package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/devbase/devbase/internal/adapters"
)

var doctorDeps = []string{"git", "gh", "engram", "codebase-memory-mcp", "semgrep"}

func runDoctor(args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	dir := fs.String("dir", ".", "project directory to inspect")
	if err := fs.Parse(args); err != nil {
		return err
	}
	home, _ := os.UserHomeDir()
	fmt.Fprintln(os.Stdout, "IDEs:")
	for _, ide := range adapters.Supported() {
		status := "✗ not detected"
		for _, rel := range ide.ConfigRel {
			if resolvePath(rel, home, *dir) != "" {
				status = "✓ config found"
				break
			}
		}
		fmt.Fprintf(os.Stdout, "  %-14s %s\n", ide.Name, status)
	}
	fmt.Fprintln(os.Stdout, "Dependencies:")
	for _, dep := range doctorDeps {
		if _, err := exec.LookPath(dep); err != nil {
			fmt.Fprintf(os.Stdout, "  ✗ %-20s missing\n", dep)
		} else {
			fmt.Fprintf(os.Stdout, "  ✓ %s\n", dep)
		}
	}
	return nil
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
