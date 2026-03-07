## 1. Fix NewSegmentWriter open flags

- [x] 1.1 Remove `os.O_TRUNC` from the `os.OpenFile` call in `NewSegmentWriter` (writer.go:41) — change flags to `os.O_RDWR|os.O_CREATE`
- [x] 1.2 Add `f.Seek(0, io.SeekStart)` after the `OpenFile` call, before creating the `countingWriter`
- [x] 1.3 Update the doc comment on `NewSegmentWriter` — replace "creates (or truncates)" with "opens (or creates)"

## 2. Truncate on Close

- [x] 2.1 In `Close()`, add `f.Truncate(w.cw.n)` after the final `Flush()` and before `f.Close()` — this trims trailing pre-allocated/stale bytes so the file is a valid pcapng
- [x] 2.2 Handle the `Truncate` error: if it fails, still attempt `f.Close()` but return the truncate error

## 3. Add tests

- [x] 3.1 Add `TestNewSegmentWriterPreservesPrealloc`: create a temp file, write 1 MB of zeros (simulating pre-allocation), open with `NewSegmentWriter`, verify file size is still >= 1 MB before writing any packets
- [x] 3.2 Add `TestCloseOnPreallocatedFile`: pre-allocate a file, open with `NewSegmentWriter`, write 3 packets, close, then verify: (a) file size == `BytesWritten()`, (b) file is readable by `pcapgo.NewNgReader` returning exactly 3 packets

## 4. Verify

- [x] 4.1 Run `make test` — all tests pass (including existing `TestWritePacketCounters` BytesWritten == file size check)
- [x] 4.2 Run `make lint` — zero lint errors
