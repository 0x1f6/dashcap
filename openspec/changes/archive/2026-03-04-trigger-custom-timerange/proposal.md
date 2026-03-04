## Why

The trigger API currently always persists exactly the configured `--pre-duration` window (default 5 minutes). Users need the ability to request a different time range per trigger call — for example, "save the last 30 minutes" or "save everything since timestamp X" — without changing the global configuration. This enables more flexible incident response and forensic workflows.

## What Changes

- **Rename `pre-duration` to `default-duration`** across CLI flag, config struct, YAML config, and documentation to clarify that this value serves as the default when no per-trigger range is specified.
- Extend `POST /api/v1/trigger` to accept an optional JSON request body with time-range parameters:
  - `duration` (string, e.g. `"30m"`): persist the last N minutes/seconds instead of the default duration.
  - `since` (RFC 3339 timestamp, e.g. `"2025-01-01T12:00:00Z"`): persist everything from this timestamp until now. Only if straightforward to implement.
- When no body or no time-range parameters are provided, the configured `default-duration` is used — fully backward-compatible.
- The effective time range is recorded in the trigger metadata (`metadata.json`) for auditability.
- If the requested range exceeds the available ring buffer data: **persist everything available** rather than failing. The response and metadata include a warning indicating that the persisted data is shorter than requested, so the caller is informed but no data is lost.

## Capabilities

### New Capabilities
- `trigger-timerange`: API parameters for specifying a custom time range on trigger calls (duration or since-timestamp), with best-effort persistence, warnings for incomplete data, and metadata recording.

### Modified Capabilities

_(none — existing specs are not affected at the requirement level)_

## Impact

- **API**: `POST /api/v1/trigger` gains an optional JSON request body (currently accepts no body). Existing clients sending no body continue to work unchanged.
- **CLI**: `--pre-duration` flag renamed to `--default-duration`. **BREAKING** for users who specify `--pre-duration` explicitly.
- **Config**: `pre_duration` YAML key renamed to `default_duration`.
- **Code**: Changes in `internal/api/server.go` (request parsing), `internal/trigger/trigger.go` (window calculation), `internal/persist/persist.go` (metadata fields), `internal/config/config.go` (rename), `cmd/dashcap/main.go` (flag rename).
- **Dependencies**: None — uses only stdlib `time` parsing.
