## 1. Configuration

- [x] 1.1 Add auth and TLS fields to `config.Config`: `APIToken string`, `APINoAuth bool`, `TLSCert string`, `TLSKey string`
- [x] 1.2 Add validation: `TLSCert` and `TLSKey` must both be set or both empty
- [x] 1.3 Add CLI flags in `cmd/dashcap/main.go`: `--api-token`, `--no-auth`, `--tls-cert`, `--tls-key`
- [x] 1.4 Add `DASHCAP_API_TOKEN` environment variable support with precedence: CLI flag > env var > auto-generated

## 2. Token Generation

- [x] 2.1 Implement token generation function: 32 bytes from `crypto/rand`, hex-encoded to 64 chars
- [x] 2.2 Integrate token resolution in startup: apply precedence logic (flag > env > generated), print token to stderr
- [x] 2.3 Skip token generation when `--no-auth` is set

## 3. Auth Middleware

- [x] 3.1 Implement bearer token auth middleware as `http.Handler` wrapper in `internal/api/`
- [x] 3.2 Exempt `GET /api/v1/health` from authentication
- [x] 3.3 Use `crypto/subtle.ConstantTimeCompare` for token validation
- [x] 3.4 Return `401` with `{"error": "unauthorized"}` for missing/invalid tokens
- [x] 3.5 Wire middleware into `api.New()` — wrap `ServeMux` when auth is enabled, skip when `--no-auth`

## 4. TLS Support

- [x] 4.1 Update `ListenAndServe` to use `tls.LoadX509KeyPair` and `srv.ServeTLS` when cert/key are configured
- [x] 4.2 Log warning when auth is enabled without TLS: `WARNING: API auth enabled without TLS — tokens sent in cleartext`

## 5. Tests

- [x] 5.1 Unit tests for token generation (length, randomness, hex format)
- [x] 5.2 Unit tests for auth middleware (valid token, invalid token, missing header, health exemption)
- [x] 5.3 Unit tests for config validation (TLS cert/key pairing)
- [x] 5.4 Integration test: authenticated request flow via `httptest.Server`
