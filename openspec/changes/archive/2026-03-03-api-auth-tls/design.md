## Context

dashcap's REST API currently runs as a plain HTTP server on a configurable port (default 9800). There is no authentication — anyone who can reach the port can trigger saves, list triggers, and view capture metadata. This is acceptable for local-only development but unsuitable for production, where dashcap may be exposed on a network segment with other hosts.

The API serves five endpoints via `net/http.ServeMux`. Configuration is CLI-flag-only via Cobra, stored in a `config.Config` struct.

## Goals / Non-Goals

**Goals:**
- Protect state-changing endpoints (`POST /api/v1/trigger`) and data-reading endpoints from unauthorized access
- Provide TLS to encrypt API traffic (tokens in `Authorization` headers must not be sent in cleartext)
- Make auth and TLS enabled by default, but allow operators to opt out
- Zero new external dependencies — stdlib only

**Non-Goals:**
- Multi-user / role-based access control — a single shared token is sufficient for Phase 1
- Automatic certificate provisioning (ACME / Let's Encrypt) — operators supply cert/key files
- Mutual TLS (mTLS) / client certificate authentication
- Token rotation at runtime — requires restart to change token

## Decisions

### 1. Bearer token authentication via middleware

**Decision**: Implement auth as an `http.Handler` middleware that wraps the existing `ServeMux`.

**Rationale**: This is the simplest approach — a single middleware function checks `Authorization: Bearer <token>` on every request and returns 401 if missing/invalid. No changes needed to individual handler functions. The health endpoint (`GET /api/v1/health`) is exempted so load balancers and monitoring can still probe liveness without credentials.

**Alternatives considered**:
- API key in query parameter — leaks in logs and browser history
- Basic Auth — works but Bearer is the standard for token-based auth
- Per-handler auth checks — error-prone, violates DRY

### 2. Token generation and configuration

**Decision**: At startup, generate a 32-byte (256-bit) cryptographically random token encoded as hex (64 characters). Allow override via `--api-token` flag or `DASHCAP_API_TOKEN` environment variable. Print the active token to stderr on startup.

**Rationale**: Auto-generation provides secure defaults. Hex encoding is simple and copy-paste friendly. Environment variable support enables container/systemd deployments without exposing the token in process arguments. CLI flag takes precedence over env var for explicit override.

**Token precedence**: `--api-token` flag > `DASHCAP_API_TOKEN` env var > auto-generated

### 3. TLS via cert/key files

**Decision**: Add `--tls-cert` and `--tls-key` flags. When both are set, the server uses `tls.LoadX509KeyPair` and `srv.ServeTLS`. When neither is set, the server runs plain HTTP (with a log warning if auth is also enabled, since tokens would be sent in cleartext).

**Rationale**: Operator-supplied certs give full control. Most production deployments behind a reverse proxy can terminate TLS there and run dashcap plain. Self-signed certs work for direct access.

**Alternatives considered**:
- Auto-generate self-signed cert — convenient but causes TLS verification failures for clients; better to be explicit
- Require TLS when auth is enabled — too restrictive for reverse-proxy setups

### 4. `--no-auth` flag to disable authentication

**Decision**: Auth is enabled by default. Pass `--no-auth` to disable it. When disabled, no token is generated and the middleware is skipped entirely.

**Rationale**: Secure by default. The explicit opt-out flag makes the operator's intent clear in command history and process arguments.

### 5. Config struct additions

**Decision**: Add to `config.Config`:
- `APIToken string` — the active token (auto-generated or user-supplied)
- `APINoAuth bool` — disable authentication
- `TLSCert string` — path to TLS certificate file
- `TLSKey string` — path to TLS private key file

**Rationale**: Keeps all configuration in the existing struct. Validation ensures cert/key are either both set or both empty.

## Risks / Trade-offs

- **Token in stderr** — the token is printed at startup. If stderr is captured to a world-readable log, the token is exposed. → Mitigation: operators should restrict log file permissions, or supply their own token via env var.
- **No TLS by default** — tokens sent over plain HTTP are vulnerable to eavesdropping. → Mitigation: log a warning when auth is enabled without TLS. Documentation should recommend TLS or a reverse proxy.
- **Single token** — all clients share one token; compromising it compromises the entire API. → Mitigation: acceptable for Phase 1; multi-token support can be added later.
- **Constant-time comparison** — token comparison must use `crypto/subtle.ConstantTimeCompare` to avoid timing attacks. → Already planned for the middleware implementation.
