## Why

dashcap has a REST API for triggering saves, querying status, and inspecting the ring buffer, but no built-in CLI client. Users currently rely on curl or other HTTP tools, which is cumbersome (token handling, JSON parsing, URL construction). A built-in client makes dashcap self-contained and provides both human-friendly and machine-parseable output.

## What Changes

- New `dashcap client` subcommand group with sub-commands for each API endpoint (`health`, `status`, `trigger`, `triggers`, `ring`)
- `--pretty` flag for formatted, colored output (default when stdout is a TTY); `--json` flag to force plain JSON even in a TTY
- Connection target configurable via `--host` (default: `localhost`) and `--port` (default: `9800`)
- Token handling: `--token` flag with fallback to `DASHCAP_API_TOKEN` environment variable
- TLS support: `--tls` flag with optional `--tls-skip-verify` for self-signed certificates
- CLI help documentation split into server flags vs. client flags for clarity

## Capabilities

### New Capabilities
- `cli-client`: CLI client subcommands for the REST API with human-readable and machine-readable output

### Modified Capabilities

_(no existing specs affected)_

## Impact

- New package `internal/client` for HTTP client logic
- New subcommands under `cmd/dashcap/` (separate files for client commands)
- Dependencies: no new external dependencies needed (stdlib `net/http` + `encoding/json`, cobra already present)
- API endpoints remain unchanged — client is a pure consumer
