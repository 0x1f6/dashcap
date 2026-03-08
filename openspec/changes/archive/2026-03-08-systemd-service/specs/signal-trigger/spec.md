## ADDED Requirements

### Requirement: SIGUSR1 trigger handler
The daemon SHALL register a handler for `SIGUSR1` that triggers a default-duration capture save with source `"signal"`. The handler SHALL respect the existing trigger debouncing logic.

#### Scenario: Signal triggers capture save
- **WHEN** `SIGUSR1` is sent to the dashcap process
- **THEN** a capture save is initiated with source `"signal"` and default duration

#### Scenario: Signal trigger appears in history
- **WHEN** a SIGUSR1-triggered save completes
- **THEN** it appears in `GET /api/v1/triggers` with source `"signal"`

#### Scenario: Signal respects debounce
- **WHEN** `SIGUSR1` is sent twice within the 5-second debounce interval
- **THEN** the second signal is ignored and a debug log message is emitted

#### Scenario: Signal combined with systemctl
- **WHEN** operator runs `systemctl kill --signal=USR1 dashcap@eth0`
- **THEN** the same SIGUSR1 handler fires and triggers a capture save

### Requirement: SIGUSR1 does not interfere with shutdown
The SIGUSR1 handler SHALL NOT interfere with SIGTERM/SIGINT shutdown handling. Both signal types SHALL be handled independently.

#### Scenario: Shutdown during signal trigger
- **WHEN** SIGTERM is received while a SIGUSR1-triggered save is in progress
- **THEN** shutdown proceeds normally; the in-progress save may complete or be aborted
