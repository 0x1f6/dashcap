## ADDED Requirements

### Requirement: Individual trigger status endpoint
The system SHALL expose `GET /api/v1/trigger/{id}` that returns the current state of a single trigger by its ID. The endpoint SHALL require authentication when API token is configured.

#### Scenario: Completed trigger returns record with metadata
- **WHEN** a GET request is made to `/api/v1/trigger/{id}` with a valid trigger ID whose status is "completed"
- **THEN** the system responds with HTTP 200 and a JSON body containing the trigger record fields (`id`, `timestamp`, `source`, `status`, `saved_path`, `warning`) plus a `metadata` object with the full `TriggerMeta` from `metadata.json`

#### Scenario: Pending trigger returns 202 with retry hint
- **WHEN** a GET request is made to `/api/v1/trigger/{id}` with a valid trigger ID whose status is "pending"
- **THEN** the system responds with HTTP 202 and a JSON body containing the trigger record fields and a `retry_after` field indicating how many seconds the caller should wait before polling again

#### Scenario: Failed trigger returns record with error
- **WHEN** a GET request is made to `/api/v1/trigger/{id}` with a valid trigger ID whose status is "failed"
- **THEN** the system responds with HTTP 200 and a JSON body containing the trigger record fields including the `error` field describing the failure

#### Scenario: Unknown trigger ID returns 404
- **WHEN** a GET request is made to `/api/v1/trigger/{id}` with an ID that does not exist in the trigger history
- **THEN** the system responds with HTTP 404 and a JSON error body

### Requirement: Dispatcher lookup by ID
The `Dispatcher` SHALL provide a `Get(id string)` method that returns a snapshot copy of the `TriggerRecord` for the given ID, or nil if no record with that ID exists.

#### Scenario: Lookup existing trigger
- **WHEN** `Get` is called with an ID that exists in the history
- **THEN** it returns a copy of the corresponding `TriggerRecord`

#### Scenario: Lookup non-existent trigger
- **WHEN** `Get` is called with an ID that does not exist in the history
- **THEN** it returns nil

### Requirement: Metadata enrichment for completed triggers
When a completed trigger has a non-empty `saved_path`, the handler SHALL read `metadata.json` from that directory and include the parsed `TriggerMeta` in the response under the `metadata` key. If the metadata file cannot be read, the response SHALL still return the trigger record without the metadata field.

#### Scenario: Metadata file exists and is valid
- **WHEN** the trigger is completed and `saved_path` contains a valid `metadata.json`
- **THEN** the response includes a `metadata` object with all `TriggerMeta` fields (trigger_id, timestamp, source, interface, durations, time window, stats)

#### Scenario: Metadata file is missing or unreadable
- **WHEN** the trigger is completed but `metadata.json` cannot be read from `saved_path`
- **THEN** the response still returns HTTP 200 with the trigger record fields, and the `metadata` field is omitted
