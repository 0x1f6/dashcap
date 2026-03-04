## Context

`SegmentWriter.BytesWritten()` counts only raw packet payload bytes via `w.written += int64(len(data))` in `WritePacket()`. The capture loop in `main.go` uses `w.BytesWritten() >= segmentSize` to decide segment rotation. Because pcapng framing overhead (Section Header Block, Interface Description Block, Enhanced Packet Block headers, padding) is never counted, segments systematically exceed the configured `segmentSize` on disk.

## Goals / Non-Goals

**Goals:**
- `BytesWritten()` returns total file bytes written (pcapng framing + packet data)
- Segment rotation in the capture loop accurately reflects actual disk usage
- Zero-cost change — no extra syscalls, no performance regression

**Non-Goals:**
- Changing the pcapng writer library or packet format
- Modifying the ring manager or segment metadata semantics beyond the byte counter
- Predicting exact pcapng overhead per packet (we measure, not estimate)

## Decisions

### Approach: countingWriter wrapper around *os.File

**Decision:** Introduce an unexported `countingWriter` struct that wraps `*os.File`, implements `io.Writer`, and atomically increments a byte counter on every `Write()` call. Pass this wrapper to `pcapgo.NewNgWriter()` instead of the raw file.

**Rationale:** `pcapgo.NewNgWriter` accepts `io.Writer`. By wrapping the file, every byte — SHB, IDB, EPB headers, packet data, alignment padding — flows through the counter automatically. This is more reliable than:
- Manual overhead estimation (fragile, depends on pcapng library internals)
- `File.Seek(0, io.SeekCurrent)` per query (extra syscall on every check)

**Trade-off:** One extra indirection per `Write()` call. This is negligible — the cost is a pointer dereference and an integer addition, dwarfed by the underlying `write(2)` syscall.

### BytesWritten includes initial headers

**Decision:** `BytesWritten()` counts from the very first byte written to the file, including the SHB and IDB that `pcapgo.NewNgWriter` writes during construction.

**Rationale:** The goal is to track actual file size for rotation. The initial headers are part of the file's disk footprint and must be included.

### Remove the `written` field

**Decision:** Replace `SegmentWriter.written int64` with a pointer to the `countingWriter`, and have `BytesWritten()` read from the wrapper's counter.

**Rationale:** Eliminates dual bookkeeping. The counter lives in one place.

## Risks / Trade-offs

- **Test expectations change**: `TestWritePacketCounters` currently asserts `BytesWritten() == 200` for 2x100-byte packets. This will now be larger. The test must be updated to assert `BytesWritten() > 200` and cross-check against actual file size.
- **SegmentMeta.Bytes semantics shift**: `RingManager.Rotate()` stores `BytesWritten()` into `SegmentMeta.Bytes`. This now represents total file bytes rather than payload bytes. This is arguably more correct for a field named "Bytes" in a segment context, but consumers (if any) that relied on payload-only semantics would see a change. Currently no external consumers exist.

## Migration Plan

1. Add `countingWriter` struct to `writer.go`
2. Wire it into `NewSegmentWriter` — wrap `*os.File` before passing to `pcapgo.NewNgWriter`
3. Replace `written` field with `cw *countingWriter`, update `BytesWritten()` to return `cw.n`
4. Remove `w.written += int64(len(data))` from `WritePacket()`
5. Update `TestWritePacketCounters` to assert against actual file size
6. Run `make test` and `make lint`
