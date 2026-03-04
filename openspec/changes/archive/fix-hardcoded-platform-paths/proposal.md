## Why

Two paths in `cmd/dashcap/main.go` are hardcoded to Linux-only values:

1. **Lock directory** (line 112): `filepath.Join("/run/dashcap", ...)` — DESIGN.md §8.2 specifies:
   - Linux: `/run/dashcap/`
   - Windows: `C:\ProgramData\dashcap\locks\`
   - macOS: `/var/run/dashcap/`

2. **Data directory default** (line 95): `filepath.Join("/var/lib/dashcap", ...)` — DESIGN.md §9 specifies:
   - Linux/macOS: `/var/lib/dashcap/`
   - Windows: `C:\ProgramData\dashcap\`

Running dashcap on Windows would attempt to create `/run/dashcap/` and `/var/lib/dashcap/` — paths that don't exist and are invalid on Windows. This contradicts the Phase 1 goal of supporting both Linux and Windows (DESIGN.md §3.1, §12 Phase 1).

## What Changes

- Move platform-specific default paths into the `storage` package (or a new `platform` subpackage) behind build tags, matching the existing pattern used for `DiskOps`
- Expose functions like `DefaultDataDir()` and `DefaultLockDir()` that return the correct path per platform
- Update `main.go` to call these functions instead of hardcoding paths

## Capabilities

### Modified Capabilities

- `main`: Lock directory and data directory defaults are platform-aware, following DESIGN.md §9 platform abstraction table

### New Capabilities

- `storage` (or `platform`): `DefaultDataDir()` and `DefaultLockDir()` functions with per-platform implementations via build tags

## Impact

- `internal/storage/disk_unix.go`: Add `DefaultDataDir()` → `/var/lib/dashcap/`, `DefaultLockDir()` → `/run/dashcap/`
- `internal/storage/disk_windows.go`: Add `DefaultDataDir()` → `C:\ProgramData\dashcap\`, `DefaultLockDir()` → `C:\ProgramData\dashcap\locks\`
- `internal/storage/storage.go`: Add `DefaultDataDir()` and `DefaultLockDir()` to the `DiskOps` interface, or expose as package-level functions
- `cmd/dashcap/main.go`: Replace hardcoded `/run/dashcap` and `/var/lib/dashcap` with calls to the new platform functions
- Optionally add a separate `prealloc_darwin.go`-style split for macOS (`/var/run/dashcap/` vs `/run/dashcap/`)
