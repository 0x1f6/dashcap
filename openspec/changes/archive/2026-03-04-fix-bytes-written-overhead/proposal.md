## Why

`SegmentWriter.BytesWritten()` only counts raw packet payload bytes (`len(data)`) but ignores pcapng framing overhead (Section Header Block, Interface Description Block, Enhanced Packet Block headers). The capture loop in `main.go` uses `BytesWritten() >= segmentSize` to decide when to rotate — so segments systematically grow larger than `segmentSize` because the pcapng structural overhead is never accounted for.

This means:
- Ring segments exceed the pre-allocated file size, causing writes beyond the fallocate boundary
- The total ring buffer footprint is unpredictably larger than `segment_count * segment_size`
- The disk safety guarantee ("fixed footprint, no growth at runtime" per DESIGN.md §5.4) is violated

## What Changes

- Replace the payload-only byte counter with a counter that tracks total bytes written to the underlying file, including all pcapng framing
- The most reliable approach is to use the file's actual write position (via `File.Seek(0, io.SeekCurrent)` or wrapping the writer to count bytes passing through) rather than manually estimating pcapng overhead
- Update `SegmentWriter.BytesWritten()` to return the total file size, not just payload

## Capabilities

### Modified Capabilities

- `segment-writer`: `BytesWritten()` returns total file bytes written (pcapng headers + packet data) instead of only packet payload bytes
- `capture-loop`: Rotation decision is now accurate — segments rotate at the correct file size boundary

## Impact

- `internal/buffer/writer.go`: Change byte counting mechanism to track actual file position
- `internal/buffer/writer_test.go`: Update `TestWritePacketCounters` — `BytesWritten()` will now return a value larger than the sum of packet payloads
- `cmd/dashcap/main.go`: No changes needed (comparison logic remains `w.BytesWritten() >= segmentSize`)
