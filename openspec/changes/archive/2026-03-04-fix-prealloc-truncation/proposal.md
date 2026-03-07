## Why

The ring manager pre-allocates segment files at startup using `fallocate` (Linux) or `SetEndOfFile` (Windows) via `preallocSegment()`. However, when `NewSegmentWriter` opens the same file for writing, it uses `os.O_TRUNC` which immediately truncates the file back to zero bytes. The pre-allocated disk space is released and segments grow dynamically as packets are written.

This defeats the core design guarantee from DESIGN.md §5.4:
- "Pre-Allocation at Startup: all segment files are created with their full size"
- "No Runtime Growth: the instance never increases its disk usage during operation"
- "New data overwrites existing segments in-place"

The same issue occurs on rotation: the new segment is opened with `O_TRUNC`, discarding whatever pre-allocated space remained.

## What Changes

- Remove `os.O_TRUNC` from `NewSegmentWriter` — instead, seek to the beginning and let the pcapng writer overwrite the existing content from offset 0
- After closing a segment (on rotation), the file retains its pre-allocated size on disk even if the written pcapng data is smaller — this is intentional and matches the "fixed footprint" design
- On rotation, the writer resets to the start of the file without truncating, ensuring the fallocate reservation persists

## Capabilities

### Modified Capabilities

- `segment-writer`: Opens pre-allocated files without truncation, preserving disk reservation
- `ring-manager`: Rotation no longer destroys pre-allocated space on the next segment

## Impact

- `internal/buffer/writer.go`: Remove `os.O_TRUNC` from `OpenFile` flags, add `File.Seek(0, 0)` and `File.Truncate` only after pcapng writing is complete (to trim any trailing garbage from the previous segment's larger content)
- `internal/buffer/writer_test.go`: Add test verifying that the file size after `Close()` does not exceed the original pre-allocated size
- `internal/buffer/ring.go`: No changes — `preallocSegment` already works correctly, the fix is entirely in the writer
