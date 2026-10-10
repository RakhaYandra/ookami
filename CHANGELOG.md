# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-10-09

### Added

- Initial stable release (Phase 0–7 history retained below as 0.2.0–0.8.0):
  - `doctor`: 18 checks across 6 categories (system, development, storage, network, gpu, services)
  - Weighted scoring with per-category breakdown and renormalization for absent categories
  - Output modes: grouped human, `--json` (ordered categories), `--quiet`, `--verbose` (`--json` wins over `--quiet`)
  - Exit codes 0-4 (ok / warning / critical / usage-config error)
  - `doctor --fix` allowlist engine: only `systemctl start|restart <unit>`, TTY-gated `[y/N]` confirmation, per-item result plus rerun
  - `check <category>` per-category runs with remediation hints (`→ Suggested action:`)
  - Docs: README, CONTRIBUTING, docs/ (architecture, checks catalogue, scoring, exit codes, output contract)

## [0.8.0] - 2026-10-09

### Added

- Phase 7 fix engine:
  - New `internal/fixer`: `Plan()` allows only `systemctl start|restart <unit>` (exactly 2 args, unit regex `^[a-zA-Z0-9@._:\-]+$`) from warning/critical results with remediation data; `Confirm()` prompts `Execute? [y/N]` on stdout, TTY-gated (non-TTY auto-declines without reading), `y/yes` only; `Apply()` runs via system `sudo` when non-root (direct when root), per-item `✓ fixed` / `✗ <first error line>` / `skipped.` lines, continues past failures
  - `doctor --fix` now executes: renders, applies fixes, prints `fix: N fixed, N failed, N skipped.` to stderr, re-runs all checks on a fresh context and renders again; `--fix` forces human output even with `--json`/`--quiet` (`info: --fix renders human output`); no `--yes` flag
  - Live host unchanged: `echo n | ookami doctor --fix` → `info: no fixable items.`, Score 99/100, 1 warning, exit 1 (PostgreSQL no-unit path carries nil remediation, prompt never appears)

## [0.7.0] - 2026-10-09

### Added

- Phase 6 automation/output:
  - Remediation hints: `→ Suggested action: <cmd> (<desc>[, requires sudo])` under warning/critical/unknown results with remediation data (`suggestedActionLine` in `internal/output`); live host shows none (PostgreSQL no-unit path carries nil, only a stopped unit does)
  - `RenderHumanVerbose` (`--verbose` on `doctor` + `check`): sorted `key=value` detail lines per result; `--json` wins over `--quiet`, `doctor` Long documents the precedence, `--fix` usage notes the Phase 7 notice
  - Single category order: `output` now aliases `scoring.CategoryOrder` (`system, development, storage, network, gpu, services`); human grouping and JSON `categories` follow the same order
  - Live `doctor` unchanged: Score 99/100, 1 warning, exit 1; exit matrix via built binary: `check network --quiet` → 0, `doctor --quiet` → 1, `check bogus` → 4

## [0.6.0] - 2026-10-09

### Added

- Phase 5 scoring:
  - Weighted `Breakdown()` in `internal/scoring` (weights system 20 / development 30 / storage 15 / network 15 / services 10 / gpu 10; penalties 25 critical / 5 warning / 2 unknown, floor 0; global = weight-averaged over active categories only, integer division)
  - Renormalization: absent categories excluded from divisor; `gpu-none` (no hardware) excluded alone and with others; `check <category>` scores 100× its own weight
  - `--json` (`doctor` + `check`) now emits ordered `categories: [{category,status,score}]` via `RenderJSONBreakdown` (follows `scoring.CategoryOrder`); legacy `RenderJSON` kept for compat
  - Live `doctor` score 95 → 99 on same host (5 categories at 100 + services at 95: `(90×100+95×10)/100=99`); 18 checks total unchanged

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
