## Why

The `Config` struct defines `MinFreePercent float64` (config.go line 33) and the example configuration documents it as `min_free_percent: 5`. However, the disk safety check in `RingManager.NewRingManager()` (ring.go lines 50–56) only checks `MinFreeAfterAlloc` (absolute bytes) and completely ignores `MinFreePercent`.

DESIGN.md §5.4 specifies: "the allocation would leave less than a configurable minimum (default: 1 GB **or** 5% of partition, **whichever is larger**)". The current implementation only enforces the absolute threshold, missing the percentage-based check entirely.

On a small partition (e.g., 15 GB), 5% = 750 MB which is less than the 1 GB absolute — so the absolute check is sufficient. But on a large partition (e.g., 100 GB), 5% = 5 GB which is far more protective than the 1 GB absolute. Without the percentage check, dashcap could consume nearly all free space on large partitions.

## What Changes

- Extend the disk safety check in `NewRingManager` to also query total partition size
- Calculate `minFreePercent` as `totalSize * cfg.MinFreePercent / 100`
- Use `max(MinFreeAfterAlloc, minFreePercent)` as the effective safety margin
- Add `TotalBytes(path string) (uint64, error)` to the `DiskOps` interface to provide total partition size

## Capabilities

### Modified Capabilities

- `ring-manager`: Disk safety check enforces both absolute and percentage-based free space thresholds, using whichever is larger
- `storage.DiskOps`: New `TotalBytes()` method for querying total partition size

## Impact

- `internal/storage/storage.go`: Add `TotalBytes(path string) (uint64, error)` to `DiskOps` interface
- `internal/storage/disk_unix.go`: Implement `TotalBytes` using `statfs.Blocks * statfs.Bsize`
- `internal/storage/disk_windows.go`: Implement `TotalBytes` using the existing `GetDiskFreeSpaceEx` `totalBytes` output parameter
- `internal/buffer/ring.go`: Extend safety check to compute percentage-based minimum and use `max()` of both thresholds
- `internal/buffer/ring_test.go`: Add test case for percentage-based rejection (fake disk with small total size)
- `internal/storage/disk_unix_test.go`: Add test for `TotalBytes`
- Test fakes (`fakeDisk`, `triggerTestDisk`, `apiTestDisk`): Add stub `TotalBytes` method
