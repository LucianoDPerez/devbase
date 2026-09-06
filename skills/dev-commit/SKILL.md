---
name: dev-commit
description: Conventional commit workflow. Use when committing, pushing, or preparing a PR.
---

# Dev Commit

1. **Gate first**: `devbase gate --dir .` must say VERIFIED. BLOCKED stops here.
2. **Stage with intent**: review `git status` and `git diff`; stage only what belongs together.
3. **Message**: conventional commits (`feat:`, `fix:`, `chore:`, `docs:`, `test:`), imperative, no AI attribution, no `Co-Authored-By`.
4. **Push/PR**: push to a branch, open a PR with what/why/evidence. Never force-push to `main`.
