package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/devbase/devbase/internal/gate"
	"github.com/devbase/devbase/internal/ui"
)

func runGate(args []string) error {
	fs := flag.NewFlagSet("gate", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: devbase gate [--dir PATH] [--json]")
		fmt.Fprintln(os.Stderr, "  Deterministic checks (build per stack + Semgrep ERROR-only) pinned")
		fmt.Fprintln(os.Stderr, "  to commit + working-tree hash. Prints VERIFIED/BLOCKED/INCOMPLETE,")
		fmt.Fprintln(os.Stderr, "  exits 0/1/2. --json emits the devbase.evidence.v1 envelope.")
		fs.PrintDefaults()
	}
	dir := fs.String("dir", ".", "project directory to verify")
	asJSON := fs.Bool("json", false, "emit machine-readable evidence envelope")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rep := gate.Run(*dir)
	ev := gate.Evidence{Schema: gate.EvidenceSchema, Verdict: rep.Verdict(), Report: rep}
	if *asJSON {
		data, _ := json.MarshalIndent(ev, "", "  ")
		fmt.Fprintln(os.Stdout, string(data))
	} else {
		fmt.Fprintln(os.Stdout, ui.Section("Evidence @ "+rep.SHA+"+"+rep.WorkingTree))
		for _, r := range rep.Results {
			req := ""
			if r.Required {
				req = " [required]"
			}
			switch r.Status {
			case gate.Fail:
				fmt.Fprintln(os.Stdout, ui.Fail(r.Name+req, firstLine(r.Detail)))
			case gate.Skip:
				fmt.Fprintln(os.Stdout, ui.Warn(r.Name+req, firstLine(r.Detail)))
			default:
				fmt.Fprintln(os.Stdout, ui.Ok(r.Name+req, firstLine(r.Detail)))
			}
		}
		fmt.Fprintln(os.Stdout)
		fmt.Fprintln(os.Stdout, ui.Verdict(string(ev.Verdict), rep.SHA))
	}
	writeArtifact(*dir, ev)
	switch ev.Verdict {
	case gate.Fail:
		os.Exit(1)
	case gate.Incomplete:
		os.Exit(2)
	}
	return nil
}

// writeArtifact persists the evidence envelope next to the project.
// Best effort: a gate that cannot record evidence says so but keeps its verdict.
func writeArtifact(dir string, ev gate.Evidence) {
	data, _ := json.Marshal(ev)
	evDir := filepath.Join(dir, ".devbase", "evidence")
	if err := os.MkdirAll(evDir, 0o755); err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(evDir, "latest.json"), append(data, '\n'), 0o644)
	_ = os.WriteFile(filepath.Join(evDir, ev.Report.SHA+".json"), append(data, '\n'), 0o644)
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
