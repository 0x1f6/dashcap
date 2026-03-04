## Why

The dashcap REST API currently accepts unauthenticated requests from anyone who can reach the port. Since the `/api/v1/trigger` endpoint causes disk writes (saving captured traffic), unrestricted access is a security risk — any network-adjacent actor can trigger saves, fill disk, or observe capture metadata. The API needs token-based authentication and TLS to protect it in production deployments.

## What Changes

- Add bearer-token authentication middleware that validates every API request (except `/api/v1/health`)
- Auto-generate a secure random token at startup; optionally accept a user-supplied token via CLI flag (`--api-token`) or environment variable (`DASHCAP_API_TOKEN`)
- Add TLS support to the HTTP server using user-provided cert/key files (`--tls-cert`, `--tls-key`)
- Add `--no-auth` flag to explicitly disable authentication (off by default — auth is enabled)
- Print the active token to stderr on startup so the operator can retrieve it
- **BREAKING**: Existing unauthenticated API clients will need to add an `Authorization: Bearer <token>` header

## Capabilities

### New Capabilities
- `api-auth`: Bearer-token authentication middleware, token generation, and token configuration
- `api-tls`: TLS termination for the HTTP server with cert/key file configuration

### Modified Capabilities

_(none — no existing spec requirements change)_

## Impact

- **Code**: `internal/api/` (new auth middleware, TLS listener), `internal/config/` (new fields), `cmd/dashcap/main.go` (new CLI flags)
- **APIs**: All endpoints except `/api/v1/health` require `Authorization: Bearer <token>` header when auth is enabled
- **Dependencies**: Go stdlib only (`crypto/rand`, `crypto/tls`, `encoding/hex`) — no new external dependencies
- **Systems**: Operators need to supply TLS cert/key files for HTTPS; token printed to stderr at startup for scripted retrieval
