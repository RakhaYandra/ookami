# Ookami — Developer Machine Doctor for Linux

Is your Linux machine ready for development? Ookami inspects your setup and reports what is healthy, what needs attention, and what is missing.

> Status (v0.7): Phase 6 done. `doctor` is functional for system/development/storage/services/network/gpu (18 checks) with weighted scoring, remediation hints, verbose details, and stable `--json` for automation.

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
Score: 99/100
1 warning, 0 critical issues
Development environment needs attention.
```

(Real output, 2026-10-09, exit 1. Values are machine-specific and will differ on your host.)

Remediation lines (`    → Suggested action: <command> (<description>[, requires sudo])`) print under warning/critical/unknown results that carry remediation data. This host shows none: the PostgreSQL no-unit path carries no remediation (only a stopped unit does). `--verbose` adds sorted `key=value` detail lines per result (e.g. `latency_ms`, `driver`).

## How scoring works

Weighted per-category score (0–100): system 20, development 30, storage 15, network 15, services 10, gpu 10. Each category starts at 100 with penalti 25/5/2 per critical/warning/unknown result (pass/info no effect, floor 0); global score is the weight-averaged sum over active categories only (renormalisasi kategori absen). GPU eksklusi bila tak ada hardware (`gpu-none` info-only excluded; single-category `check` renormalizes to its own weight). Live example above: 5 categories at 100 + services at 95 → (90×100+95×10)/100=99, status `warning`.

JSON output (`--json`) includes ordered `categories` breakdown:

```json
{"score":100,"status":"healthy","categories":[{"category":"network","status":"healthy","score":100}],"results":[{"ID":"network-interface","Category":"network","Severity":"pass","Title":"Interface","Message":"enp2s0 up","Details":{"latency_ms":0},"Remediation":null},{"ID":"network-gateway","Category":"network","Severity":"pass","Title":"Gateway","Message":"default route present","Details":{"latency_ms":0},"Remediation":null},{"ID":"network-dns","Category":"network","Severity":"pass","Title":"DNS","Message":"github.com resolves","Details":{"latency_ms":183},"Remediation":null},{"ID":"network-internet","Category":"network","Severity":"pass","Title":"Internet","Message":"connected","Details":{"latency_ms":35},"Remediation":null},{"ID":"network-github","Category":"network","Severity":"pass","Title":"GitHub","Message":"reachable","Details":{"latency_ms":187},"Remediation":null},{"ID":"network-hub","Category":"network","Severity":"pass","Title":"Docker Hub","Message":"reachable","Details":{"latency_ms":1117},"Remediation":null}]}
```

(`go run ./cmd/ookami check network --json`, 2026-10-09. `latency_ms` values are machine-specific.)

## Features

- Functional `doctor` (18 checks: 5 system, 7 development, 1 storage, 3 services, 1 network suite → 6 results, 1 gpu suite → 3 results) with grouped human output, score, and status line
- Working `--json` and `--quiet` output modes
- `check <category>` for `system`, `development`, `storage`, `services`, `network`, `gpu` (all functional)
- Category validation for `check`: `system|development|storage|network|gpu|services`
- Exit codes 0–4 for scripting
- `--no-color` flag and `NO_COLOR` env support, plus `--verbose`
- Remediation hints (`→ Suggested action`) under actionable findings

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

## Automation

`--json` schema is stable for v0.1: `{score, status, categories[], results[]}`, with `categories` ordered `system, development, storage, network, gpu, services`. Adding keys is minor; removing/renaming is breaking. `--quiet` prints only `warning|critical|unknown` titles (`⚠ <Title>`); `--verbose` adds sorted `key=value` detail lines to human output. `--json` wins over `--quiet` when both are passed.

```sh
ookami doctor --quiet; echo $?
# ⚠ PostgreSQL → exit 1 (live host, 2026-10-09)
```

## Roadmap

Planned checks and automation are tracked internally; this README documents only what v0.7 does (Phase 6 complete).

## Contributing

Issues and pull requests are welcome — please keep changes small and add tests with `go test ./...`.
