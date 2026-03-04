## ADDED Requirements

### Requirement: Trigger with custom duration
The trigger API (`POST /api/v1/trigger`) SHALL accept an optional JSON body with a `duration` field (Go duration string, e.g. `"30m"`, `"2h"`) to override the default duration for that trigger call.

#### Scenario: Trigger with explicit duration
- **WHEN** `POST /api/v1/trigger` is called with body `{"duration": "30m"}`
- **THEN** the system SHALL persist ring buffer data covering the last 30 minutes and return HTTP 202 with the trigger record

#### Scenario: Trigger without body uses default duration
- **WHEN** `POST /api/v1/trigger` is called with no request body
- **THEN** the system SHALL persist ring buffer data covering the configured `--default-duration` (formerly `--pre-duration`) and return HTTP 202 with the trigger record

#### Scenario: Trigger with empty JSON object uses default duration
- **WHEN** `POST /api/v1/trigger` is called with body `{}`
- **THEN** the system SHALL persist ring buffer data covering the configured `--default-duration` and return HTTP 202

### Requirement: Trigger with since timestamp
The trigger API SHALL accept an optional `since` field (RFC 3339 timestamp) in the JSON body to persist all data from that timestamp until now.

#### Scenario: Trigger with since timestamp
- **WHEN** `POST /api/v1/trigger` is called with body `{"since": "2025-01-01T12:00:00Z"}`
- **THEN** the system SHALL persist ring buffer data from `2025-01-01T12:00:00Z` until the current time and return HTTP 202

#### Scenario: Trigger with future since timestamp
- **WHEN** `POST /api/v1/trigger` is called with a `since` value that is in the future
- **THEN** the system SHALL return HTTP 400 with an error message indicating the timestamp is in the future

### Requirement: Mutual exclusivity of duration and since
The `duration` and `since` parameters SHALL be mutually exclusive.

#### Scenario: Both duration and since provided
- **WHEN** `POST /api/v1/trigger` is called with body `{"duration": "10m", "since": "2025-01-01T12:00:00Z"}`
- **THEN** the system SHALL return HTTP 400 with an error message indicating that `duration` and `since` are mutually exclusive

### Requirement: Invalid duration rejected
The system SHALL validate the `duration` field and reject invalid values.

#### Scenario: Invalid duration string
- **WHEN** `POST /api/v1/trigger` is called with body `{"duration": "notaduration"}`
- **THEN** the system SHALL return HTTP 400 with an error message indicating the duration is invalid

#### Scenario: Zero duration
- **WHEN** `POST /api/v1/trigger` is called with body `{"duration": "0s"}`
- **THEN** the system SHALL return HTTP 400 with an error message indicating the duration must be positive

### Requirement: Best-effort persistence with warning
When the requested time range exceeds the available ring buffer data, the system SHALL persist all available data and include a warning.

#### Scenario: Requested range partially available
- **WHEN** a trigger is fired requesting the last 60 minutes
- **AND** the ring buffer only contains 20 minutes of data
- **THEN** the system SHALL persist all 20 minutes of available data
- **AND** the trigger record SHALL include a `warning` field indicating that the persisted data is shorter than requested
- **AND** the metadata.json SHALL include `actual_from` and `actual_to` timestamps reflecting the actual persisted range

#### Scenario: No data available at all
- **WHEN** a trigger is fired requesting a time range
- **AND** the ring buffer contains no overlapping segments
- **THEN** the trigger SHALL be marked as `failed` with an appropriate error message

### Requirement: Metadata records effective time range
The persisted `metadata.json` SHALL record the requested and actual time range for auditability.

#### Scenario: Metadata with custom duration
- **WHEN** a trigger completes with `duration: "30m"` and full data is available
- **THEN** `metadata.json` SHALL contain `requested_duration`, `actual_from`, `actual_to`, and `default_duration` fields

#### Scenario: Metadata with default duration
- **WHEN** a trigger completes without explicit time range parameters
- **THEN** `metadata.json` SHALL contain `requested_duration` set to `"default"`, along with `actual_from`, `actual_to`, and `default_duration` fields

### Requirement: Rename pre-duration to default-duration
The CLI flag `--pre-duration` SHALL be renamed to `--default-duration` and the config YAML key `pre_duration` SHALL be renamed to `default_duration`.

#### Scenario: CLI flag renamed
- **WHEN** the user starts dashcap with `--default-duration 10m`
- **THEN** the default trigger duration SHALL be set to 10 minutes

#### Scenario: Old flag no longer accepted
- **WHEN** the user starts dashcap with `--pre-duration 10m`
- **THEN** dashcap SHALL fail with an error indicating the flag has been renamed to `--default-duration`
