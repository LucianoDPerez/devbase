// Package mcpmerge adds an MCP server entry to an IDE's JSON config without
// touching existing entries. Unknown or TOML configs are refused with an
// error so the caller can print a manual step instead.
package mcpmerge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EnsureEntry adds name→entry under topKey in the JSON file at path, creating
// parent dirs and the file when missing. Existing entries are never
// overwritten. A .pre-devbase.bak backup is written on first modification.
func EnsureEntry(path, topKey, name string, entry map[string]any) (bool, error) {
	if strings.HasSuffix(path, ".toml") {
		return EnsureTomlEntry(path, name, entry)
	}
	data, err := os.ReadFile(path)
	cfg := map[string]any{}
	if err == nil && len(data) > 0 {
		dec := json.NewDecoder(strings.NewReader(string(data)))
		if err := dec.Decode(&cfg); err != nil {
			return false, fmt.Errorf("cannot parse %s (left untouched): %w", path, err)
		}
	} else if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	servers, ok := cfg[topKey].(map[string]any)
	if !ok {
		servers = map[string]any{}
		cfg[topKey] = servers
	}
	if _, exists := servers[name]; exists {
		return false, nil
	}
	if _, err := os.Stat(path); err == nil {
		orig, _ := os.ReadFile(path)
		if err := os.WriteFile(path+".pre-devbase.bak", orig, 0o644); err != nil {
			return false, err
		}
	}
	servers[name] = entry
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	out, _ := json.MarshalIndent(cfg, "", "  ")
	return true, os.WriteFile(path, append(out, '\n'), 0o644)
}

// EnsureTomlEntry appends a [mcp_servers.<name>] table for Codex-style TOML
// configs (verified against `engram setup codex` output). Existing tables are
// never touched. Backup policy mirrors the JSON path.
func EnsureTomlEntry(path, name string, entry map[string]any) (bool, error) {
	command, _ := entry["command"].(string)
	if command == "" {
		return false, fmt.Errorf("toml entry needs a command: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	body := string(data)
	header := "[mcp_servers." + name + "]"
	for _, line := range strings.Split(body, "\n") {
		if strings.TrimSpace(line) == header {
			return false, nil
		}
	}
	if len(data) > 0 {
		if err := os.WriteFile(path+".pre-devbase.bak", data, 0o644); err != nil {
			return false, err
		}
	} else if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	var b strings.Builder
	b.WriteString(body)
	if body != "" && !strings.HasSuffix(body, "\n") {
		b.WriteString("\n")
	}
	b.WriteString("\n" + header + "\n")
	b.WriteString(fmt.Sprintf("command = %q\n", command))
	if args, ok := entry["args"].([]any); ok {
		quoted := make([]string, 0, len(args))
		for _, a := range args {
			s, ok := a.(string)
			if !ok {
				return false, fmt.Errorf("toml args must be strings: %s", path)
			}
			quoted = append(quoted, fmt.Sprintf("%q", s))
		}
		b.WriteString("args = [" + strings.Join(quoted, ", ") + "]\n")
	}
	return true, os.WriteFile(path, []byte(b.String()), 0o644)
}
