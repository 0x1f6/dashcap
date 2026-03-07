## 1. Segment Writer

- [x] 1.1 Add `SHBInfo` struct (Version, Hostname, Interface) to `internal/buffer/writer.go`
- [x] 1.2 Change `NewSegmentWriter` to accept `SHBInfo` parameter and call `pcapgo.NewNgWriterInterface` with `NgWriterOptions{SectionInfo: ...}` instead of `pcapgo.NewNgWriter`
- [x] 1.3 Set `Application` to `dashcap <version>` and `Comment` to `host=<hostname> interface=<iface>`, keep `Hardware`/`OS` from defaults

## 2. Ring Manager

- [x] 2.1 Update `Ring` (or equivalent manager) to accept and pass `SHBInfo` when creating new `SegmentWriter` instances on rotation

## 3. Persist Layer

- [x] 3.1 Update `persist` package to pass `SHBInfo` when writing the merged `capture.pcapng` for triggered saves

## 4. CLI Wiring

- [x] 4.1 Resolve `os.Hostname()` once at startup in `cmd/dashcap/main.go` and pass version, hostname, and interface name down to ring/writer construction

## 5. Tests

- [x] 5.1 Add unit test: verify `NewSegmentWriter` with `SHBInfo` produces pcapng with correct `shb_userappl` and `shb_comment` (read back with `pcapgo.NgReader` and check `SectionInfo`)
- [x] 5.2 Update existing writer/ring tests to pass the new `SHBInfo` parameter
