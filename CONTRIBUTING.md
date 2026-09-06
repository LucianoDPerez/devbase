# Contributing to DevBase

## Workflow

1. Open an issue first for anything beyond a trivial fix.
2. Branch from `main`: `feat/...`, `fix/...`, `chore/...`, `docs/...`.
3. Open a PR against `main`. CI must pass. Keep PRs small and reviewable.

## Commits

Conventional commits only (`feat:`, `fix:`, `chore:`, `docs:` ...).
No `Co-Authored-By` trailers, no AI attribution in commits or PRs.

## Code

- `go generate ./...`, `gofmt -l .` (empty), `go vet ./...`, `go build ./...`
  and `go test ./...` must pass before pushing.
- New stacks need a pack AND a `detect` matrix case; the
  `TestDetectStacksHavePacks` meta-test enforces the pairing.
- One row per IDE in `internal/adapters`; core never hardcodes IDE paths.
- External tools self-configure via their own installers; DevBase invokes
  them instead of duplicating their logic.
- Fail open with a clear message when an adapter can't apply; never block
  the install because of one IDE.
