# Ookami — Developer Machine Doctor for Linux

Is your Linux machine ready for development? Ookami inspects your setup and reports what is healthy, what needs attention, and what is missing.

> Status (v0.5): Phase 4 done. `doctor` is functional for system/development/storage/services/network/gpu (18 checks).

## Example

```text
$ ookami doctor
OOKAMI — Developer Machine Doctor

System
  ✓ OS Omarchy
  ✓ Kernel 7.2.5-3-omarchy x86_64
  ✓ CPU AMD Ryzen 7 7840HS w/ Radeon 780M Graphics (8 cores/16 threads, 61°C)
  ✓ Memory 8.0 / 15 GB available (45% used)
  ℹ Uptime 2m
Development
  ✓ Git 2.55.0
  ✓ Go 1.27.1
  ✓ Node 26.8.1 (npm 11.19.0)
  ✓ Python 3.14.8
  ✓ PHP 8.5.10
  ✓ Docker 29.7.2
  ✓ Docker Daemon running
Storage
  ✓ / 29% used
  ✓ /home 29% used
  ✓ /var 29% used
Network
  ✓ Interface enp2s0 up
  ✓ Gateway default route present
  ✓ DNS github.com resolves
  ✓ Internet connected
  ✓ GitHub reachable
  ✓ Docker Hub reachable
GPU
  ✓ NVIDIA GeForce RTX 4050 Max-Q / Mobile driver 610.57.04
  ✓ AMD Phoenix1 amdgpu
  ℹ CUDA not installed
Services
  ⚠ PostgreSQL installed, no systemd unit
  ℹ Redis not installed
  ℹ MySQL not installed
----------------------------------------
Score: 95/100
1 warning, 0 critical issues
Development environment needs attention.
```

(Real output, 2026-10-08, exit 1. Values are machine-specific and will differ on your host.)

## Features

- Functional `doctor` (18 checks: 5 system, 7 development, 1 storage, 3 services, 1 network suite → 6 results, 1 gpu suite → 3 results) with grouped human output, score, and status line
- Working `--json` and `--quiet` output modes
- `check <category>` for `system`, `development`, `storage`, `services`, `network`, `gpu` (all functional)
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
| `ookami doctor` | `--fix`, `--json`, `--quiet` | Full health check (system/development/storage/services/network/gpu); `--json`/`--quiet` switch output, `--fix` prints a Phase 7 notice |
| `ookami check <category>` | — | Single-category check; `system`, `development`, `storage`, `services`, `network`, `gpu` all functional |
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

Planned checks and automation are tracked internally; this README documents only what v0.5 does (Phase 4 complete).

## Contributing

Issues and pull requests are welcome — please keep changes small and add tests with `go test ./...`.
