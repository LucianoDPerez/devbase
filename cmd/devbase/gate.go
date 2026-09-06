package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/devbase/devbase/internal/gate"
)

func runGate(args []string) error {
	fs := flag.NewFlagSet("gate", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: devbase gate [--dir PATH]")
		fmt.Fprintln(os.Stderr, "  Deterministic checks (build per stack + Semgrep ERROR-only) pinned")
		fmt.Fprintln(os.Stderr, "  to the git SHA. Prints VERIFIED/BLOCKED, exits 1 when BLOCKED.")
		fs.PrintDefaults()
	}
	dir := fs.String("dir", ".", "project directory to verify")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rep := gate.Run(*dir)
	fmt.Fprintf(os.Stdout, "sha: %s\n", rep.SHA)
	for _, r := range rep.Results {
		fmt.Fprintf(os.Stdout, "%-12s %-6s %s\n", r.Name, r.Status, firstLine(r.Detail))
	}
	verdict := rep.Verdict()
	fmt.Fprintf(os.Stdout, "verdict: %s\n", verdict)
	if verdict == gate.Fail {
		os.Exit(1)
	}
	return nil
}

func firstLine(s string) string {
	for i, c := range s {
		if c == '\n' {
			return s[:i]
		}
	}
	if len(s) > 160 {
		return s[:160] + "…"
	}
	return s
}
