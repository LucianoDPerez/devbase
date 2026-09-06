// Command devbase is the DevBase operating standard CLI.
package main

import (
	"fmt"
	"os"
)

// version is set at release time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	if len(os.Args) < 2 {
		fmt.Print(helpText())
		return
	}
	var err error
	switch os.Args[1] {
	case "init":
		err = runInit(os.Args[2:])
	case "doctor":
		err = runDoctor(os.Args[2:])
	case "gate":
		err = runGate(os.Args[2:])
	case "version", "--version", "-v":
		fmt.Println("devbase " + version)
	case "help", "--help", "-h":
		fmt.Print(helpText())
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		fmt.Fprint(os.Stderr, helpText())
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func helpText() string {
	return `DevBase — operating standard for development agents.

Usage:
  devbase <command> [options]

Commands:
  init      Detect the project stack and write contextual rules to .devbase/
  doctor    Show detected IDEs, dependencies and optional API keys (read-only)
  gate      Run deterministic verification, emit VERIFIED/BLOCKED with evidence

Global options:
  help, --help, -h     Show this help
  version              Print version

Examples:
  devbase doctor
  devbase init --dir /path/to/project
  devbase init --dir . --wire
  devbase gate --dir .
`
}
