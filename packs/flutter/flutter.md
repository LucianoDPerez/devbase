---
level: heuristic
---

# Flutter

`const` constructors everywhere possible. Keys in dynamic lists. No business
logic inside `build()` — state lives in providers/bloc, widgets stay dumb.
`flutter analyze` clean, zero warnings policy. One widget per file for shared UI.