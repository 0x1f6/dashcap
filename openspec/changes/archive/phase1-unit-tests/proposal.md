## Why

The Phase 1 implementation is complete and builds cleanly (`go build ./...`, `go vet ./...`), but has zero test files. Without tests, regressions in the core capture loop, ring buffer rotation, persistence, and API are invisible until they hit production.

## What Changes

- Add unit tests for `internal/config` — validation, defaults, size derivation
- Add unit tests for `internal/buffer` — `SegmentWriter` write/close lifecycle, `RingManager` rotation and `SegmentsInWindow` windowing logic
- Add unit tests for `internal/persist` — `SaveCapture` directory layout, metadata JSON correctness, copy fidelity
- Add unit tests for `internal/trigger` — `Dispatcher.Trigger` happy path, history ordering, concurrent safety
- Add unit tests for `internal/api` — HTTP handler responses for `/health`, `/status`, `/trigger`, `/triggers`, `/ring` using `net/http/httptest`
- Add unit tests for `internal/storage` — `disk_unix.go` `FreeBytes`, `Preallocate`, `LockFile`/`UnlockFile` (Linux-only build tags)
- Add a `parseSize` / `sanitize` test in `cmd/dashcap` (exported via a `_test.go` file in the same package)

## Capabilities

### New Capabilities

- `phase1-tests`: A passing `go test ./...` suite covering all Phase 1 packages with ≥ 80% line coverage on business logic paths.

### Modified Capabilities

<!-- none -->

## Impact

- New `*_test.go` files in every `internal/` package and `cmd/dashcap/`
- No changes to production code
- `Makefile` `test` target already runs `go test ./...` — no Makefile changes needed
