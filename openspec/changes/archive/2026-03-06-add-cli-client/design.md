## Context

dashcap runs as a daemon capturing packets into a ring buffer and exposes a REST API (port 9800) for status queries and trigger saves. The API supports bearer token auth and optional TLS. Currently users interact with the API via curl or similar tools. The project uses cobra for CLI commands and has no external HTTP client dependencies.

## Goals / Non-Goals

**Goals:**
- Provide a `dashcap client` subcommand group that covers all existing API endpoints
- Human-friendly pretty output by default in TTY, machine-parseable JSON otherwise
- Seamless token handling via flag or environment variable
- Separate client flags cleanly from server (capture) flags in CLI help

**Non-Goals:**
- Unix socket / IPC transport (Windows compatibility concern, API-only for now)
- Interactive/TUI mode
- Client-side caching or retry logic
- Watching/polling endpoints

## Decisions

### 1. Subcommand structure: `dashcap client <action>` vs top-level commands

**Decision:** Use `dashcap client <action>` (e.g. `dashcap client status`, `dashcap client trigger`).

**Rationale:** Keeps client commands namespaced and clearly separated from the server/capture root command. Avoids ambiguity (`dashcap status` could be confused with local daemon status). Cobra subcommand groups map naturally to this.

**Alternative considered:** Top-level commands (`dashcap status`, `dashcap trigger`) — rejected because it mixes client and server concerns and makes help output confusing.

### 2. Output mode: `--pretty` / `--json` flags with TTY auto-detection

**Decision:** Default to pretty output when stdout is a TTY, plain JSON when piped. `--pretty` forces pretty, `--json` forces JSON. Flags are mutually exclusive.

**Rationale:** Follows the convention of tools like `gh`, `kubectl`, `jq`. Auto-detection covers the common case; explicit flags handle edge cases (TTY but want JSON for copy-paste, or non-TTY but want pretty for logging).

**Pretty format:** Tabular/key-value text output with ANSI colors. No external dependency — use stdlib `text/tabwriter` and raw ANSI escape codes.

### 3. HTTP client: stdlib only

**Decision:** Use `net/http` directly. No external HTTP client library.

**Rationale:** The API surface is small (5 endpoints, simple JSON). Adding a dependency would be overkill. A thin `internal/client` package wraps the HTTP calls and returns typed Go structs.

### 4. Connection flags as persistent flags on `client` command

**Decision:** `--host`, `--port`, `--token`, `--tls`, `--tls-skip-verify` are persistent flags on the `client` parent command, inherited by all subcommands.

**Rationale:** Avoids repeating flags on every subcommand. Cobra persistent flags are designed for this.

### 5. Token resolution order

**Decision:** `--token` flag > `DASHCAP_API_TOKEN` env > error (no silent fallback to no-auth).

**Rationale:** Mirrors the server-side resolution order. If neither flag nor env is set, the client attempts the request without auth — the server will reject it if auth is enabled. No need to fail eagerly on the client side since the server might have `--no-auth`.

### 6. Command files organization

**Decision:** Add client commands in `cmd/dashcap/client.go` (parent + all subcommands in one file).

**Rationale:** Five small subcommands don't warrant separate files. If complexity grows, can split later.

## Risks / Trade-offs

- **Pretty output maintenance**: Adding new API fields requires updating pretty formatters → Mitigation: Keep formatters simple, one function per endpoint.
- **API version coupling**: Client assumes `/api/v1/` prefix → Mitigation: Acceptable for now since client and server ship as the same binary.
- **No connection reuse across commands**: Each invocation creates a fresh HTTP client → Acceptable since CLI commands are one-shot.
