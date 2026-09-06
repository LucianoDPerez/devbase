---
level: heuristic
---

# Rust

Ownership first: borrow, don't clone. No `unwrap`/`expect` in production paths —
propagate with `?` or typed errors (thiserror/anyhow). `cargo clippy` and
`cargo fmt --check` clean. `unsafe` only with a `SAFETY:` comment. `cargo audit`
in CI.