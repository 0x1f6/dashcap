## ADDED Requirements

### Requirement: Leveled log output
The server SHALL support two log levels: **info** and **debug**. The default level SHALL be info. When running at info level, debug-level messages SHALL NOT appear in output.

#### Scenario: Default log level is info
- **WHEN** the server starts without a `--debug` flag
- **THEN** the log level SHALL be info and debug messages SHALL be suppressed

#### Scenario: Debug log level via flag
- **WHEN** the server starts with `--debug`
- **THEN** the log level SHALL be debug and both info and debug messages SHALL appear in output

### Requirement: Action events logged at info level
The server SHALL log the following events at info level:
- Server startup (interface, port, buffer size)
- API server listen address
- API token generation
- Incoming API requests (method, path, status)
- Trigger fired (source, duration/since parameters)
- Trigger completed (output path, segment count)
- Trigger failed (error reason)
- Signal received (signal type)
- Server shutdown

#### Scenario: Trigger event logged at info level
- **WHEN** a trigger is fired via API or signal
- **THEN** an info-level log entry SHALL be emitted with the trigger source and requested time window

#### Scenario: API request logged at info level
- **WHEN** an API request is received
- **THEN** an info-level log entry SHALL be emitted with the HTTP method, path, and response status code

#### Scenario: Server startup logged at info level
- **WHEN** the server starts successfully
- **THEN** info-level log entries SHALL be emitted showing the capture interface, API listen address, and buffer configuration

### Requirement: Internal events logged at debug level
The server SHALL log the following events at debug level:
- Ring segment rotation (current segment index, total segments)
- Segment completion (bytes written, packets counted)
- Ring buffer wrap-around (oldest segment overwritten)
- Packet write errors (individual packet level)
- Segment pre-allocation progress
- Capture window segment selection details

#### Scenario: Ring rotation logged at debug level
- **WHEN** the ring buffer rotates to the next segment
- **THEN** a debug-level log entry SHALL be emitted with the new segment index

#### Scenario: Ring rotation not visible at info level
- **WHEN** the ring buffer rotates and the log level is info
- **THEN** no log entry about the rotation SHALL appear in output

### Requirement: Debug flag in CLI
The server SHALL accept a `--debug` command-line flag that sets the log level to debug.

#### Scenario: Debug flag accepted
- **WHEN** the server is invoked with `dashcap --debug capture ...`
- **THEN** the server SHALL start with debug-level logging enabled

#### Scenario: Debug flag absent
- **WHEN** the server is invoked without `--debug`
- **THEN** the server SHALL start with info-level logging (default)
