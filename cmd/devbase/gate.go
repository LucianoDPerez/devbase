package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/devbase/devbase/internal/gate"
	"github.com/devbase/devbase/internal/ui"
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
	fmt.Fprintln(os.Stdout, ui.Section("Evidence @ "+rep.SHA))
	for _, r := range rep.Results {
		switch r.Status {
		case gate.Fail:
			fmt.Fprintln(os.Stdout, ui.Fail(r.Name, firstLine(r.Detail)))
		case gate.Skip:
			fmt.Fprintln(os.Stdout, ui.Warn(r.Name, firstLine(r.Detail)))
		default:
			fmt.Fprintln(os.Stdout, ui.Ok(r.Name, firstLine(r.Detail)))
		}
	}
	fmt.Fprintln(os.Stdout)
	verdict := rep.Verdict()
	fmt.Fprintln(os.Stdout, ui.Verdict(verdict == gate.Pass, rep.SHA))
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
