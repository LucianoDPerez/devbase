---
level: heuristic
---

# Next.js

Server Components by default; `'use client'` only at the leaves that need it.
Understand fetch caching and revalidation per route — stale data is a bug.
Route handlers validate input like any API boundary. Server-only secrets never
imported into client bundles. Use the Metadata API, not manual `<head>` tags.