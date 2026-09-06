---
level: heuristic
---

# Ruby on Rails

Skinny controllers, logic in models/services/jobs. Strong parameters always.
Eager-load associations to avoid N+1 (bullet in development). Reversible
migrations with `change`. Slow work goes to ActiveJob, never the request cycle.
No secrets in credentials committed without master key discipline.