## Why

The server currently uses ad-hoc `log.Printf` calls with no log levels. All output — from important events like triggers firing and API requests to internal details like ring segment rotations — goes to the same stream. Operators need to see actionable events at a glance without noise from internal buffer management, while still being able to enable verbose output for debugging.

## What Changes

- Introduce a structured logging approach with at least two levels: **info** (default) and **debug**.
- **Info level** logs user-visible actions: trigger events, API requests, server start/stop, signal handling, and errors.
- **Debug level** logs internal behavior: ring segment completions, segment rotations/overflows, packet write details, and buffer statistics.
- Replace existing `log.Printf` calls with level-appropriate log calls throughout the codebase.
- Add a CLI flag (e.g., `--debug` or `--log-level`) to enable debug output.

## Capabilities

### New Capabilities
- `structured-logging`: Introduces leveled logging (info/debug) with a logger abstraction that replaces raw `log.Printf` calls across the server.

### Modified Capabilities
<!-- No existing spec requirements are changing — this is purely additive behavior. -->

## Impact

- **Code**: All files using `log.Printf` — primarily `cmd/dashcap/main.go`, `internal/api/server.go`, `internal/buffer/ring.go`, `internal/trigger/trigger.go`, `internal/persist/persist.go`.
- **APIs**: No API changes.
- **CLI**: New `--debug` or `--log-level` flag added to the command.
- **Dependencies**: Uses Go's `log/slog` (stdlib, Go 1.21+) — no new external dependencies.
