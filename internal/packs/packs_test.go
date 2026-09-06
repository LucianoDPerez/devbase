package packs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devbase/devbase/internal/detect"
)

// Every pack file must declare its enforcement taxonomy so the gate and the
// agents know what a rule is: advisory, heuristic, or deterministic.
func TestPackTaxonomy(t *testing.T) {
	allowed := map[string]bool{"advisory": true, "heuristic": true, "deterministic": true}
	var count int
	err := filepath.Walk("../../packs", func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".md") {
			return err
		}
		count++
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		lines := strings.Split(string(data), "\n")
		if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
			t.Errorf("%s: missing frontmatter", p)
			return nil
		}
		level := ""
		closed := false
		for _, l := range lines[1:] {
			if strings.TrimSpace(l) == "---" {
				closed = true
				break
			}
			if v, ok := strings.CutPrefix(strings.TrimSpace(l), "level:"); ok {
				level = strings.TrimSpace(v)
			}
		}
		if !closed {
			t.Errorf("%s: frontmatter not closed", p)
		} else if !allowed[level] {
			t.Errorf("%s: level %q not in advisory/heuristic/deterministic", p, level)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatal("no pack files found")
	}
}

// Every pack directory must write at least one non-empty rule file.
func TestWriteAllStacks(t *testing.T) {
	for _, stack := range Stacks() {
		dir := t.TempDir()
		written, err := Write(dir, []string{stack})
		if err != nil {
			t.Errorf("Write(%s): %v", stack, err)
			continue
		}
		if len(written) == 0 {
			t.Errorf("Write(%s): no files written", stack)
			continue
		}
		for _, f := range written {
			data, err := os.ReadFile(f)
			if err != nil || len(data) == 0 {
				t.Errorf("Write(%s): %s empty or unreadable", stack, f)
			}
		}
	}
}

// Detection must never return a stack that has no pack. This test fails the
// moment someone adds a stack to detect without its rules.
func TestDetectStacksHavePacks(t *testing.T) {
	known := map[string]bool{}
	for _, s := range Stacks() {
		known[s] = true
	}
	fixtures := []map[string]string{
		{"package.json": `{"dependencies":{"next":"15","react":"19"}}`},
		{"package.json": `{"dependencies":{"react":"19"}}`},
		{"tsconfig.json": `{}`},
		{"composer.json": `{"require":{"laravel/framework":"^11"}}`},
		{"requirements.txt": "django==5.0\n"},
		{"pyproject.toml": "[project]\n"},
		{"go.mod": "module x\n"},
		{"Cargo.toml": "[package]\n"},
		{"Gemfile": "gem 'rails'\n"},
		{"pom.xml": "<artifactId>spring-boot-starter</artifactId>"},
		{"build.gradle.kts": "plugins { kotlin(\"jvm\") }"},
		{"app.csproj": "<Project/>"},
		{"CMakeLists.txt": "x", "main.cpp": "x"},
		{"CMakeLists.txt": "x"},
		{"Package.swift": "x"},
		{"pubspec.yaml": "name: x\n"},
		{"schema.sql": "select 1;"},
		{},
	}
	for i, files := range fixtures {
		dir := t.TempDir()
		for name, content := range files {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		for _, stack := range detect.Detect(dir) {
			if stack == "security" {
				continue // cross-cutting pack, always present
			}
			if !known[stack] {
				t.Errorf("fixture %d: detected stack %q has no pack", i, stack)
			}
		}
	}
}
