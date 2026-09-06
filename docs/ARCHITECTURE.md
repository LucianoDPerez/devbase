# DevBase Architecture

`devbase init` detects the stack (`internal/detect`) and loads only matching
packs from `packs/` (generic first: `core`, last: `security`).

`devbase doctor` reports detected IDEs (`internal/adapters`, one row per IDE)
and missing dependencies. It never writes without asking.

`devbase gate` runs deterministic checks (tests, lint, build, Semgrep
ERROR-only) against the exact git SHA and emits VERIFIED / BLOCKED with
evidence. LLM review is advisory only and can never mark VERIFIED.

External tools (Engram, codebase-memory-mcp) self-configure via their own
installers; DevBase invokes them instead of duplicating their agent wiring.
GitHub access defaults to `gh` CLI + skill; GitHub MCP is a fallback for
shell-less environments.
