# Secrets

No hardcoded API keys, tokens, or credentials in code or committed examples.
Backend verifies auth on every request; cookies use httpOnly + secure + sameSite.

Level: deterministic. Enforced by secret scanning, push protection, and Semgrep
secrets rules. Gate merges on ERROR from day one; escalate WARNING later.
