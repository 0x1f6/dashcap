## 1. Logger Setup

- [x] 1.1 Add `--debug` flag to CLI config in `internal/config/config.go`
- [x] 1.2 Initialize `slog.Logger` in `cmd/dashcap/main.go` based on `--debug` flag, set as default via `slog.SetDefault()`

## 2. Replace log calls in main.go

- [x] 2.1 Replace `log.Printf` calls for startup, signal handling, and shutdown with `slog.Info`
- [x] 2.2 Replace `log.Printf` calls in `captureLoop` for ring rotation and packet errors with `slog.Debug`
- [x] 2.3 Remove `log` import from `main.go` after migration

## 3. Replace log calls in API layer

- [x] 3.1 Add request logging to API handlers in `internal/api/server.go` using `slog.Info` (method, path, status)
- [x] 3.2 Replace any existing `log.Printf` calls in API code with appropriate slog level

## 4. Replace log calls in trigger and persist

- [x] 4.1 Replace `log.Printf` in `internal/trigger/trigger.go` — trigger fired/completed/failed at info level, detail logs at debug level
- [x] 4.2 Replace `log.Printf` in `internal/persist/persist.go` with appropriate slog level

## 5. Replace log calls in buffer layer

- [x] 5.1 Replace `log.Printf` in `internal/buffer/ring.go` — segment rotation and wrap-around at debug level

## 6. Tests

- [x] 6.1 Verify the server builds and existing tests pass with the new logging
- [x] 6.2 Add a test that `--debug` flag is recognized and sets log level accordingly
