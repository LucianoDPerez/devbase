// Package detect identifies the project stack from marker files and dependency
// manifests so DevBase loads only the contextual rule packs that apply.
package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Detect returns stack packs in load order (generic first, specific last,
// "security" always last).
func Detect(dir string) []string {
	stacks := []string{"core"}
	content := readMany(dir, []string{
		"package.json", "composer.json", "requirements.txt", "pyproject.toml",
		"Gemfile", "Cargo.toml", "pubspec.yaml", "pom.xml",
	})

	if has(dir, "composer.json") || has(dir, "artisan") {
		stacks = append(stacks, "php")
		if contains(content, "laravel/framework") {
			stacks = append(stacks, "php/laravel")
		}
	}
	if has(dir, "package.json") || has(dir, "tsconfig.json") {
		stacks = append(stacks, "js")
		switch {
		case contains(content, `"next"`) || contains(content, "next/"):
			stacks = append(stacks, "js/react", "js/nextjs")
		case contains(content, `"react"`):
			stacks = append(stacks, "js/react")
		}
	}
	if has(dir, "requirements.txt") || has(dir, "pyproject.toml") {
		stacks = append(stacks, "python")
		if contains(content, "django") {
			stacks = append(stacks, "python/django")
		}
	}
	if has(dir, "go.mod") {
		stacks = append(stacks, "go")
	}
	if has(dir, "Cargo.toml") {
		stacks = append(stacks, "rust")
	}
	if has(dir, "Gemfile") {
		stacks = append(stacks, "ruby")
		if contains(content, "rails") {
			stacks = append(stacks, "ruby/rails")
		}
	}
	if globAny(dir, []string{"*.csproj", "*.sln"}) {
		stacks = append(stacks, "csharp")
	}
	if has(dir, "CMakeLists.txt") || globAny(dir, []string{"*.cpp", "*.hpp", "*.cc"}) {
		if globAny(dir, []string{"*.cpp", "*.hpp", "*.cc"}) {
			stacks = append(stacks, "cpp")
		} else {
			stacks = append(stacks, "c")
		}
	}
	if has(dir, "pom.xml") || has(dir, "build.gradle") || has(dir, "build.gradle.kts") {
		if has(dir, "build.gradle.kts") || contains(content, "kotlin") {
			stacks = append(stacks, "kotlin")
		} else {
			stacks = append(stacks, "java")
		}
		if contains(content, "spring") {
			stacks = append(stacks, "java/spring")
		}
	}
	if has(dir, "Package.swift") || globAny(dir, []string{"*.xcodeproj"}) {
		stacks = append(stacks, "swift")
	}
	if has(dir, "pubspec.yaml") {
		stacks = append(stacks, "flutter")
	}
	if globAny(dir, []string{"*.sql"}) {
		stacks = append(stacks, "sql")
	}
	return append(stacks, "security")
}

// WebFrontend reports whether dir (or a conventional subdir like frontend/,
// web/, client/) hosts a browser UI: a frontend framework dependency or a
// bundler/framework config file. Backend-only Node (express, fastify) does
// not count — Playwright needs a UI to drive.
func WebFrontend(dir string) bool {
	subs := []string{".", "frontend", "web", "client", "app", "ui"}
	for _, sub := range subs {
		base := filepath.Join(dir, sub)
		manifest := readMany(base, []string{"package.json"})
		if manifest == "" {
			continue
		}
		for _, fw := range []string{`"react"`, `"vue"`, `"angular"`, `"svelte"`, `"next"`, `"nuxt"`, `"solid-js"`, `"preact"`, `"@vitejs/`, `"vite"`, `"webpack"`} {
			if contains(manifest, fw) {
				return true
			}
		}
		if globAny(base, []string{"vite.config.*", "next.config.*", "vue.config.*", "angular.json", "nuxt.config.*", "svelte.config.*"}) {
			return true
		}
	}
	return false
}

func has(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}

func globAny(dir string, patterns []string) bool {
	for _, p := range patterns {
		if m, _ := filepath.Glob(filepath.Join(dir, p)); len(m) > 0 {
			return true
		}
	}
	return false
}

// readMany concatenates small manifest files for keyword sniffing. package.json
// dependencies are also flattened so `"react"` matches `"react": "^19"`.
func readMany(dir string, names []string) string {
	var b strings.Builder
	for _, n := range names {
		data, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil || len(data) > 200_000 {
			continue
		}
		b.Write(data)
		b.WriteByte('\n')
		if n == "package.json" {
			var pkg struct {
				Deps    map[string]string `json:"dependencies"`
				DevDeps map[string]string `json:"devDependencies"`
			}
			if json.Unmarshal(data, &pkg) == nil {
				for d := range pkg.Deps {
					b.WriteString(`"` + d + `"` + "\n")
				}
				for d := range pkg.DevDeps {
					b.WriteString(`"` + d + `"` + "\n")
				}
			}
		}
	}
	return strings.ToLower(b.String())
}

func contains(haystack, needle string) bool {
	return strings.Contains(haystack, strings.ToLower(needle))
}
