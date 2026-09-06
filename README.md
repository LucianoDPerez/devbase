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

`v0.2.0`: `doctor`, `init` (+`--wire`) y `gate` funcionales.

## Requisitos

- Git (siempre), `gh` (integración GitHub vía CLI + skill, no MCP)
- Opcional según el proyecto: `engram`, `codebase-memory-mcp`, `semgrep`

## Instalación

```bash
# macOS / Linux
curl -fsSL https://raw.githubusercontent.com/LucianoDPerez/devbase/main/install.sh | bash
```

```powershell
# Windows (PowerShell)
irm https://raw.githubusercontent.com/LucianoDPerez/devbase/main/install.ps1 | iex
```

Verifican SHA-256 contra `checksums.txt` del release. Desde fuente: `go build -o devbase ./cmd/devbase` (Go 1.23+).

## Uso

```bash
# Qué IDEs tenés configurados y qué dependencias faltan (no escribe nada)
devbase doctor

# Detectar stack y escribir reglas contextuales en .devbase/
devbase init --dir /ruta/al/proyecto

# Además ejecuta el setup de herramientas externas en los IDEs detectados
devbase init --dir /ruta/al/proyecto --wire

# Verificación con evidencia: build por stack + Semgrep ERROR-only, atado al SHA
devbase gate --dir /ruta/al/proyecto
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
- [x] `init`: reglas contextuales en `.devbase/` + `--wire` (setup externo por IDE)
- [x] `gate`: VERIFIED / BLOCKED con SHA exacto (build + Semgrep ERROR-only)
- [x] Instalador `install.sh` / `install.ps1` + releases con checksums
- [ ] Adapters fase 2: Cline, Roo Code, Kilo Code, Continue.dev
- [ ] `gate`: tests del proyecto + PHPStan/ESLint por stack
