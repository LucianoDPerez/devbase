package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devbase/devbase/internal/adapters"
	"github.com/devbase/devbase/internal/pm"
)

// The catalog must cover every installable tool and default everything on,
// except local-only mode (sharing rules with the team stays the default).
func TestBuildCatalogComplete(t *testing.T) {
	cat := buildCatalog(pm.Detect(), t.TempDir(), []string{"core", "js", "security"})
	seen := map[string]bool{}
	for _, e := range cat {
		seen[e.ID] = true
		if e.ID == "opt:gitignore" {
			if e.On {
				t.Error("opt:gitignore defaults on, want off")
			}
			continue
		}
		if !e.On {
			t.Errorf("catalog %s defaults off, want on", e.ID)
		}
		if e.Use == "" {
			t.Errorf("catalog %s missing one-line use", e.ID)
		}
	}
	for _, d := range pm.Deps() {
		if !seen["tool:"+d.Bin] {
			t.Errorf("tool %s missing from catalog", d.Bin)
		}
	}
	for _, id := range []string{"mcp:context7", "mcp:engram", "mcp:codebase-memory", "stack:js", "skill:dev-review"} {
		if !seen[id] {
			t.Errorf("catalog missing %s", id)
		}
	}
}

// Pack taxonomy frontmatter must never leak into rendered rule bodies.
func TestStripFrontmatter(t *testing.T) {
	in := "---\nlevel: heuristic\n---\n\n# Go\n\nbody\n"
	if got := stripFrontmatter(in); got != "# Go\n\nbody\n" {
		t.Errorf("stripFrontmatter = %q", got)
	}
	plain := "# Title\n\nbody\n"
	if got := stripFrontmatter(plain); got != plain {
		t.Errorf("plain body altered: %q", got)
	}
}

// jsManager resolves the package manager from lockfiles.
func TestJsManager(t *testing.T) {
	cases := []struct {
		name  string
		files []string
		want  string
	}{
		{"pnpm", []string{"package.json", "pnpm-lock.yaml"}, "pnpm"},
		{"yarn", []string{"package.json", "yarn.lock"}, "yarn"},
		{"npm", []string{"package.json"}, "npm"},
		{"none", []string{}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range tc.files {
				if err := os.WriteFile(filepath.Join(dir, f), []byte("{}"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if got := jsManager(dir); got != tc.want {
				t.Errorf("jsManager() = %q, want %q", got, tc.want)
			}
		})
	}
}

// IDE picker maps checked entries back to adapters, nothing else.
func TestTargetsFromSelection(t *testing.T) {
	got := targetsFromSelection(adapters.Supported(), map[string]bool{
		"ide:cursor": true, "ide:opencode": false,
	})
	if len(got) != 1 || got[0].Name != "cursor" {
		t.Errorf("targetsFromSelection = %v", got)
	}
	if got := targetsFromSelection(adapters.Supported(), map[string]bool{}); len(got) != 0 {
		t.Errorf("empty selection = %v, want none", got)
	}
}

// Deselected stacks drop out, core always stays.
func TestFilterStacks(t *testing.T) {
	got := filterStacks(
		[]string{"core", "js", "js/react", "security"},
		map[string]bool{"stack:js": true},
	)
	want := map[string]bool{"core": true, "js": true}
	if len(got) != len(want) {
		t.Fatalf("filterStacks = %v", got)
	}
	for _, s := range got {
		if !want[s] {
			t.Errorf("unexpected stack %s", s)
		}
	}
}

// Local-only mode ignores everything created, preserves user content, and is
// idempotent.
func TestWriteGitignore(t *testing.T) {
	dir := t.TempDir()
	user := "# my rules\nnode_modules/\n"
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(user), 0o644); err != nil {
		t.Fatal(err)
	}
	rendered := []string{filepath.Join(dir, "CLAUDE.md")}
	skillPaths := []string{filepath.Join(dir, ".agents", "skills", "dev-review", "SKILL.md")}
	if err := writeGitignore(dir, rendered, skillPaths); err != nil {
		t.Fatal(err)
	}
	if err := writeGitignore(dir, rendered, skillPaths); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
	body := string(data)
	for _, want := range []string{"# my rules", "node_modules/", ".devbase/", "CLAUDE.md",
		".agents/skills/dev-review/SKILL.md", "*.pre-devbase.bak", gitignoreMarker} {
		if !strings.Contains(body, want) {
			t.Errorf("gitignore missing %q:\n%s", want, body)
		}
	}
	if n := strings.Count(body, gitignoreMarker); n != 1 {
		t.Errorf("marker written %d times, want 1", n)
	}
}
