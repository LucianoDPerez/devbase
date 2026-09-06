// Package adapters maps supported IDEs/agents to their MCP config locations
// and JSON key formats. One row per IDE; core never hardcodes paths.
package adapters

// IDE describes how DevBase reads/writes one agent's configuration.
type IDE struct {
	Name string
	// ConfigRel are candidate global config paths, "HOME:" or "REPO:" prefixed.
	ConfigRel []string
	// ProjectMarker is a directory/file in the project proving usage.
	ProjectMarker string
	JSONKey       string // top-level key holding MCP servers
	RulesFile     string // legacy single-file rules path (reference only)
}

// Supported lists the Tier-1 IDEs wired by `devbase init`.
func Supported() []IDE {
	return []IDE{
		{
			Name:          "opencode",
			ConfigRel:     []string{"HOME:.config/opencode/opencode.json", "REPO:opencode.json"},
			ProjectMarker: "opencode.json",
			JSONKey:       "mcp",
			RulesFile:     "AGENTS.md",
		},
		{
			Name:          "claude-code",
			ConfigRel:     []string{"HOME:.claude/settings.json", "REPO:.mcp.json"},
			ProjectMarker: ".claude",
			JSONKey:       "mcpServers",
			RulesFile:     "CLAUDE.md",
		},
		{
			Name:          "cursor",
			ConfigRel:     []string{"HOME:.cursor/mcp.json", "REPO:.cursor/mcp.json"},
			ProjectMarker: ".cursor",
			JSONKey:       "mcpServers",
			RulesFile:     ".cursor/rules/",
		},
		{
			Name:          "vscode-copilot",
			ConfigRel:     []string{"HOME:.copilot/mcp-config.json", "REPO:.vscode/mcp.json", "REPO:.mcp.json"},
			ProjectMarker: ".vscode",
			JSONKey:       "servers",
			RulesFile:     ".github/copilot-instructions.md",
		},
		{
			Name:          "windsurf",
			ConfigRel:     []string{"HOME:.codeium/windsurf/mcp_config.json"},
			ProjectMarker: ".windsurf",
			JSONKey:       "mcpServers",
			RulesFile:     ".windsurf/rules/",
		},
		{
			Name:          "codex",
			ConfigRel:     []string{"HOME:.codex/config.toml"},
			ProjectMarker: "",
			JSONKey:       "mcp_servers",
			RulesFile:     "AGENTS.md",
		},
	}
}
