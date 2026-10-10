# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.5.0] - 2026-10-09

### Added

- Phase 4 GPU:
  - `GPUSuite` in `internal/checks` (1 check → 3 results: NVIDIA driver via `nvidia-smi`, AMD via sysfs driver, CUDA info-only), wired through `doctor.DefaultChecks()` (18 checks total)
  - Detection via `lspci -nn` with sysfs (`/sys/bus/pci/devices`) fallback; `glxinfo` best-effort only
  - `check gpu` functional; no future-phase notices remain

## [0.4.0] - 2026-10-08

### Added

- Phase 3 network:
  - `NetworkSuite` in `internal/checks` (1 check → 6 results: interface, gateway, DNS, internet, GitHub, Docker Hub), wired through `doctor.DefaultChecks()` (17 checks total)
  - `check network` functional; only `gpu` still prints a future-phase notice
  - `doctor` per-check timeout raised to 45s (covers suite worst-case: 6 layers × 5s sequential)

## [0.3.0] - 2026-10-08

### Added

- Phase 2 services:
  - 3 functional checks (PostgreSQL, Redis, MySQL) via generic `SystemdServiceCheck` in `internal/checks`, wired through `doctor.DefaultChecks()` (16 checks total)
  - `check services` functional; `network`/`gpu` still print a future-phase notice

## [0.2.0] - 2026-10-08

### Added

- Phase 1 core doctor:
  - 13 functional checks (5 system, 7 development, 1 storage) via `internal/checks`, wired through `internal/doctor`
  - Grouped `RenderHuman` output (category groups, severity symbols, score + warning/critical footer + status line)

## [0.1.0] - 2026-10-08

### Added

- Phase 0 foundation:
  - CLI skeleton (`doctor`, `check`, `version`) on Cobra
  - Core model, runner, and check registry
  - Scoring stub, exit codes (0-4), config, output stub

> Note: stubs only, not yet functional. `doctor` and `check` print a placeholder line; real checks land in later releases.
