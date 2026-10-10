# Ookami Architecture

## Overview

Ookami is a local-first system health checker written in Go. It runs ordered checks from a central registry, aggregates their severities into a single score and status, and renders human/JSON output with stable exit codes. All execution stays on-machine using the standard library; the network suite is the sole exception (TCP dial + HTTPS HEAD probes, no telemetry).

## Directory layout

```
ookami/
├── cmd/ookami/main.go        # entrypoint, calls cli.Execute()
├── internal/
│   ├── cli/                  # cobra commands: root, doctor, check, version
│   ├── checks/               # concrete checks: OS/kernel/CPU/memory/uptime, toolchains, filesystem, systemd services, network suite, gpu suite
│   ├── doctor/               # DefaultChecks(), FilterByCategory(), concurrent RunAll()
│   ├── config/               # Config, Default(), Load(), Validate()
│   ├── model/                # Check, Result, Severity, Category, Remediation
│   ├── output/               # RenderHuman/JSON/Quiet + ExitCode
│   ├── registry/             # Register(), Ordered(), ByCategory()
│   ├── runner/               # Runner iface, OSRunner, MockRunner
│   └── scoring/              # Score()/Breakdown() weighted
├── Makefile
├── go.mod / go.sum
└── bin/                      # build output (gitignored)
```

Tests (`*_test.go`) sit next to each package.

## Packages

- **model**: core types. `Check` interface (`Metadata()` + `Run(ctx)`) returns a `Result`; `Severity` is `pass/info/warning/critical/unknown`; `CheckMetadata` carries `Category` and `Optional` flag.
- **runner**: command execution abstraction. `OSRunner` runs binaries with a default 5s timeout; `MockRunner` replays scripted handlers and records calls for tests.
- **checks + doctor**: concrete checks live in `checks` (OS, kernel, CPU, memory, uptime; git/go/node/python/php/docker/daemon; filesystem; PostgreSQL/Redis/MySQL services; `NetworkSuite`; `GPUSuite`), each returning `[]Result` via a `Runner`; `doctor.DefaultChecks()` orders 18 checks system → development → storage → services → network → gpu, `FilterByCategory()` subsets them for `check <category>`, and `RunAll()` executes them concurrently with per-check timeouts (45s, covers the suite worst-case) while preserving input order (a panicking or empty check yields one `unknown` result).
- **services**: one generic `SystemdServiceCheck` backs all three service checks (PostgreSQL, Redis, MySQL), each configured with candidate `UnitNames` plus fallback `Binaries`. It probes `systemctl show -p LoadState -p ActiveState <unit>` and treats `LoadState=loaded` + `ActiveState=active` as pass (running) and loaded-but-inactive as warning (stopped); when no unit is loaded it falls back to binary presence via `LookPath` (installed without a unit → warning, neither → info). Remediation is data-only: a stopped unit carries a `Remediation{Command: "systemctl", Args: ["start", unit], Safe: true, RequiresSudo: true}` payload, but nothing auto-applies — `--fix` still prints the Phase 7 notice.
- **network**: one `NetworkSuite` check yields 6 `network-*` results (interface, gateway, DNS, internet, GitHub, Docker Hub) as a fail-fast cascade — first failure stops the chain, remaining layers report `info/skipped (<reason>)`. Interface/gateway fail as `critical` (no local path), DNS the same (no name resolution), while internet/GitHub/Hub fail as `warning` (local stack works, remote unreachable). All five hooks are injectable (`Ifaces`, `RouteReader`, `Resolve`, `Dial`, `Head`) so tests run offline with fakes; live defaults are stdlib-only (`net.Interfaces`, `/proc/net/route` default-route scan, `LookupIPAddr(github.com)`, dial `1.1.1.1:443`, `HEAD https://github.com` + `HEAD https://registry-1.docker.io/v2/` with 2xx/3xx pass, 401 accepted for the Hub). Per-layer timeout comes from `Config.NetworkTimeoutSec` (default 5s); results carry only `latency_ms` in `Details` — no IPs or addresses are printed.
- **gpu**: one `GPUSuite` check (`ID gpu-suite`, `Optional`) yields per-GPU results plus CUDA. Detection is `lspci -nn` (class `[0300]`/`[0302]`, vendor from PCI ID `10de/1002/8086`) with sysfs (`/sys/bus/pci/devices`, class `0x03xx`) fallback when lspci yields nothing, plus a best-effort vendor match back to sysfs for the bound driver when lspci supplied the names. NVIDIA driver versions come from `nvidia-smi --query-gpu=name,driver_version --format=csv,noheader` (missing version → `warning/driver unavailable`); AMD/Intel report the bound driver with best-effort `glxinfo` GL version appended (`driver, GL x.y`, driver-only, or GL-only). CUDA (`nvcc --version`) is info-only and appears only when at least one NVIDIA GPU passes — `info/<version>` or `info/not installed`, never affecting the score. All four hooks (`Lspci/SysFS/GlxInfo/NvccVersion`) are injectable so tests run offline with fakes. Titles carry no category prefix (`NVIDIA <name>`, `AMD <name>`, `CUDA`); the `GPU` group header comes from the output renderer.
- **registry**: global check catalog. `Register()` appends, `Ordered()` returns a copy in registration order, `ByCategory()` filters by category.
- **scoring**: weighted aggregation. `Breakdown(results)` scores each active category from 100 minus 25/critical, 5/warning, 2/unknown (pass/info no effect, floor 0) with weights system 20 / development 30 / storage 15 / network 15 / services 10 / gpu 10; global score is the weight-averaged sum over active categories only (absent categories renormalized out, `gpu-none` no-hardware result excluded, integer division). `Score()` returns global score + worst-severity status (`healthy/warning/critical`, unknown counts as warning). Live `doctor` on the reference host is 99 (`(90×100+95×10)/100`, services warning 95, rest 100); check count unchanged at 18.
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
- **bounded execution**: every command has a 5s default timeout (`OSRunner.Timeout`); network layers reuse `Config.NetworkTimeoutSec` (default 5s) and `doctor` allows 45s per check to cover the suite worst-case; an existing context deadline is respected.
- **optional ≠ critical**: `CheckMetadata.Optional` marks non-essential checks so missing tooling degrades to non-critical results instead of failing the run.
- **no telemetry / no PII**: everything runs locally, nothing is uploaded; `Result.Details` carries only machine facts needed for display (`latency_ms` for network, no IPs or resolved addresses).
- **deterministic ordered output**: registry preserves registration order, renderers iterate the result slice as-is, JSON normalizes nil to `[]`.
