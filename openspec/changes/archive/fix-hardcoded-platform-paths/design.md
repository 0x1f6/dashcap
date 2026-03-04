## Context

`cmd/dashcap/main.go` hardcodes two Linux-only paths (lines 95, 112) that make dashcap fail on Windows. DESIGN.md §9 defines per-platform defaults for both data and lock directories. The `internal/storage` package already uses build tags (`disk_unix.go`, `disk_windows.go`) for platform-specific `DiskOps` implementations — the same pattern should be reused for default paths.

## Goals / Non-Goals

**Goals:**

- Data directory and lock directory defaults follow DESIGN.md §8.2 and §9 platform tables
- Windows builds use `C:\ProgramData\dashcap\` paths instead of Unix paths
- Reuse the existing build-tag split in `internal/storage`

**Non-Goals:**

- macOS-specific lock path (`/var/run/dashcap/`) — macOS is not a Phase 1 target (DESIGN.md §12). The unix build tag covers both Linux and macOS; the Linux path `/run/dashcap/` will be used on macOS for now. A darwin-specific split can be added later.
- Making paths user-configurable beyond the existing `--data-dir` flag
- Changing the `DiskOps` interface

## Decisions

### Package-level functions, not interface methods

**Decision:** Expose `DefaultDataDir()` and `DefaultLockDir()` as package-level functions in `internal/storage`, not as methods on the `DiskOps` interface.

**Rationale:** Default paths are static constants that don't require disk I/O or instance state. Adding them to `DiskOps` would force every implementation (including test mocks) to carry methods that just return a string.

**Trade-off:** If a future platform needs runtime detection (e.g. reading an env var), the function can be changed without breaking the interface contract.

### Build-tag files mirror existing pattern

**Decision:** Add the functions to the existing `disk_unix.go` and `disk_windows.go` files rather than creating new files.

**Rationale:** The functions are small (two one-liners each). Separate files would add noise without adding clarity. The build tags are already in place.

### No darwin-specific split in Phase 1

**Decision:** macOS uses the same paths as Linux (`/run/dashcap/`, `/var/lib/dashcap/`) via the `unix` build tag.

**Rationale:** Phase 1 targets Linux and Windows only. The macOS lock path difference (`/var/run/` vs `/run/`) can be addressed when macOS becomes a target.

## Risks / Trade-offs

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| macOS users hit `/run/dashcap/` which may not exist | Low — not a Phase 1 target | Document as known limitation; trivial to add `disk_darwin.go` later |
| Windows `C:\ProgramData\` requires elevated permissions | Medium | Same as current design intent; dashcap on Windows already needs admin for packet capture |

## Migration Plan

1. Add `DefaultDataDir()` and `DefaultLockDir()` to `disk_unix.go` and `disk_windows.go`
2. Update `main.go` to call `storage.DefaultDataDir()` and `storage.DefaultLockDir()`
3. Remove the hardcoded path strings from `main.go`
