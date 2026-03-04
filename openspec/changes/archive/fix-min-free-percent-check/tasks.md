## Tasks

### 1 — Extend DiskOps interface

- [x] Add `TotalBytes(path string) (uint64, error)` to the `DiskOps` interface in `internal/storage/storage.go`

### 2 — Platform implementations

- [x] Implement `TotalBytes` in `internal/storage/disk_unix.go` using `unix.Statfs` (`stat.Blocks * uint64(stat.Bsize)`)
- [x] Implement `TotalBytes` in `internal/storage/disk_windows.go` returning the `totalBytes` output from `GetDiskFreeSpaceEx`

### 3 — Update disk safety check

- [x] In `NewRingManager` (`internal/buffer/ring.go`), call `disk.TotalBytes(cfg.DataDir)` to obtain total partition size
- [x] Calculate `minFreePercent := uint64(float64(totalDisk) * cfg.MinFreePercent / 100)`
- [x] Use `max(minFreeAbs, minFreePercent)` as the effective safety margin in the existing comparison
- [x] Update the error message to include the effective margin and which threshold applied

### 4 — Update test fakes

- [x] Add `TotalBytes` stub to `fakeDisk` in `internal/buffer/ring_test.go` (with configurable `totalBytes` field)
- [x] Add `TotalBytes` stub to `triggerTestDisk` in `internal/trigger/trigger_test.go` (return `100 << 30`)
- [x] Add `TotalBytes` stub to `apiTestDisk` in `internal/api/server_test.go` (return `100 << 30`)

### 5 — Tests

- [x] Add unit test for `TotalBytes` in `internal/storage/disk_unix_test.go`
- [x] Add `TestNewRingManagerPercentRejectsAllocation` in `internal/buffer/ring_test.go`: fake disk with small total size so the percentage threshold is larger and triggers rejection
- [x] Verify existing `TestNewRingManagerInsufficientSpace` still passes (absolute threshold path)

### 6 — Verify

- [x] `go build ./...` compiles without errors
- [x] `go test ./...` passes all tests
