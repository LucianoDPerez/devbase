// Package adapters maps supported IDEs/agents to their MCP config locations
// and JSON key formats. One row per IDE; core never hardcodes paths.
package adapters

// IDE describes how DevBase reads/writes one agent's configuration.
type IDE struct {
	Name      string
	ConfigRel []string // candidate config paths, "HOME:" or "REPO:" prefixed
	JSONKey   string   // top-level key holding MCP servers
	RulesFile string   // where project rules live for this IDE
}

// Supported lists the Tier-1 IDEs wired by `devbase init`.
func Supported() []IDE {
	return []IDE{
		{
			Name:      "opencode",
			ConfigRel: []string{"HOME:.config/opencode/opencode.json", "REPO:opencode.json"},
			JSONKey:   "mcp",
			RulesFile: "AGENTS.md",
		},
		{
			Name:      "claude-code",
			ConfigRel: []string{"HOME:.claude/settings.json", "REPO:.mcp.json"},
			JSONKey:   "mcpServers",
			RulesFile: "CLAUDE.md",
		},
		{
			Name:      "cursor",
			ConfigRel: []string{"HOME:.cursor/mcp.json", "REPO:.cursor/mcp.json"},
			JSONKey:   "mcpServers",
			RulesFile: ".cursor/rules",
		},
		{
			Name:      "vscode-copilot",
			ConfigRel: []string{"HOME:.copilot/mcp-config.json", "REPO:.vscode/mcp.json", "REPO:.mcp.json"},
			JSONKey:   "servers",
			RulesFile: ".github/copilot-instructions.md",
		},
		{
			Name:      "windsurf",
			ConfigRel: []string{"HOME:.codeium/windsurf/mcp_config.json"},
			JSONKey:   "mcpServers",
			RulesFile: ".windsurf/rules",
		},
		{
			Name:      "codex",
			ConfigRel: []string{"HOME:.codex/config.toml"},
			JSONKey:   "mcp_servers",
			RulesFile: "AGENTS.md",
		},
	}
}
