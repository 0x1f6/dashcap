## ADDED Requirements

### Requirement: Token init subcommand
The CLI SHALL provide a `dashcap token-init` subcommand that initializes the API token file. It SHALL generate a new cryptographically random token only if the file does not exist or is empty, write it to `/etc/dashcap/api-token` with permissions `0640` and ownership `root:dashcap`, and exit.

#### Scenario: First run — no token file exists
- **WHEN** `dashcap token-init` is run and `/etc/dashcap/api-token` does not exist
- **THEN** a new 64-character hex token is generated and written to the file with `0640 root:dashcap`

#### Scenario: Token file already exists with content
- **WHEN** `dashcap token-init` is run and `/etc/dashcap/api-token` already contains a token
- **THEN** the file is left unchanged and the command exits with code 0

#### Scenario: Token file exists but is empty
- **WHEN** `dashcap token-init` is run and `/etc/dashcap/api-token` exists but is empty
- **THEN** a new token is generated and written to the file

#### Scenario: Parent directory does not exist
- **WHEN** `dashcap token-init` is run and `/etc/dashcap/` does not exist
- **THEN** the directory is created with `0750 root:dashcap` before writing the token

#### Scenario: Not running as root
- **WHEN** `dashcap token-init` is run without root privileges
- **THEN** the command exits with an error message indicating root is required

### Requirement: Server reads token from file
The daemon SHALL include the token file `/etc/dashcap/api-token` in its token resolution chain. The full chain SHALL be: `--api-token` flag → `DASHCAP_API_TOKEN` env → token file → auto-generate (with warning).

#### Scenario: Token from file
- **WHEN** the daemon starts without `--api-token` flag and without `DASHCAP_API_TOKEN` env
- **AND** `/etc/dashcap/api-token` contains a valid token
- **THEN** the daemon uses the token from the file

#### Scenario: Flag takes precedence over file
- **WHEN** `--api-token mytoken` is passed and a token file exists
- **THEN** the daemon uses `mytoken` from the flag

#### Scenario: No token file available
- **WHEN** no flag, no env var, and no token file exists (or is unreadable)
- **THEN** the daemon auto-generates a token and logs it to stderr with its value

### Requirement: Secure token logging
The daemon SHALL NOT log the token value when it originates from a flag, environment variable, or file. It SHALL only log the source mechanism. The token value SHALL only be logged when it is auto-generated as a last-resort fallback (since the operator has no other way to retrieve it).

#### Scenario: Token from flag — no value logged
- **WHEN** `--api-token mytoken` is passed
- **THEN** the log contains `source=flag` but does NOT contain the token value

#### Scenario: Token from env — no value logged
- **WHEN** `DASHCAP_API_TOKEN` env var provides the token
- **THEN** the log contains `source=env` but does NOT contain the token value

#### Scenario: Token from file — no value logged
- **WHEN** token is read from the token file
- **THEN** the log contains `source=file` and the file path but does NOT contain the token value

#### Scenario: Auto-generated token — value logged
- **WHEN** no flag, env var, or file provides a token and one is auto-generated
- **THEN** the log contains the generated token value (WARN level) so the operator can retrieve it

### Requirement: Token file path configurability
The token file path SHALL default to `/etc/dashcap/api-token` but SHALL be overridable via `--token-file` flag and `api.token_file` in the YAML config.

#### Scenario: Custom token file path
- **WHEN** `--token-file /opt/dashcap/token` is passed
- **THEN** the daemon reads the token from `/opt/dashcap/token`
