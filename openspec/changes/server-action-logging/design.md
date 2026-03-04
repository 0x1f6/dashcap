## Context

dashcap currently logs all output via `log.Printf` to stderr with no level distinction. Operators running the server see ring rotation messages interleaved with important events like triggers and API requests. There is no way to filter output by severity. The project targets Go 1.21+ which includes `log/slog` in the standard library.

## Goals / Non-Goals

**Goals:**
- Introduce two log levels: **info** (default) and **debug**
- Info level covers operator-relevant events: server start/stop, API requests, trigger events, signal handling, errors
- Debug level covers internal mechanics: ring segment rotations, segment completions, buffer overflow/wrap-around, packet-level details
- Provide a `--debug` CLI flag to enable debug output
- Use Go's stdlib `log/slog` — no external dependencies

**Non-Goals:**
- Structured JSON logging output (can be added later via slog handler swap)
- Log file output or log rotation (operators can redirect stderr)
- Per-package or per-module log level control
- Request-scoped logging with trace IDs
- Metrics or observability integration

## Decisions

### 1. Use `log/slog` from Go stdlib

**Rationale**: Available since Go 1.21, provides leveled logging with structured fields out of the box. No external dependency needed. The `slog.TextHandler` produces human-readable output similar to current `log.Printf` format.

**Alternatives considered**:
- `zerolog` / `zap`: More features but adds external dependency for a simple two-level need.
- Custom wrapper around `log.Printf`: More work, less standard, no structured fields.

### 2. Global logger initialized at startup

**Rationale**: Create the slog logger in `main.go` during startup based on the `--debug` flag, set it as the default via `slog.SetDefault()`. All code uses `slog.Info()` / `slog.Debug()` package-level functions. This avoids threading a logger through every function/struct.

**Alternatives considered**:
- Dependency injection (pass logger to each component): Cleaner but requires API changes to every struct constructor. Over-engineered for a CLI tool.

### 3. Two levels only: info and debug

**Rationale**: The use case is binary — operators want action logs or full verbose output. Warn/error aren't needed as separate levels since errors are already logged and are infrequent. `slog.Info` for actions, `slog.Debug` for internals.

### 4. `--debug` flag instead of `--log-level`

**Rationale**: Simpler UX. A boolean flag is easier to document and use than a level string. If more granularity is needed later, `--log-level` can be added without breaking `--debug`.

## Risks / Trade-offs

- **[Minimal behavior change]** Log output format changes from `log.Printf` to slog's text format (includes level prefix, timestamp format differs slightly). → Acceptable since no tooling parses the current log format.
- **[Debug noise in tests]** Tests that capture log output may need adjustment. → Tests should not depend on log output; if they do, fix them.
