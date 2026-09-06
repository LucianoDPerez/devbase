---
name: dev-review
description: Structured pre-commit code review. Use when reviewing a diff, before committing, or when asked to review code.
---

# Dev Review

Review the current diff (`git diff HEAD`) with this order:

1. **Correctness**: does it do what it claims? Trace callers of changed functions.
2. **Tests**: behavior changes carry tests asserting the visible contract.
3. **Security**: no secrets, no injection (parameterize queries), auth checked server-side.
4. **Rules**: naming, single responsibility, no dead code, no magic numbers.
5. **Evidence**: run `devbase gate --dir .` — BLOCKED means the review fails.

Report findings only, grouped CRITICAL / WARNING / SUGGESTION, with file and line.
Do not rewrite code unasked; propose the fix.
