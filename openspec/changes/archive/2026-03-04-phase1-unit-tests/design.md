## Context

dashcap's Phase 1 implementation (`capture`, `buffer`, `config`, `trigger`, `persist`, `api`, `storage`) is complete and compiles cleanly. No `*_test.go` files exist. The `Makefile` has a `test` target (`go test ./... -race -cover`) but running it today produces only "no test files" notices. The disk-safety and preallocation paths in `buffer` and `storage` need careful coverage because they guard against data loss and startup failures.

## Goals / Non-Goals

**Goals:**
- `go test ./...` passes with zero failures
- Race detector (`-race`) is clean on all concurrent tests
- Business-logic paths in `buffer`, `config`, `trigger`, and `persist` have ≥ 80% line coverage
- Tests are fast (< 5 s total), self-contained, and require no root privileges or real network interface
- All tests pass in the devcontainer without additional tooling

**Non-Goals:**
- End-to-end test against a real network interface (requires `CAP_NET_RAW`)
- Integration test for `storage.LockFile` across two processes (deferred to Phase 3)
- Benchmarks (deferred to Phase 4)
- Mocking the entire `gopacket/pcap` stack (only `capture.Source` interface is mocked)

## Decisions

### Avoid real disk preallocation in buffer tests

**Decision:** Use `os.TempDir()` for all file I/O in tests. For `Preallocate`, pass a real `*os.File` in a temp directory — `fallocate` is safe on tmpfs and will fall back to `ftruncate` gracefully.

**Rationale:** Tests must be runnable by any developer without sudo. `/var/lib/dashcap` requires elevated privileges; `os.TempDir()` does not.

**Alternative considered:** Mocking `storage.DiskOps` with a fake. Rejected for `storage` tests because we want to verify the actual syscall paths. For `buffer` tests, a `fakeDisk` stub is used to control `FreeBytes` return values and avoid actual preallocation during ring manager tests.

### Use `net/http/httptest` for API tests

**Decision:** Test all HTTP handlers via `httptest.NewRecorder()` directly, without starting a real TCP listener.

**Rationale:** Faster and more deterministic than listening on a port. Covers JSON encoding, HTTP status codes, and response body structure without port conflicts.

### Fake `capture.Source` for trigger/ring tests

**Decision:** Implement a minimal `fakeSource` in `_test.go` files that implements `capture.Source` and returns synthetic packets from a channel.

**Rationale:** `capture.PcapSource` requires libpcap and a real interface. For unit-testing the ring rotation and trigger logic, we only need a controllable `ReadPacketData()` source.

### Test package strategy: black-box where possible

**Decision:** Use `package <pkg>_test` (external test packages) for API, trigger, and persist tests. Use `package <pkg>` (white-box) only where internal fields need inspection (e.g., `buffer` ring index verification).

**Rationale:** External test packages catch unintentional coupling to internals and model how the package is actually used.

### No test helpers package

**Decision:** Each package's test file defines its own local helpers (`tempDir(t)`, `fakeDisk{}`). No shared `internal/testutil` package.

**Rationale:** The test suite is small enough that shared helpers add complexity without benefit. Each `*_test.go` file is self-contained.

## Risks / Trade-offs

- **tmpfs preallocation**: `fallocate` on tmpfs may return `EOPNOTSUPP` and fall back to `ftruncate`. The test should tolerate both outcomes. → Mitigation: `storage.Preallocate` already handles this fallback; the test just verifies `f.Stat().Size() == size`.
- **Race on `trigger.Dispatcher.save`**: The `save` goroutine updates `rec.Status` under `d.mu`; tests must wait for it to complete. → Mitigation: Poll `rec.Status` with a short `time.Sleep` loop or use a `sync.WaitGroup` injected via a test hook.
- **`pcapgo.NgWriter` minimum write**: Writing zero packets to a `SegmentWriter` then calling `Close()` must produce a valid (if empty) pcapng file. Verify with a `pcapgo.NgReader` round-trip in the test.

## Migration Plan

1. Add `*_test.go` files per package (no production code changes)
2. Run `go test ./... -race -cover` — fix any failures
3. Run `golangci-lint run` — fix any lint issues in test code
4. Update `Makefile` `test` target if coverage threshold enforcement is desired (optional)

No rollback needed — test files are additive and never shipped in the binary.
