## ADDED Requirements

### Requirement: Load configuration from YAML file
The system SHALL load configuration from a YAML file when one is available. The YAML file maps to the existing runtime config fields using the structure defined in `configs/dashcap.example.yaml`. If a config file is loaded, the system SHALL log the file path at info level.

#### Scenario: Explicit config file via --config flag
- **WHEN** the user starts dashcap with `--config /path/to/config.yaml`
- **THEN** the system SHALL read and parse the YAML file at that path
- **THEN** the parsed values SHALL be applied to the runtime configuration

#### Scenario: Explicit config file does not exist
- **WHEN** the user starts dashcap with `--config /nonexistent.yaml`
- **THEN** the system SHALL exit with an error message indicating the file was not found

#### Scenario: Invalid YAML syntax
- **WHEN** the config file contains invalid YAML syntax
- **THEN** the system SHALL exit with an error message including the parse error details

#### Scenario: Unknown keys in config file
- **WHEN** the config file contains keys that do not map to any config field
- **THEN** the system SHALL exit with an error message identifying the unknown key

### Requirement: Platform-specific default config path discovery
The system SHALL search for a config file at platform-specific default paths when `--config` is not specified. If no default config file is found, the system SHALL proceed with defaults and CLI flags only (no error).

#### Scenario: Default config file exists on Linux/macOS
- **WHEN** `--config` is not specified
- **AND** the file `/etc/dashcap/dashcap.yaml` exists
- **THEN** the system SHALL load configuration from that file

#### Scenario: Default config file exists on Windows
- **WHEN** `--config` is not specified
- **AND** the file `C:\ProgramData\dashcap\dashcap.yaml` exists
- **THEN** the system SHALL load configuration from that file

#### Scenario: No default config file found
- **WHEN** `--config` is not specified
- **AND** no config file exists at the platform default path
- **THEN** the system SHALL proceed using hardcoded defaults and CLI flags only

### Requirement: CLI flags override config file values
CLI flags explicitly set by the user SHALL take precedence over values from the config file. Config file values SHALL take precedence over hardcoded defaults. The full precedence order is: CLI flags > config file > hardcoded defaults.

#### Scenario: CLI flag overrides config file value
- **WHEN** the config file sets `api.tcp_port: 8080`
- **AND** the user passes `--api-port 9900`
- **THEN** the runtime API port SHALL be `9900`

#### Scenario: Config file overrides hardcoded default
- **WHEN** the config file sets `buffer.size: 5GB`
- **AND** the user does not pass `--buffer-size`
- **THEN** the runtime buffer size SHALL be `5GB` (5368709120 bytes)

#### Scenario: CLI flag at default value does not override config file
- **WHEN** the config file sets `api.tcp_port: 8080`
- **AND** the user does not pass `--api-port` (Cobra default is 9800)
- **THEN** the runtime API port SHALL be `8080` (from config file, not the Cobra default)

### Requirement: Human-readable size strings in YAML
The YAML config file SHALL support the same human-readable size suffixes as the CLI flags: `KB`, `MB`, `GB`, `TB`. Plain integer values (in bytes) SHALL also be accepted.

#### Scenario: Size with MB suffix
- **WHEN** the config file contains `buffer.segment_size: 50MB`
- **THEN** the runtime segment size SHALL be `52428800` bytes

#### Scenario: Size as plain bytes
- **WHEN** the config file contains `buffer.size: 1073741824`
- **THEN** the runtime buffer size SHALL be `1073741824` bytes

### Requirement: Duration strings in YAML
The YAML config file SHALL support Go-style duration strings (e.g., `5m`, `30s`, `1h`) for duration fields.

#### Scenario: Duration with minute suffix
- **WHEN** the config file contains `trigger.default_duration: 10m`
- **THEN** the runtime default duration SHALL be 10 minutes

### Requirement: Config validation after merge
After merging defaults, config file, and CLI flags, the system SHALL validate the final configuration using the existing `Config.Validate()` method. Validation errors SHALL cause the system to exit with a descriptive error message.

#### Scenario: Config file sets invalid buffer ratio
- **WHEN** the config file sets `buffer.size: 100MB` and `buffer.segment_size: 200MB`
- **THEN** the system SHALL exit with a validation error (`buffer_size must be >= segment_size`)
