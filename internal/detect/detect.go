// Package detect identifies the project stack from marker files so DevBase
// loads only the contextual rule packs that apply.
package detect

import (
	"os"
	"path/filepath"
)

// Stack is a detected technology stack identifier, e.g. "php/laravel".
type Stack string

// Detect inspects dir for well-known marker files and returns the matching
// stack packs in load order (generic first, specific last).
func Detect(dir string) []Stack {
	has := func(name string) bool {
		_, err := os.Stat(filepath.Join(dir, name))
		return err == nil
	}
	stacks := []Stack{"core"}
	switch {
	case has("artisan") || has("composer.json"):
		stacks = append(stacks, "php", "php/laravel")
	case has("package.json"):
		stacks = append(stacks, "js")
	case has("go.mod"):
		stacks = append(stacks, "go")
	case has("requirements.txt") || has("pyproject.toml"):
		stacks = append(stacks, "python")
	}
	return append(stacks, "security")
}
