## ADDED Requirements

### Requirement: Exclusion filter configuration via YAML
The system SHALL support an `exclusions` list in the YAML configuration file. Each entry SHALL have a `name` (string, for identification/logging) and a `filter` (string, BPF expression in tcpdump syntax).

#### Scenario: Valid exclusions in config file
- **WHEN** the config file contains an `exclusions` list with valid BPF expressions
- **THEN** dashcap SHALL combine them into a single negated BPF filter and apply it to the capture source at startup

#### Scenario: Empty exclusions list
- **WHEN** the config file contains an empty `exclusions` list or no `exclusions` key
- **THEN** dashcap SHALL capture all traffic (no BPF filter applied)

#### Scenario: Exclusion with empty name
- **WHEN** an exclusion entry has an empty `name` field
- **THEN** dashcap SHALL reject the configuration and exit with an error indicating the exclusion name is required

#### Scenario: Exclusion with empty filter
- **WHEN** an exclusion entry has an empty `filter` field
- **THEN** dashcap SHALL reject the configuration and exit with an error indicating the exclusion filter expression is required

### Requirement: Exclusion filter via CLI flag
The system SHALL support an `--exclude` CLI flag that accepts a single BPF expression. This flag SHALL take precedence over config file exclusions and be merged with them.

#### Scenario: CLI exclude flag provided
- **WHEN** the user passes `--exclude "host 10.0.0.50"`
- **THEN** dashcap SHALL add an exclusion named `cli` with that filter expression, merged with any config file exclusions

#### Scenario: CLI exclude flag combined with config file exclusions
- **WHEN** the user passes `--exclude "port 443"` and the config file has exclusions `[{name: dns, filter: "udp port 53"}]`
- **THEN** the combined BPF filter SHALL be `not (port 443) and not (udp port 53)`

### Requirement: BPF filter compilation and application
The system SHALL combine all configured exclusions into a single BPF expression of the form `not (<expr1>) and not (<expr2>) and ...` and apply it to the capture source before the capture loop starts.

#### Scenario: Single exclusion filter
- **WHEN** one exclusion is configured with filter `host 10.0.0.50 and port 443`
- **THEN** the compiled BPF filter SHALL be `not (host 10.0.0.50 and port 443)`

#### Scenario: Multiple exclusion filters
- **WHEN** two exclusions are configured with filters `host 10.0.0.50` and `udp port 53`
- **THEN** the compiled BPF filter SHALL be `not (host 10.0.0.50) and not (udp port 53)`

#### Scenario: Filter applied at kernel level
- **WHEN** a BPF filter is compiled and applied
- **THEN** excluded packets SHALL be filtered at the kernel level before reaching userspace, preserving ring buffer space for relevant traffic

### Requirement: BPF filter validation at startup
The system SHALL validate all BPF filter expressions at startup. If any expression is invalid, dashcap SHALL refuse to start and exit with an error message identifying the invalid exclusion.

#### Scenario: Invalid BPF syntax in a named exclusion
- **WHEN** an exclusion named `bad_rule` has filter `invalid syntax !!!`
- **THEN** dashcap SHALL exit with an error message containing the exclusion name `bad_rule` and the BPF compilation error

#### Scenario: All exclusions valid
- **WHEN** all configured exclusion filters have valid BPF syntax
- **THEN** dashcap SHALL proceed with startup and apply the combined filter

### Requirement: Active filter exposed in status API
The `GET /api/v1/status` response SHALL include a `bpf_filter` field containing the active combined BPF filter expression as a string.

#### Scenario: No exclusions configured
- **WHEN** no exclusions are configured
- **THEN** the `bpf_filter` field in the status response SHALL be an empty string `""`

#### Scenario: Exclusions configured
- **WHEN** exclusions are configured and applied
- **THEN** the `bpf_filter` field SHALL contain the full combined BPF expression (e.g., `"not (host 10.0.0.50) and not (udp port 53)"`)

### Requirement: Exclusion logging at startup
The system SHALL log each configured exclusion at startup for operational visibility.

#### Scenario: Exclusions logged at startup
- **WHEN** exclusions are configured
- **THEN** dashcap SHALL log each exclusion name and filter expression at INFO level, and log the final combined BPF expression
