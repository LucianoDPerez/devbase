package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/devbase/devbase/internal/detect"
)

func runInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	dir := fs.String("dir", ".", "project directory to initialize")
	if err := fs.Parse(args); err != nil {
		return err
	}
	stacks := detect.Detect(*dir)
	fmt.Fprintf(os.Stdout, "detected stacks: %v\n", stacks)
	fmt.Fprintln(os.Stdout, "init: rule packs + IDE wiring not implemented yet (next slice)")
	return nil
}
