## ADDED Requirements

### Requirement: Trigger debouncing
The `Dispatcher` SHALL reject trigger requests that arrive within the debounce interval (5 seconds) of the last accepted trigger. The debounce interval SHALL be defined as a package-level constant `DefaultDebounceInterval`.

#### Scenario: Trigger within debounce window is rejected
- **WHEN** a trigger is requested less than 5 seconds after the last accepted trigger
- **THEN** the Dispatcher returns `ErrDebounced` and does not create a new trigger record

#### Scenario: Trigger after debounce window is accepted
- **WHEN** a trigger is requested 5 or more seconds after the last accepted trigger
- **THEN** the Dispatcher accepts the trigger and processes it normally

#### Scenario: First trigger is always accepted
- **WHEN** no previous trigger has been fired in the current session
- **THEN** the Dispatcher accepts the trigger without debounce checks

### Requirement: HTTP 429 on debounced trigger
The API handler for `POST /api/v1/trigger` SHALL return HTTP 429 (Too Many Requests) when the Dispatcher returns `ErrDebounced`. The response SHALL include a `Retry-After` header indicating the number of seconds until the next trigger will be accepted, and a JSON error body.

#### Scenario: API returns 429 with Retry-After
- **WHEN** a POST to `/api/v1/trigger` is debounced by the Dispatcher
- **THEN** the server responds with HTTP 429, a `Retry-After` header with remaining cooldown seconds (rounded up), and a JSON body `{"error": "trigger debounced, retry after N seconds"}`
