## ADDED Requirements

### Requirement: TLS support via cert and key files
The API server SHALL support TLS when both `--tls-cert` and `--tls-key` flags are provided. The server SHALL load the certificate and key using `tls.LoadX509KeyPair` and serve HTTPS instead of HTTP.

#### Scenario: TLS enabled with valid cert and key
- **WHEN** dashcap starts with `--tls-cert /path/to/cert.pem --tls-key /path/to/key.pem` and both files are valid
- **THEN** the API server SHALL listen for HTTPS connections using the provided certificate

#### Scenario: TLS cert without key
- **WHEN** dashcap starts with `--tls-cert /path/to/cert.pem` but without `--tls-key`
- **THEN** dashcap SHALL exit with an error: `--tls-cert and --tls-key must both be set`

#### Scenario: TLS key without cert
- **WHEN** dashcap starts with `--tls-key /path/to/key.pem` but without `--tls-cert`
- **THEN** dashcap SHALL exit with an error: `--tls-cert and --tls-key must both be set`

#### Scenario: No TLS flags
- **WHEN** dashcap starts without `--tls-cert` and `--tls-key`
- **THEN** the API server SHALL listen for plain HTTP connections

### Requirement: Warning when auth enabled without TLS
The server SHALL log a warning when authentication is enabled but TLS is not configured, since bearer tokens would be transmitted in cleartext.

#### Scenario: Auth enabled without TLS
- **WHEN** dashcap starts with auth enabled (default) and without TLS flags
- **THEN** the server SHALL log a warning: `WARNING: API auth enabled without TLS — tokens sent in cleartext`

#### Scenario: Auth enabled with TLS
- **WHEN** dashcap starts with auth enabled and with valid TLS configuration
- **THEN** the server SHALL NOT log the cleartext warning

#### Scenario: Auth disabled without TLS
- **WHEN** dashcap starts with `--no-auth` and without TLS flags
- **THEN** the server SHALL NOT log the cleartext warning
