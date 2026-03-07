## Context

The project builds a Go binary (`dashcap`) with CGO_ENABLED=1 due to the `gopacket` dependency on libpcap. A provisional Forgejo CI exists at `.forgejo/workflows/ci.yml`. The goal is to set up proper GitHub Actions CI/CD that will later be adapted back to Forgejo.

The Makefile provides all build targets: `lint` (golangci-lint), `test` (race + count=1), `cover` (race + coverprofile), `build`, and `cross` (linux-amd64, windows-amd64, darwin-arm64).

## Goals / Non-Goals

**Goals:**
- CI workflow: lint, test (race + coverage) on every push/PR to main
- Release workflow: build Linux amd64 + Windows amd64 binaries on version tags, publish as GitHub Release with checksums
- Dependabot config for Go modules and Actions versions
- Clean, adaptable workflow files that can later be ported back to Forgejo

**Non-Goals:**
- macOS/ARM builds in CI (cross target exists in Makefile but not prioritized for CI)
- Multi-libpcap-version testing (deferred — all current Debian releases ship 1.10.x; a previous compatibility issue needs separate investigation)
- Windows CI testing (build-only — no libpcap/Npcap available on GitHub-hosted Windows runners for test execution)
- Replacing or modifying the Forgejo CI
- Container image publishing

## Decisions

### 1. Single Linux container image for CI

Use `ubuntu-latest` runner with `libpcap-dev` installed via apt. No multi-distro matrix — all current Debian releases ship libpcap 1.10.x, so a matrix adds cost without value. If libpcap version testing becomes needed, it can be added later by building libpcap from source.

### 2. Windows cross-compilation from Linux

Cross-compile for Windows using `x86_64-w64-mingw32-gcc` and WinPcap/Npcap developer headers on the Linux runner. This avoids Windows runners which cost 2x Actions minutes.

**Alternative considered:** Native Windows runner with Npcap SDK. Rejected due to Npcap license restrictions and higher minute cost.

### 3. Reuse Makefile targets

The CI workflow calls `make lint`, `make cover`, and `make build` directly. The release workflow uses the same LDFLAGS pattern from the Makefile's `cross` target. This keeps the workflows thin and the build logic in one place.

### 4. `softprops/action-gh-release` for releases

Standard community action for creating GitHub Releases from tag-triggered workflows. Handles idempotent release creation and multi-file artifact upload.

### 5. Separate CI and Release workflows

`ci.yml` runs on push/PR — fast feedback (lint + test + build verify). `release.yml` runs only on `v*` tags — heavier build with cross-compilation and artifact upload. Keeps PR feedback fast.

## Risks / Trade-offs

- **[Windows cross-compile may have CGO edge cases]** → WinPcap headers are well-tested with MinGW. Can add a native Windows runner later if issues arise.
- **[No Windows test execution]** → Build-only for now. Functional testing on Windows remains manual.
- **[libpcap compatibility issue unresolved]** → A previous binary portability issue between distros is not yet understood. Deferred for separate investigation — may be a dynamic linking / SONAME issue rather than an API incompatibility.
