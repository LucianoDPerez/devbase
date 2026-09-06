package detect

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Every supported stack must be detected from realistic marker files.
// This is the contract the packs and the docs rely on.
func TestDetectMatrix(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		want  []string
	}{
		{"next", map[string]string{"package.json": `{"dependencies":{"next":"15","react":"19"}}`},
			[]string{"core", "js", "js/react", "js/nextjs", "security"}},
		{"react", map[string]string{"package.json": `{"dependencies":{"react":"19"}}`},
			[]string{"core", "js", "js/react", "security"}},
		{"typescript", map[string]string{"tsconfig.json": `{}`},
			[]string{"core", "js", "security"}},
		{"laravel", map[string]string{"composer.json": `{"require":{"laravel/framework":"^11"}}`},
			[]string{"core", "php", "php/laravel", "security"}},
		{"django", map[string]string{"requirements.txt": "django==5.0\n"},
			[]string{"core", "python", "python/django", "security"}},
		{"plain-python", map[string]string{"pyproject.toml": "[project]\nname=\"x\"\n"},
			[]string{"core", "python", "security"}},
		{"go", map[string]string{"go.mod": "module x\n"},
			[]string{"core", "go", "security"}},
		{"rust", map[string]string{"Cargo.toml": "[package]\n"},
			[]string{"core", "rust", "security"}},
		{"rails", map[string]string{"Gemfile": "gem 'rails'\n"},
			[]string{"core", "ruby", "ruby/rails", "security"}},
		{"spring", map[string]string{"pom.xml": "<artifactId>spring-boot-starter</artifactId>"},
			[]string{"core", "java", "java/spring", "security"}},
		{"kotlin", map[string]string{"build.gradle.kts": "plugins { kotlin(\"jvm\") }"},
			[]string{"core", "kotlin", "security"}},
		{"csharp", map[string]string{"app.csproj": "<Project/>"},
			[]string{"core", "csharp", "security"}},
		{"cpp", map[string]string{"CMakeLists.txt": "cmake_minimum_required()", "main.cpp": "int main(){}"},
			[]string{"core", "cpp", "security"}},
		{"c", map[string]string{"CMakeLists.txt": "cmake_minimum_required()"},
			[]string{"core", "c", "security"}},
		{"swift", map[string]string{"Package.swift": "// swift-tools"},
			[]string{"core", "swift", "security"}},
		{"flutter", map[string]string{"pubspec.yaml": "name: x\n"},
			[]string{"core", "flutter", "security"}},
		{"sql", map[string]string{"schema.sql": "select 1;"},
			[]string{"core", "sql", "security"}},
		{"empty", map[string]string{},
			[]string{"core", "security"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range tc.files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if got := Detect(dir); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Detect() = %v, want %v", got, tc.want)
			}
		})
	}
}
