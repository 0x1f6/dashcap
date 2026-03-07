## Context

The ring manager pre-allocates all segment files at startup via `preallocSegment()`, which calls `fallocate` (Linux) or `SetEndOfFile` (Windows) to reserve the full segment size on disk. This fulfils DESIGN.md §5.4: "Pre-Allocation at Startup: all segment files are created with their full size."

However, `NewSegmentWriter` opens each file with `os.O_TRUNC` (writer.go:41), which immediately truncates the file to zero bytes. The pre-allocated disk blocks are released and the segment grows dynamically as packets are written. The same truncation occurs on every rotation when the next segment is opened.

This means the core invariant — "the instance never increases its disk usage during operation" — is violated from the first second of operation.

## Goals / Non-Goals

**Goals:**
- Pre-allocated disk space persists through `NewSegmentWriter` — opening a segment for writing does not release reserved blocks
- Closed segments remain valid pcapng files readable by Wireshark and `pcapgo.NewNgReader`
- The fix is confined to `writer.go`; no changes to `ring.go` or the pre-allocation logic

**Non-Goals:**
- Changing the pre-allocation strategy (fallocate flags, FALLOC_FL_KEEP_SIZE, etc.)
- Making segment files retain their full pre-allocated size after Close — truncation to actual content size on Close is acceptable and necessary for pcapng validity
- Modifying the ring manager rotation logic

## Decisions

### Open without O_TRUNC, seek to offset 0

**Decision:** Replace `os.O_RDWR|os.O_CREATE|os.O_TRUNC` with `os.O_RDWR|os.O_CREATE` in `NewSegmentWriter`. After opening, explicitly seek to offset 0 before passing the file to `pcapgo.NewNgWriter`.

**Rationale:** `O_RDWR` without `O_APPEND` already positions the cursor at offset 0, but an explicit `Seek(0, io.SeekStart)` makes the intent clear and guards against future flag changes. The pcapng writer then overwrites existing content from the beginning, using the already-allocated disk blocks.

**Alternative considered:** Using `O_WRONLY` — rejected because `O_RDWR` is needed for `Truncate()` on Close and matches the existing permission model.

### Truncate to BytesWritten on Close

**Decision:** In `Close()`, call `f.Truncate(cw.n)` after the final `Flush()` and before `f.Close()`. This trims the file to the exact number of bytes written by the pcapng writer.

**Rationale:** Without truncation, the file retains its full pre-allocated size (e.g. 100 MB) with valid pcapng data at the beginning and stale zeros/old data trailing after it. Most pcapng readers (including `pcapgo.NewNgReader`) will fail or produce spurious errors when encountering trailing garbage. Truncating to the written size guarantees a clean, valid pcapng file.

**Trade-off:** After Close, the file's logical size shrinks to content size, releasing pre-allocated blocks beyond that point. This means the total disk footprint of closed segments is smaller than `segment_count * segment_size`. However, the key pre-allocation benefits are preserved:
1. **Disk space guarantee** — checked and reserved at startup
2. **No fragmentation** — blocks are contiguous from fallocate
3. **No runtime growth surprises** — the active segment writes into pre-allocated space, never requesting new blocks from the filesystem

### Existing BytesWritten == file size invariant is preserved

**Decision:** The existing test assertion `BytesWritten() == fi.Size()` in `TestWritePacketCounters` remains valid because `Close()` truncates the file to `BytesWritten()`.

**Rationale:** No test changes needed for this assertion. We add a *new* test specifically for the pre-allocation preservation behaviour.

## Risks / Trade-offs

- **pcapng header collision on reuse:** When a segment is reopened for the second rotation cycle, the file already contains old pcapng data. Since we write from offset 0 and truncate on Close, the old content is fully overwritten or trimmed. No risk of header collision.
- **Sparse file edge case:** On filesystems that don't support `fallocate` (e.g. some network mounts), `Preallocate` may fall back to writing zeros. The fix is unaffected — the file still exists at full size and is overwritten from offset 0.
- **countingWriter and Truncate interaction:** `cw.n` reflects bytes passed through the writer. Since `pcapgo.NewNgWriter` writes SHB+IDB during construction and EPBs during `WritePacket`, `cw.n` after `Flush()` is the exact byte offset of valid pcapng content. `Truncate(cw.n)` is safe.

## Migration Plan

1. Remove `os.O_TRUNC` from the `OpenFile` call in `NewSegmentWriter`
2. Add `f.Seek(0, io.SeekStart)` after opening
3. In `Close()`, add `f.Truncate(w.cw.n)` between `Flush()` and `f.Close()`
4. Add test: pre-allocate a file, open with `NewSegmentWriter`, verify the file was not truncated to 0 during open
5. Add test: write packets to a pre-allocated file, close, verify the file is a valid pcapng and its size equals `BytesWritten()`
6. Run `make test` and `make lint`
