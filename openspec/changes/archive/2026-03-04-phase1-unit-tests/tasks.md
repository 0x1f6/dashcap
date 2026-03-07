## 1. Config Package Tests

- [x] 1.1 Create `internal/config/config_test.go` (package `config_test`)
- [x] 1.2 Test `Defaults()` — verify `BufferSize`, `SegmentSize`, `SegmentCount`, `APIPort`, `Promiscuous` values
- [x] 1.3 Test `Validate()` derives `SegmentCount` correctly (1 GB / 100 MB = 10)
- [x] 1.4 Test `Validate()` returns error when `Interface` is empty
- [x] 1.5 Test `Validate()` returns error when `BufferSize < SegmentSize`
- [x] 1.6 Test `Validate()` returns error when derived `SegmentCount < 2`

## 2. Storage Package Tests

- [x] 2.1 Create `internal/storage/disk_unix_test.go` (`//go:build linux`, package `storage_test`)
- [x] 2.2 Test `FreeBytes` with `os.TempDir()` — assert result > 0
- [x] 2.3 Test `Preallocate` — create temp file, call `Preallocate(f, 1<<20)`, assert `Stat().Size() == 1<<20`
- [x] 2.4 Test `LockFile` — assert returns nil on an open temp file
- [x] 2.5 Test `LockFile` + `UnlockFile` cycle — both return nil

## 3. Buffer: SegmentWriter Tests

- [x] 3.1 Create `internal/buffer/writer_test.go` (package `buffer`)
- [x] 3.2 Test `NewSegmentWriter` — creates file, `StartTime()` is within last second
- [x] 3.3 Test `WritePacket` — write 2 packets of 100 bytes; assert `PacketCount() == 2`, `BytesWritten() == 200`
- [x] 3.4 Test `Close` + pcapng round-trip — open resulting file with `pcapgo.NewNgReader`, read packets, verify count matches

## 4. Buffer: RingManager Tests

- [x] 4.1 Create `internal/buffer/ring_test.go` (package `buffer`)
- [x] 4.2 Define `fakeDisk` struct implementing `storage.DiskOps` with configurable `FreeBytes` return
- [x] 4.3 Test `NewRingManager` rejects insufficient free space (fakeDisk returns 0)
- [x] 4.4 Test `NewRingManager` succeeds when free space is adequate — segment files are created
- [x] 4.5 Test `Rotate()` advances to `segment_001.pcapng` after first rotation
- [x] 4.6 Test `Rotate()` wraps around after N rotations on a 3-segment ring
- [x] 4.7 Test `SegmentsInWindow` — set up two segments with non-overlapping time ranges, assert only the matching one is returned
- [x] 4.8 Test `Close()` on a freshly created ring — returns nil

## 5. Persist Package Tests

- [x] 5.1 Create `internal/persist/persist_test.go` (package `persist_test`)
- [x] 5.2 Create helper that writes a small temp file to use as a fake segment
- [x] 5.3 Test `SaveCapture` creates a directory under `savedDir` matching `<timestamp>_<source>`
- [x] 5.4 Test `SaveCapture` writes `metadata.json` — unmarshal and check `trigger_id`, `source`, `interface`
- [x] 5.5 Test `SaveCapture` copies segment content byte-for-byte into the saved directory
- [x] 5.6 Test `SaveCapture` returns the correct `savedPath`

## 6. Trigger Package Tests

- [x] 6.1 Create `internal/trigger/trigger_test.go` (package `trigger_test`)
- [x] 6.2 Set up a minimal `fakeDisk` + `fakeRingManager` stub that `SegmentsInWindow` returns an empty slice
- [x] 6.3 Test `Trigger("api")` returns a record with `Status == "pending"` and non-empty `ID`
- [x] 6.4 Test `History()` returns newest-first order after 3 sequential triggers
- [x] 6.5 Test concurrent safety: launch 10 goroutines calling `Trigger("test")`, assert `len(History()) == 10` (run with `-race`)

## 7. API Package Tests

- [x] 7.1 Create `internal/api/server_test.go` (package `api_test`)
- [x] 7.2 Create `newTestServer(t)` helper that wires a minimal `config.Config`, a stub `*buffer.RingManager`, and a `*trigger.Dispatcher`
- [x] 7.3 Test `GET /api/v1/health` — assert HTTP 200, body contains `"ok"`
- [x] 7.4 Test `GET /api/v1/status` — assert HTTP 200, body contains the configured interface name
- [x] 7.5 Test `POST /api/v1/trigger` — assert HTTP 202, body contains non-empty `"id"` field
- [x] 7.6 Test `GET /api/v1/triggers` — assert HTTP 200, body is a JSON array
- [x] 7.7 Test `GET /api/v1/ring` — assert HTTP 200, body is a JSON array

## 8. CLI Helper Tests

- [x] 8.1 Create `cmd/dashcap/main_test.go` (package `main`)
- [x] 8.2 Test `parseSize("2GB", &n)` → `n == 2<<30`
- [x] 8.3 Test `parseSize("100MB", &n)` → `n == 100<<20`
- [x] 8.4 Test `parseSize("512KB", &n)` → `n == 512<<10`
- [x] 8.5 Test `parseSize("notasize", &n)` returns an error
- [x] 8.6 Test `sanitize("Wi-Fi 2.4GHz")` — result contains only `[a-zA-Z0-9_-]` characters

## 9. Verification

- [x] 9.1 Run `go test ./... -race` — zero failures, no race conditions
- [x] 9.2 Run `go test ./... -cover` — confirm `internal/config`, `internal/buffer`, `internal/persist`, `internal/trigger` coverage ≥ 80%
- [x] 9.3 Run `golangci-lint run` — zero lint errors including in test files
- [x] 9.4 Run `go vet ./...` — clean
