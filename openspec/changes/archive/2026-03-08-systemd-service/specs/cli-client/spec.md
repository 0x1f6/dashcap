## MODIFIED Requirements

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
