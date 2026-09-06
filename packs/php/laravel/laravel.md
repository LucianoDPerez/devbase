---
level: heuristic
---

# Laravel

Fat models, thin controllers is a smell — push logic to actions/services.
Validate with FormRequests, authorize with Policies/Gates, never trust
frontend-only checks. Eloquent: eager-load to avoid N+1.