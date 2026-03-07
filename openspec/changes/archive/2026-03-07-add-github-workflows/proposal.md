## Why

The project has a provisional Forgejo CI at `.forgejo/workflows/ci.yml` covering lint, test, and build. Moving to GitHub Actions is an opportunity to set up CI/CD properly: test against multiple libpcap versions (the Debian/Fedora matrix in Forgejo was a proxy for libpcap 0.8 vs 1.x compatibility), add Windows cross-compilation, and automate GitHub Releases.

## What Changes

- Add a **CI workflow** (`.github/workflows/ci.yml`) — lint, test with race detector and coverage on push/PR to main. Test matrix covers multiple libpcap versions (not tied to specific distros).
- Add a **Release workflow** (`.github/workflows/release.yml`) — triggered on `v*` tags. Build matrix produces Linux binaries tested against different libpcap versions, plus Windows binaries (cross-compiled with Npcap SDK or MinGW). Creates a GitHub Release with all binaries and checksums.
- Add `.github/dependabot.yml` to keep Go modules and Actions versions up to date.
- The existing `.forgejo/workflows/ci.yml` remains as-is (can be removed later if desired).

## Capabilities

### New Capabilities
- `ci-pipeline`: GitHub Actions CI workflow — lint, test (race + coverage) against multiple libpcap versions on push/PR.
- `release-pipeline`: GitHub Actions release workflow — build matrix for Linux (libpcap variants) and Windows, create GitHub Release with binaries and checksums on version tags.

### Modified Capabilities
<!-- None — existing Forgejo CI is untouched. -->

## Impact

- New files: `.github/workflows/ci.yml`, `.github/workflows/release.yml`, `.github/dependabot.yml`.
- Reuses existing Makefile targets and build logic.
- Linux builds require `libpcap-dev`; Windows builds require Npcap SDK or WinPcap headers for cross-compilation.
- No changes to application code, APIs, or existing Forgejo workflows.
