package mcpmerge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureEntryRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	entry := map[string]any{"command": "npx", "args": []any{"-y", "x"}}

	changed, err := EnsureEntry(path, "mcpServers", "context7", entry)
	if err != nil || !changed {
		t.Fatalf("first call: changed=%v err=%v", changed, err)
	}
	// Idempotent: second call changes nothing and writes no backup.
	changed, err = EnsureEntry(path, "mcpServers", "context7", entry)
	if err != nil || changed {
		t.Fatalf("second call: changed=%v err=%v", changed, err)
	}
	if _, err := os.Stat(path + ".pre-devbase.bak"); !os.IsNotExist(err) {
		t.Error("no backup expected when file did not exist")
	}
}

func TestEnsureEntryPreservesAndBacksUp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	orig := `{"mcpServers":{"mine":{"command":"srv"}}}`
	if err := os.WriteFile(path, []byte(orig), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := EnsureEntry(path, "mcpServers", "context7", map[string]any{"command": "npx"})
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), `"mine"`) || !strings.Contains(string(data), `"context7"`) {
		t.Errorf("existing entry lost: %s", data)
	}
	backup, err := os.ReadFile(path + ".pre-devbase.bak")
	if err != nil || string(backup) != orig {
		t.Error("backup missing or altered")
	}
	// Existing entry is never overwritten.
	other := map[string]any{"command": "evil"}
	changed, err = EnsureEntry(path, "mcpServers", "mine", other)
	if err != nil || changed {
		t.Fatalf("overwrite attempt: changed=%v err=%v", changed, err)
	}
}

func TestEnsureTomlEntry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	orig := "[mcp_servers]\n\n[mcp_servers.engram]\ncommand = \"engram\"\n"
	if err := os.WriteFile(path, []byte(orig), 0o644); err != nil {
		t.Fatal(err)
	}
	entry := map[string]any{"command": "npx", "args": []any{"-y", "@upstash/context7-mcp@4.0.5"}}
	changed, err := EnsureEntry(path, "mcp_servers", "context7", entry)
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	data, _ := os.ReadFile(path)
	body := string(data)
	for _, want := range []string{`[mcp_servers.engram]`, `[mcp_servers.context7]`, `command = "npx"`, `"@upstash/context7-mcp@4.0.5"`} {
		if !strings.Contains(body, want) {
			t.Errorf("toml missing %q:\n%s", want, body)
		}
	}
	changed, err = EnsureEntry(path, "mcp_servers", "context7", entry)
	if err != nil || changed {
		t.Fatalf("second call: changed=%v err=%v (must be idempotent)", changed, err)
	}
	if _, err := os.Stat(path + ".pre-devbase.bak"); err != nil {
		t.Error("expected backup of pre-existing toml")
	}
}

func TestEnsureEntryRefusals(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.json")
	_ = os.WriteFile(bad, []byte("{nope"), 0o644)
	if _, err := EnsureEntry(bad, "x", "y", map[string]any{}); err == nil {
		t.Error("invalid JSON must be refused")
	}
	if _, err := os.Stat(bad + ".pre-devbase.bak"); !os.IsNotExist(err) {
		t.Error("refused file must not gain a backup")
	}
}
