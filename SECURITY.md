# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.15.x  | ✅                 |
| < 0.15  | ❌ (upgrade)       |
| main    | ✅ (pre-release)   |

## Reporting a Vulnerability

Use **GitHub Security Advisories** (private) on this repository:
`Security` tab → `Report a vulnerability`.

Do not open public issues for vulnerabilities. Include steps to reproduce,
affected version/commit, and impact. We aim to acknowledge within 72 hours.

## Scope Notes

DevBase reads your codebase and writes agent configuration files by design.
It never exfiltrates code: verification runs locally, and findings stay on
your machine unless you push them yourself.
