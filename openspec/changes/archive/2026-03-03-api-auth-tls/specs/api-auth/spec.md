## ADDED Requirements

### Requirement: Bearer token authentication on API endpoints
The API server SHALL require a valid `Authorization: Bearer <token>` header on all endpoints except `GET /api/v1/health`. Requests without a valid token SHALL receive a `401 Unauthorized` response with a JSON body `{"error": "unauthorized"}`.

#### Scenario: Valid token provided
- **WHEN** a client sends a request with `Authorization: Bearer <valid-token>` header
- **THEN** the request SHALL be processed normally by the target endpoint

#### Scenario: Missing Authorization header
- **WHEN** a client sends a request without an `Authorization` header
- **THEN** the server SHALL respond with HTTP 401 and JSON body `{"error": "unauthorized"}`

#### Scenario: Invalid token
- **WHEN** a client sends a request with `Authorization: Bearer <wrong-token>`
- **THEN** the server SHALL respond with HTTP 401 and JSON body `{"error": "unauthorized"}`

#### Scenario: Health endpoint exemption
- **WHEN** a client sends `GET /api/v1/health` without an `Authorization` header
- **THEN** the server SHALL respond with HTTP 200 and `{"status": "ok"}` regardless of auth configuration

### Requirement: Token auto-generation at startup
The server SHALL generate a cryptographically secure random token (32 bytes, hex-encoded to 64 characters) at startup when no user-supplied token is configured. The token SHALL be generated using `crypto/rand`.

#### Scenario: No token configured
- **WHEN** dashcap starts without `--api-token` flag and without `DASHCAP_API_TOKEN` environment variable
- **THEN** the server SHALL generate a random 64-character hex token and print it to stderr

#### Scenario: Token printed at startup
- **WHEN** dashcap starts with auth enabled (default)
- **THEN** the active token SHALL be printed to stderr in the format: `API token: <token>`

### Requirement: User-supplied token via CLI flag
The server SHALL accept a token via `--api-token <value>` CLI flag. This flag SHALL take precedence over the `DASHCAP_API_TOKEN` environment variable.

#### Scenario: Token via CLI flag
- **WHEN** dashcap starts with `--api-token mytoken123`
- **THEN** the server SHALL use `mytoken123` as the API authentication token

#### Scenario: CLI flag overrides environment variable
- **WHEN** `DASHCAP_API_TOKEN=envtoken` is set AND `--api-token clitoken` is provided
- **THEN** the server SHALL use `clitoken` as the API authentication token

### Requirement: User-supplied token via environment variable
The server SHALL accept a token via the `DASHCAP_API_TOKEN` environment variable when no `--api-token` CLI flag is provided.

#### Scenario: Token via environment variable
- **WHEN** `DASHCAP_API_TOKEN=envtoken` is set and no `--api-token` flag is provided
- **THEN** the server SHALL use `envtoken` as the API authentication token

### Requirement: Disable authentication with --no-auth
The server SHALL support a `--no-auth` flag that disables token authentication entirely. When `--no-auth` is set, no token is generated and all requests are processed without authentication.

#### Scenario: Auth disabled
- **WHEN** dashcap starts with `--no-auth`
- **THEN** all API requests SHALL be processed without requiring an `Authorization` header

#### Scenario: Default is auth enabled
- **WHEN** dashcap starts without `--no-auth`
- **THEN** authentication SHALL be enabled and a token SHALL be required

### Requirement: Timing-safe token comparison
Token validation SHALL use constant-time comparison (`crypto/subtle.ConstantTimeCompare`) to prevent timing side-channel attacks.

#### Scenario: Token comparison is constant-time
- **WHEN** the auth middleware compares a request token against the configured token
- **THEN** it SHALL use `crypto/subtle.ConstantTimeCompare` regardless of whether the token matches
