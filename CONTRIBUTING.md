# Contributing to Ookami

## Prerequisites

- Linux
- Go 1.27+ (`go version`)

## Build, Test, Gate

```sh
make build          # ./bin/ookami
go test -race ./...
go vet ./...
make lint           # golangci-lint run ./...
```

Note: `make test` runs `go test ./...` without `-race`. Use `go test -race ./...` for the full gate.

All four must be green before opening a PR or moving to the next phase.

## Commits

Conventional Commits: `feat:`, `fix:`, `refactor:`, `docs:` (+ scope optional, e.g. `feat(checks): ...`). One logical change per commit.

## Rules

1. **No phase-skipping (Rule 1).** Finish the current phase fully green before starting the next.
2. **Stdlib-first.** No new dependency without justification in the PR.
3. **Each check needs all five:** implementation + test + severity + diagnosis + docs. Missing one = incomplete.
4. **No `sh -c`.** Invoke binaries directly (`exec.Command(name, args...)`), never via shell.
5. **Gate green before next phase:** `make build`, `go test -race ./...`, `go vet ./...`, `make lint`.
