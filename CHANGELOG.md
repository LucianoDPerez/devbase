# Changelog

All notable changes to DevBase. Format follows Keep a Changelog; versions follow SemVer.

## [v0.15.0] — 2026-09-06

- Missing toolchains report required INCOMPLETE instead of false FAIL.
- Codex `config.toml` gets real context7 merge (schema verified from engram output).

## [v0.14.0] — 2026-09-06

- Unit-test matrix on ubuntu/windows/macos; installer smoke jobs execute install.sh and install.ps1 for real.
- apt Node recipe removed (distro nodejs breaks npx MCPs — honest manual step instead); LF line endings enforced for gofmt on Windows.

## [v0.13.0] — 2026-09-06

- Always-on E2E battery (`scripts/e2e.sh` + CI job): clean-go, broken-node, vulnerable-py, empty-dir and dirty-tree fixtures asserting verdicts, evidence uploaded as CI artifact.

## [v0.12.0] — 2026-09-06

- Gate runs real project test suites (go test, npm test, pytest, PHPUnit/Pest, cargo test, maven/gradle, rspec, dotnet test); missing runner on a detected suite is INCOMPLETE.
- Playwright installable from setup for web frontends (dep + chromium).

## [v0.11.0] — 2026-09-06

- Strict evidence gate: required/optional checks, INCOMPLETE verdict (exit 2), commit + working-tree hash, `gate --json` (`devbase.evidence.v1`), `.devbase/evidence` artifacts, toolchain capture.
- Pack taxonomy frontmatter (advisory/heuristic/deterministic) with meta-test; pinned Context7 MCP.

## [v0.10.0] — 2026-09-06

- IDE picker screen in setup; highlighted installer next step.

## [v0.9.0] — 2026-09-06

- Interactive catalog checklist (MCPs/rules/skills/tools, all-on by default); local-only `.gitignore` mode; `--yes` for CI.

## [v0.7.0] — 2026-09-06

- Styled sectioned output; setup ends with a personalized pending checklist or READY.

## [v0.6.0] — 2026-09-06

- Conditional Playwright E2E in gate and setup hint for web frontends.

## [v0.5.0] — 2026-09-06

- `devbase setup`: per-OS dependency install (brew/apt/dnf/pacman/winget), Context7 MCP merge, external wiring.

## [v0.4.0] — 2026-09-06

- 15 new language/framework packs with dependency-aware detection (14-case matrix).

## [v0.3.0] — 2026-09-06

- Native IDE rule rendering (CLAUDE.md, AGENTS.md, `.cursor/rules`, copilot-instructions, windsurf rules) with backup policy.

## [v0.2.0] — 2026-09-06

- One-line installers, real `init` with packs, deterministic gate, full `help`.

## [v0.1.0] — 2026-09-06

- Initial scaffold: `doctor`, stack detection, IDE matrix; collaborator baseline (CI, SECURITY, Dependabot) and MIT license.
