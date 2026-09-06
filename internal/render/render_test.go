package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sampleSecs() []Section {
	return []Section{
		{Stack: "core", Body: "# Solid\n\nDo one thing well.\n"},
		{Stack: "security", Body: "# Secrets\n\nNo hardcoded keys.\n"},
		{Stack: "js/react", Body: "# React\n\nHooks over classes.\n"},
	}
}

// Every Tier-1 IDE must render files carrying the managed marker.
func TestFilesAllIDEs(t *testing.T) {
	for _, ide := range []string{"claude-code", "opencode", "codex", "cursor", "vscode-copilot", "windsurf"} {
		files, err := Files(ide, sampleSecs())
		if err != nil {
			t.Errorf("Files(%s): %v", ide, err)
			continue
		}
		if len(files) == 0 {
			t.Errorf("Files(%s): no targets", ide)
		}
		union := ""
		for rel, content := range files {
			if !strings.Contains(content, marker) {
				t.Errorf("Files(%s): %s missing managed marker", ide, rel)
			}
			union += content
		}
		if !strings.Contains(union, "Hooks over classes") {
			t.Errorf("Files(%s): stack content missing from all targets", ide)
		}
	}
}

func TestFilesUnknownIDE(t *testing.T) {
	if _, err := Files("not-an-ide", sampleSecs()); err == nil {
		t.Error("Files(unknown): expected error, got nil")
	}
}

// Cursor core rules must always apply; stack rules must be on-demand so they
// don't tax every request.
func TestCursorFrontmatter(t *testing.T) {
	files, err := Files("cursor", sampleSecs())
	if err != nil {
		t.Fatal(err)
	}
	core, ok := files[filepath.Join(".cursor", "rules", "devbase-core.mdc")]
	if !ok {
		t.Fatal("missing devbase-core.mdc")
	}
	if !strings.Contains(core, "alwaysApply: true") {
		t.Error("core rule must alwaysApply")
	}
	stack := files[filepath.Join(".cursor", "rules", "devbase-js-react.mdc")]
	if !strings.Contains(stack, "alwaysApply: false") {
		t.Error("stack rule must be on-demand (alwaysApply: false)")
	}
	if strings.Contains(stack, "Do one thing well") {
		t.Error("stack rule must not include core content")
	}
}

// Existing user files without the marker are backed up, never lost.
func TestWriteFilesBackup(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "CLAUDE.md")
	if err := os.WriteFile(target, []byte("# mis reglas\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	files, err := Files("claude-code", sampleSecs())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WriteFiles(dir, files); err != nil {
		t.Fatal(err)
	}
	backup, err := os.ReadFile(target + ".pre-devbase.bak")
	if err != nil || string(backup) != "# mis reglas\n" {
		t.Errorf("backup missing or wrong: %q, %v", backup, err)
	}
	// Second run must be idempotent: no new backup content, marker present.
	if _, err := WriteFiles(dir, files); err != nil {
		t.Fatal(err)
	}
	updated, _ := os.ReadFile(target)
	if !strings.Contains(string(updated), marker) {
		t.Error("rewritten file missing marker")
	}
}
