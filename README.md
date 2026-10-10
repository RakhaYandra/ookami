# Ookami — Developer Machine Doctor for Linux

Is your Linux machine ready for development? Ookami inspects your setup and reports what is healthy, what needs attention, and what is missing.

> Status (v0.1): CLI skeleton. `doctor` and `check` print a stub line; full checks land in later releases.

## Example

```text
$ ookami doctor
System        ✓  OS and resources look healthy
Development   ⚠  Go toolchain needs attention
Storage       ✓  Disk space looks healthy
Network       ✓  Connectivity looks healthy
GPU           ⚠  Driver needs attention
Services      ✓  Background services look healthy
```

(Target format. v0.1 currently prints `doctor not yet implemented (Phase 1)`.)

## Features

- `doctor`, `check`, and `version` CLI skeleton (Cobra)
- Category validation for `check`: `system|development|storage|network|gpu|services`
- Exit codes 0–4 for scripting
- `--no-color` flag and `NO_COLOR` env support, plus `--verbose`

## Requirements

- Linux
- Go 1.27+

## Quick Start

```sh
make build        # produces ./bin/ookami
./bin/ookami doctor
./bin/ookami check development
./bin/ookami version
./bin/ookami check bogus   # invalid category → error, exit 4
```

## Commands & Flags

| Command | Flags | Description |
|---|---|---|
| `ookami doctor` | `--fix`, `--json`, `--quiet` | Full health check (stub in v0.1; flags accepted, no effect yet) |
| `ookami check <category>` | — | Single-category check (stub in v0.1); category must be `system`, `development`, `storage`, `network`, `gpu`, or `services` |
| `ookami version` | — | Print version (`dev` unless built with `VERSION=...`) |
| global | `--no-color`, `--verbose` | Disable color output; verbose output (`NO_COLOR` env also disables color) |

## Exit Codes

| Code | Meaning |
|---|---|
| 0 | Healthy |
| 1 | Warning or unknown finding |
| 2 | Critical finding |
| 3 | Execution error |
| 4 | Invalid arguments or CLI usage error |

## Roadmap

Planned checks and automation are tracked internally; this README documents only what v0.1 does.

## Contributing

Issues and pull requests are welcome — please keep changes small and add tests with `go test ./...`.
