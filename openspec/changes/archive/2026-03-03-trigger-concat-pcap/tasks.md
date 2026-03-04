## 1. Persistence Layer — Concatenation Logic

- [x] 1.1 Add a `concatSegments(dst string, segments []buffer.SegmentMeta) error` function in `internal/persist/persist.go` that opens a single `pcapgo.NgWriter`, iterates sorted segments, reads packets via `pcapgo.NgReader`, and writes them to the output
- [x] 1.2 Handle active segment pre-allocation: wrap source file in `io.LimitReader(f, seg.Bytes)` when `seg.Bytes > 0` before passing to `NgReader`
- [x] 1.3 Sort segments by `StartTime` before iterating to ensure chronological packet order

## 2. Update SaveCapture

- [x] 2.1 Replace the per-segment `copyFile` loop in `SaveCapture()` with a call to `concatSegments()` targeting `capture.pcapng` in the destination directory
- [x] 2.2 Update `TriggerMeta` struct: replace `SegmentPaths []string` with `CapturePath string` (json tag `"capture_path"`)
- [x] 2.3 Set `meta.CapturePath = "capture.pcapng"` and remove the `copiedPaths` accumulation logic
- [x] 2.4 Remove the now-unused `copyFile` function

## 3. Error Handling

- [x] 3.1 Return an error from `SaveCapture` when the segments slice is empty (no segments in window)
- [x] 3.2 Ensure partial output file is cleaned up on concatenation error (remove `capture.pcapng` if merge fails mid-way)

## 4. Tests

- [x] 4.1 Update `TestSaveCaptureCopiesSegment` in `internal/persist/persist_test.go` to verify a single `capture.pcapng` is produced instead of individual segment files
- [x] 4.2 Add test: multiple segments are merged into one file with correct total packet count
- [x] 4.3 Add test: segments are output in chronological order regardless of index order (wraparound scenario)
- [x] 4.4 Add test: empty segments slice returns an error
- [x] 4.5 Add test: merged output is readable by `pcapgo.NgReader` and contains expected packets
- [x] 4.6 Verify `metadata.json` contains `capture_path` field instead of `segments` array
