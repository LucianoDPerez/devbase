# DevBase

Estándar operativo para agentes de desarrollo: **contexto + memoria + buenas prácticas + ejecución + verificación + evidencia.**

DevBase no es "otro pack de MCPs". Es el CLI que deja cualquier repo listo para trabajar con agentes (OpenCode, Claude Code, Cursor, VS Code + Copilot, Windsurf, Codex): detecta tu stack, carga solo las reglas que aplican, conecta los IDEs que tengas instalados y verifica con evidencia reproducible.

```
AGENT
  ├── Context7 · Codebase Memory · Engram · Skills + Rules
  ▼
CODING
  ▼
VERIFICATION (tests · lint · build · Semgrep · Playwright)
  ▼
EVIDENCE GATE ──► VERIFIED ✅ / BLOCKED ❌
```

## Estado

MVP temprano (`v0.1.0`): `doctor` funcional, `init` con detección de stack, `gate` en construcción.

## Requisitos

- Go 1.23+
- Git (siempre), `gh` (integración GitHub vía CLI + skill, no MCP)
- Opcional según el proyecto: `engram`, `codebase-memory-mcp`, `semgrep`

## Instalación (desde fuente, por ahora)

```bash
git clone https://github.com/LucianoDPerez/devbase.git
cd devbase
go build -o devbase ./cmd/devbase
```

> Binarios `curl | bash` / `install.ps1` vía GoReleaser: en roadmap.

## Uso

```bash
# Qué IDEs tenés configurados y qué dependencias faltan (no escribe nada)
devbase doctor

# Detectar el stack del proyecto (core → específico → security)
devbase init --dir /ruta/al/proyecto

# Verificación con evidencia (en construcción: tests + lint + build + Semgrep ERROR-only)
devbase gate
```

Ejemplo real:

```
$ devbase doctor
IDEs:
  opencode       ✓ config found
  claude-code    ✓ config found
  cursor         ✓ config found
  vscode-copilot ✗ not detected
  windsurf       ✗ not detected
  codex          ✓ config found
Dependencies:
  ✓ git
  ✓ gh
  ✓ engram
  ✓ codebase-memory-mcp
  ✗ semgrep              missing
```

## Cómo ahorra tokens

DevBase nunca carga "todas las reglas". Carga por niveles (metadata → skill → referencias, solo lo activado) y por stack detectado (un proyecto Laravel no recibe reglas de React). Medido en la industria: 60–96% menos tokens por sesión según el tipo de tarea.

## Roadmap

- [x] `doctor`: detección de IDEs + dependencias
- [x] Detección de stack (`core → php/js/go/python → security`)
- [ ] `init`: escritura de reglas contextuales + wiring MCP por IDE
- [ ] `gate`: VERIFIED / BLOCKED con SHA exacto
- [ ] Instalador `install.sh` / `install.ps1` + Homebrew / winget
- [ ] Adapters fase 2: Cline, Roo Code, Kilo Code, Continue.dev
