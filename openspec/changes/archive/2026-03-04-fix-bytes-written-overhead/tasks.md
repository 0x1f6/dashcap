## 1. Add countingWriter to writer.go

- [x] 1.1 Add unexported `countingWriter` struct with `w io.Writer` and `n int64` fields, implementing `io.Writer` — each `Write()` increments `n` by the number of bytes written
- [x] 1.2 Replace `written int64` field in `SegmentWriter` with `cw *countingWriter`
- [x] 1.3 In `NewSegmentWriter`, wrap `*os.File` in a `countingWriter` and pass it to `pcapgo.NewNgWriter`
- [x] 1.4 Remove `w.written += int64(len(data))` from `WritePacket()`
- [x] 1.5 Update `BytesWritten()` to return `w.cw.n`
- [x] 1.6 Update the doc comment on `BytesWritten()` to say "total bytes written to the file" instead of "payload bytes"

## 2. Update tests

- [x] 2.1 In `TestWritePacketCounters`, replace `BytesWritten() != 200` assertion with a check that `BytesWritten() > 200` (payload + pcapng framing)
- [x] 2.2 Add a file-size cross-check: after `Close()`, stat the file and assert `BytesWritten() == fileInfo.Size()`

## 3. Verify

- [x] 3.1 Run `make test` — all tests pass
- [x] 3.2 Run `make lint` — zero lint errors
