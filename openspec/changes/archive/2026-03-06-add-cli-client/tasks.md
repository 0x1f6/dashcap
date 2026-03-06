## 1. HTTP Client Package

- [x] 1.1 Create `internal/client` package with `Client` struct (host, port, token, TLS config)
- [x] 1.2 Implement request helper: build URL, set auth header, execute request, decode JSON response
- [x] 1.3 Implement typed methods: `Health()`, `Status()`, `Trigger(opts)`, `Triggers()`, `Ring()`
- [x] 1.4 Handle non-2xx responses: extract error message from JSON body, return as Go error

## 2. Output Formatting

- [x] 2.1 Create output mode detection: TTY check + `--pretty`/`--json` flag resolution
- [x] 2.2 Implement pretty formatters for each endpoint response (tabwriter + ANSI colors)
- [x] 2.3 Implement JSON output (plain `json.Encoder` to stdout)

## 3. Cobra Commands

- [x] 3.1 Create `cmd/dashcap/client.go` with `clientCmd` parent command and persistent flags (`--host`, `--port`, `--token`, `--tls`, `--tls-skip-verify`, `--pretty`, `--json`)
- [x] 3.2 Implement `health` subcommand
- [x] 3.3 Implement `status` subcommand
- [x] 3.4 Implement `trigger` subcommand with `--duration` and `--since` flags (mutually exclusive)
- [x] 3.5 Implement `triggers` subcommand
- [x] 3.6 Implement `ring` subcommand
- [x] 3.7 Register `clientCmd` in `rootCmd()` in `main.go`

## 4. Testing

- [x] 4.1 Unit tests for `internal/client` using `httptest.Server`
- [x] 4.2 Unit tests for output mode detection and pretty formatters
- [x] 4.3 Integration test: spin up API server, run client subcommands, verify output
