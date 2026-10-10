# Ookami Architecture

## Overview

Ookami is a local-first system health checker written in Go. It runs ordered checks from a central registry, aggregates their severities into a single score and status, and renders human/JSON output with stable exit codes. All execution stays on-machine using the standard library, with no network calls or telemetry.

## Directory layout

```
ookami/
├── cmd/ookami/main.go        # entrypoint, calls cli.Execute()
├── internal/
│   ├── cli/                  # cobra commands: root, doctor, check, version
│   ├── checks/               # concrete checks: OS/kernel/CPU/memory/uptime, toolchains, filesystem
│   ├── doctor/               # DefaultChecks(), FilterByCategory(), concurrent RunAll()
│   ├── config/               # Config, Default(), Load(), Validate()
│   ├── model/                # Check, Result, Severity, Category, Remediation
│   ├── output/               # RenderHuman/JSON/Quiet + ExitCode
│   ├── registry/             # Register(), Ordered(), ByCategory()
│   ├── runner/               # Runner iface, OSRunner, MockRunner
│   └── scoring/              # Score() stub
├── Makefile
├── go.mod / go.sum
└── bin/                      # build output (gitignored)
```

Tests (`*_test.go`) sit next to each package.

## Packages

- **model**: core types. `Check` interface (`Metadata()` + `Run(ctx)`) returns a `Result`; `Severity` is `pass/info/warning/critical/unknown`; `CheckMetadata` carries `Category` and `Optional` flag.
- **runner**: command execution abstraction. `OSRunner` runs binaries with a default 5s timeout; `MockRunner` replays scripted handlers and records calls for tests.
- **checks + doctor**: concrete checks live in `checks` (OS, kernel, CPU, memory, uptime; git/go/node/python/php/docker/daemon; filesystem), each returning `[]Result` via a `Runner`; `doctor.DefaultChecks()` orders 13 checks system → development → storage, `FilterByCategory()` subsets them for `check <category>`, and `RunAll()` executes them concurrently with per-check timeouts while preserving input order (a panicking or empty check yields one `unknown` result).
- **registry**: global check catalog. `Register()` appends, `Ordered()` returns a copy in registration order, `ByCategory()` filters by category.
- **scoring**: aggregation stub. `Score(results)` folds severities into a 0–100 score plus a `healthy/warning/critical` status string.
- **output**: rendering + exit codes. Human/JSON/quiet renderers; `ExitCode()`: 0 ok, 1 warning/unknown, 2 critical, 3 execution error, 4 invalid args.
- **config**: runtime thresholds. `Default()` supplies network/storage/memory settings, `Load("")` falls back to defaults, `Validate()` enforces `0 < Warn < Crit <= 100`.
- **cli** (cobra): user entry. Root command wires `doctor` (`--fix/--json/--quiet`), `check <category>`, and `version`; global `--no-color/--verbose` flags; errors exit non-zero.

## Flow

```
CLI (cobra) → registry.Ordered()/ByCategory()
            → Check.Run(ctx) via Runner
            → scoring.Score(results)
            → output.Render* + ExitCode()
```

`doctor` runs the full ordered set; `check <category>` runs one category subset.

## Principles

- **stdlib-first**: prefer `os/exec`, `context`, `encoding/json`; external deps only for CLI (`cobra`) and terminal styling.
- **no `sh -c`**: invoke binaries directly with argv, never through a shell:
  ```go
  cmd := exec.CommandContext(ctx, name, args...)
  out, err := cmd.Output()
  ```
- **bounded execution**: every command has a 5s default timeout (`OSRunner.Timeout`); an existing context deadline is respected.
- **optional ≠ critical**: `CheckMetadata.Optional` marks non-essential checks so missing tooling degrades to non-critical results instead of failing the run.
- **no telemetry / no PII**: everything runs locally, nothing is uploaded; `Result.Details` carries only machine facts needed for display.
- **deterministic ordered output**: registry preserves registration order, renderers iterate the result slice as-is, JSON normalizes nil to `[]`.
