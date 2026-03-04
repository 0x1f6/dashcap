## Context

DESIGN.md §5.4 specifies that dashcap should refuse to start when the
allocation would leave less than **1 GB or 5 % of the partition, whichever is
larger**. The `Config` struct already carries both `MinFreeAfterAlloc` (absolute
bytes, default 1 GB) and `MinFreePercent` (default 5), but the disk safety
check in `NewRingManager` only evaluates the absolute threshold. The
percentage-based check is completely missing.

On large partitions (e.g. 100 GB) the percentage threshold (5 GB) is far more
protective than the 1 GB absolute — so the current code allows dashcap to
consume nearly all free space on those systems.

## Goals

1. Enforce both thresholds as specified in DESIGN.md §5.4.
2. Keep the change minimal — no new packages, no config schema change.
3. Maintain testability by extending the existing `DiskOps` interface.

## Decisions

### D1 — Extend `DiskOps` with `TotalBytes`

Add `TotalBytes(path string) (uint64, error)` to the `DiskOps` interface.
This mirrors the existing `FreeBytes` method and keeps disk queries behind
the same abstraction so tests can inject fake values.

**Unix:** `stat.Blocks * uint64(stat.Bsize)` from `unix.Statfs`.
**Windows:** return the `totalBytes` output parameter that `GetDiskFreeSpaceEx`
already provides (currently unused).

### D2 — Effective threshold = max(absolute, percentage)

In `NewRingManager`, after obtaining `totalDisk` via `TotalBytes`:

```
minFreePercent := uint64(float64(totalDisk) * cfg.MinFreePercent / 100)
effectiveMin   := max(minFreeAbs, minFreePercent)
```

The rest of the check stays the same — compare `free < ringTotal + effectiveMin`.

The error message will include both the effective margin and which threshold
triggered (absolute vs percentage) to aid debugging.

### D3 — Test fakes get a trivial `TotalBytes` stub

All three test fakes (`fakeDisk`, `triggerTestDisk`, `apiTestDisk`) return a
large constant (e.g. `100 << 30` = 100 GB) so existing tests keep passing
without changes to their assertions.

`fakeDisk` in ring_test.go gets an additional `totalBytes` field so the new
percentage-rejection test can set a custom total.

## Risks / Trade-offs

| Risk | Mitigation |
|------|-----------|
| `TotalBytes` call adds a second syscall on startup | Only called once at init; negligible overhead |
| Statfs `Blocks * Bsize` can overflow on very large volumes | Both values are `uint64`; overflow only at >16 EiB which exceeds real hardware |
| Float precision in percentage calculation | At 5 % of even a 1 PB disk the error is <1 byte; acceptable |
