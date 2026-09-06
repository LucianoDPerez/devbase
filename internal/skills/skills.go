// Package skills embeds the starter workflow skills and installs them into
// the project for every IDE that reads them.
package skills

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:generate sh -c "rm -rf data && cp -R ../../skills data"
//go:embed all:data
var files embed.FS

// Names lists the embedded skill names.
func Names() []string {
	var out []string
	entries, _ := fs.ReadDir(files, "data")
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out
}

// Install copies the selected skills to the project's agent skills locations:
// .agents/skills (agent-agnostic) and .claude/skills (Claude Code).
// A nil keep set installs everything. Returns installed SKILL.md paths.
func Install(dir string, keep map[string]bool) ([]string, error) {
	var installed []string
	for _, name := range Names() {
		if keep != nil && !keep[name] {
			continue
		}
		src := filepath.Join("data", name, "SKILL.md")
		data, err := files.ReadFile(filepath.ToSlash(src))
		if err != nil {
			return installed, err
		}
		if len(strings.TrimSpace(string(data))) == 0 {
			return installed, fs.ErrInvalid
		}
		for _, dst := range []string{
			filepath.Join(dir, ".agents", "skills", name, "SKILL.md"),
			filepath.Join(dir, ".claude", "skills", name, "SKILL.md"),
		} {
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return installed, err
			}
			if err := os.WriteFile(dst, data, 0o644); err != nil {
				return installed, err
			}
			installed = append(installed, dst)
		}
	}
	return installed, nil
}
