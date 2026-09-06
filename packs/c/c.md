---
level: heuristic
---

# C

`-Wall -Wextra -Werror`, always. Check every return that can fail (malloc, I/O).
No `strcpy`/`sprintf` — bounded variants only. No undefined behavior to "save"
a branch. AddressSanitizer + UBSan in CI debug builds.