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
		return false, fmt.Errorf("TOML configs need a manual step: %s", path)
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
