## ADDED Requirements

### Requirement: Client subcommand group
The CLI SHALL provide a `dashcap client` command group containing subcommands for each API endpoint: `health`, `status`, `trigger`, `triggers`, `ring`.

#### Scenario: Help output lists all client subcommands
- **WHEN** user runs `dashcap client --help`
- **THEN** all five subcommands (`health`, `status`, `trigger`, `triggers`, `ring`) are listed with descriptions

#### Scenario: Unknown subcommand
- **WHEN** user runs `dashcap client foobar`
- **THEN** CLI exits with error and shows available subcommands

### Requirement: Connection configuration
The `client` command SHALL accept persistent flags `--host` (default: `localhost`), `--port` (default: `9800`), `--tls` (default: false), and `--tls-skip-verify` (default: false) inherited by all subcommands.

#### Scenario: Default connection target
- **WHEN** user runs `dashcap client health` without `--host` or `--port`
- **THEN** client connects to `http://localhost:9800`

#### Scenario: Custom host and port
- **WHEN** user runs `dashcap client status --host 10.0.0.5 --port 8080`
- **THEN** client connects to `http://10.0.0.5:8080`

#### Scenario: TLS enabled
- **WHEN** user runs `dashcap client status --tls`
- **THEN** client connects using `https://` scheme

#### Scenario: TLS with skip verify
- **WHEN** user runs `dashcap client status --tls --tls-skip-verify`
- **THEN** client connects using HTTPS and skips certificate verification

### Requirement: Authentication token handling
The `client` command SHALL accept a `--token` persistent flag. If not provided, it SHALL fall back to the `DASHCAP_API_TOKEN` environment variable. If neither is set, it SHALL attempt to read the token from the token file (default `/etc/dashcap/api-token`, overridable via `--token-file`). If the token file is unreadable or missing, requests are sent without an Authorization header.

#### Scenario: Token via flag
- **WHEN** user runs `dashcap client status --token abc123`
- **THEN** request includes header `Authorization: Bearer abc123`

#### Scenario: Token via environment variable
- **WHEN** `DASHCAP_API_TOKEN=envtoken` is set and no `--token` flag is provided
- **THEN** request includes header `Authorization: Bearer envtoken`

#### Scenario: Flag overrides environment
- **WHEN** `DASHCAP_API_TOKEN=envtoken` is set and `--token flagtoken` is provided
- **THEN** request includes header `Authorization: Bearer flagtoken`

#### Scenario: Token from file fallback
- **WHEN** neither `--token` flag nor `DASHCAP_API_TOKEN` env var is set
- **AND** `/etc/dashcap/api-token` contains a valid token
- **THEN** request includes `Authorization: Bearer <token-from-file>`

#### Scenario: Custom token file path
- **WHEN** `--token-file /opt/dashcap/token` is specified and the file contains a valid token
- **THEN** the token from that file is used

#### Scenario: No token available
- **WHEN** neither `--token` flag, `DASHCAP_API_TOKEN` env var, nor readable token file is available
- **THEN** request is sent without an Authorization header

### Requirement: Output mode selection
All client subcommands SHALL support `--pretty` and `--json` flags. When stdout is a TTY and neither flag is set, pretty output is used. When stdout is not a TTY, JSON output is used. `--pretty` forces pretty output. `--json` forces JSON output. The flags are mutually exclusive.

#### Scenario: TTY default is pretty
- **WHEN** stdout is a TTY and neither `--pretty` nor `--json` is specified
- **THEN** output is human-readable formatted text

#### Scenario: Pipe default is JSON
- **WHEN** stdout is not a TTY (piped) and neither flag is specified
- **THEN** output is plain JSON

#### Scenario: Force JSON in TTY
- **WHEN** stdout is a TTY and `--json` is specified
- **THEN** output is plain JSON

#### Scenario: Force pretty in pipe
- **WHEN** stdout is not a TTY and `--pretty` is specified
- **THEN** output is human-readable formatted text

#### Scenario: Mutually exclusive flags
- **WHEN** both `--pretty` and `--json` are specified
- **THEN** CLI exits with an error message

### Requirement: Health subcommand
`dashcap client health` SHALL send `GET /api/v1/health` and display the result.

#### Scenario: Healthy server
- **WHEN** user runs `dashcap client health` and server responds 200
- **THEN** output shows health status and CLI exits with code 0

#### Scenario: Unreachable server
- **WHEN** user runs `dashcap client health` and server is unreachable
- **THEN** CLI prints an error message and exits with non-zero code

### Requirement: Status subcommand
`dashcap client status` SHALL send `GET /api/v1/status` and display the result.

#### Scenario: Successful status query
- **WHEN** user runs `dashcap client status` and server responds 200
- **THEN** output shows interface, uptime, segment count, total packets, and total bytes

### Requirement: Trigger subcommand
`dashcap client trigger` SHALL send `POST /api/v1/trigger` and display the result. It SHALL accept optional `--duration` and `--since` flags matching the API's JSON body fields.

#### Scenario: Trigger without options
- **WHEN** user runs `dashcap client trigger`
- **THEN** POST request is sent with empty body and trigger result is displayed

#### Scenario: Trigger with duration
- **WHEN** user runs `dashcap client trigger --duration 30s`
- **THEN** POST request body contains `{"duration": "30s"}`

#### Scenario: Trigger with since
- **WHEN** user runs `dashcap client trigger --since 2024-01-01T00:00:00Z`
- **THEN** POST request body contains `{"since": "2024-01-01T00:00:00Z"}`

#### Scenario: Duration and since mutually exclusive
- **WHEN** user runs `dashcap client trigger --duration 30s --since 2024-01-01T00:00:00Z`
- **THEN** CLI exits with an error message

### Requirement: Triggers subcommand
`dashcap client triggers` SHALL send `GET /api/v1/triggers` and display the trigger history.

#### Scenario: List trigger history
- **WHEN** user runs `dashcap client triggers` and server responds 200
- **THEN** output shows list of past triggers

### Requirement: Ring subcommand
`dashcap client ring` SHALL send `GET /api/v1/ring` and display ring buffer segment metadata.

#### Scenario: Show ring segments
- **WHEN** user runs `dashcap client ring` and server responds 200
- **THEN** output shows per-segment metadata (packets, bytes, timestamps)

### Requirement: HTTP error handling
All subcommands SHALL handle non-2xx HTTP responses by displaying the error message from the response body and exiting with a non-zero code.

#### Scenario: 401 Unauthorized
- **WHEN** server responds with 401
- **THEN** CLI prints an authentication error and exits with non-zero code

#### Scenario: 500 Server Error
- **WHEN** server responds with 500
- **THEN** CLI prints the error from the response body and exits with non-zero code
