# DevBase

**One command that leaves any dev with the full arsenal: proven tools, contextual rules, and evidence-based verification — on their OS and their IDE.**

```bash
curl -fsSL https://raw.githubusercontent.com/LucianoDPerez/devbase/main/install.sh | bash
cd your-project
devbase setup --dir .
```

*Leer en [español](README.es.md).*

## The problem

Every dev loses days to the same chores: hunting down which MCPs to install, copy-pasting scattered rules from the internet, fighting per-IDE configs (Cursor uses one key, VS Code another, Codex TOML), while the agent hallucinates APIs and declares "done" with no proof. DevBase attacks all three at the root.

## What it does for you

1. **Installs** what's missing for your system (brew/apt/dnf/pacman/winget): `gh`, Node, Semgrep, Engram, codebase-memory — or tells you exactly where to get it when there's no automated recipe.
2. **Detects your stack** (PHP/Laravel, React/Next, Python/Django, Go, Rust, Ruby, Java, C#, Kotlin, Swift, Flutter and more) and writes only the rules that apply, in each IDE's native format.
3. **Configures your IDEs** (OpenCode, Claude Code, Cursor, VS Code + Copilot, Windsurf, Codex): MCPs, rules and skills, backing up everything first.
4. **Verifies with evidence**: the gate runs your project's test suite, build, static analysis, Semgrep and E2E, then emits `VERIFIED`, `BLOCKED` or `INCOMPLETE` — never a "looks good to me".

![One-line install](docs/images/install.png)

## The catalog

Running `setup` shows a curated checklist — everything on by default, space to opt out:

![Interactive checklist](docs/images/devbase-setup.png)

At the end, a report tells you what each piece is for:

![Final report](docs/images/finish-setup.png)

## Use cases

- **5-minute onboarding**: clone a repo, run `setup`, and your agent already knows the stack, the conventions and the project memory.
- **Real pre-commit**: `devbase gate --dir .` blocks on a type error, a security finding or broken tests — with the exact evidence.
- **Teams**: rules live in the repo (`CLAUDE.md`, `AGENTS.md`, `.cursor/rules`…), so every agent behaves the same. Or enable *local-only mode* and nothing is committed.
- **CI**: `devbase gate --json` emits the `devbase.evidence.v1` envelope (commit + working-tree hash + toolchain + checks) for PRs and pipelines.

## Install and run

```bash
# macOS / Linux
curl -fsSL https://raw.githubusercontent.com/LucianoDPerez/devbase/main/install.sh | bash

# Windows (PowerShell)
irm https://raw.githubusercontent.com/LucianoDPerez/devbase/main/install.ps1 | iex
```

SHA-256 verified against the release `checksums.txt`. From source: `go build -o devbase ./cmd/devbase` (Go 1.25+).

```bash
cd your-project
devbase setup --dir .              # full interactive bootstrap
devbase setup --dir . --yes        # no questions asked (CI, scripts)
devbase doctor                     # read-only diagnosis
devbase gate --dir .               # VERIFIED / BLOCKED / INCOMPLETE (exit 0/1/2)
```

## Optional API keys

None required. `CONTEXT7_API_KEY` raises Context7 limits (free at https://context7.com/dashboard). `devbase doctor` warns when missing.

## Status and roadmap

`v0.13.x`: interactive setup, 25 rule packs, strict gate with JSON evidence (project suites + build + Semgrep + conditional E2E), checksum installers, CI with tests + Semgrep + always-on E2E battery.

VERIFIED means: the defined checks passed on this exact snapshot (commit + working tree + toolchain on record). Not "correct" — proven and reproducible.

Next: per-stack verification profiles (PHPStan/Pint, ESLint, ruff/mypy, clippy…), Cline/Roo/Kilo/Continue adapters.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Conventional commits, no AI attribution. Security reports via GitHub Security Advisories ([SECURITY.md](SECURITY.md)).

## License

MIT — see [LICENSE](LICENSE).
