## MODIFIED Requirements

### Requirement: Default data directory is platform-specific

The default data directory (when `--data-dir` is not provided) SHALL match the platform table in DESIGN.md §9.

#### Scenario: Linux default data directory

- **GIVEN** dashcap is running on Linux
- **AND** no `--data-dir` flag is provided
- **WHEN** the data directory is resolved
- **THEN** the base path is `/var/lib/dashcap/`

#### Scenario: Windows default data directory

- **GIVEN** dashcap is running on Windows
- **AND** no `--data-dir` flag is provided
- **WHEN** the data directory is resolved
- **THEN** the base path is `C:\ProgramData\dashcap\`

### Requirement: Lock directory is platform-specific

The lock file directory SHALL match the platform table in DESIGN.md §8.2.

#### Scenario: Linux lock directory

- **GIVEN** dashcap is running on Linux
- **WHEN** the interface lock is acquired
- **THEN** the lock file is created under `/run/dashcap/`

#### Scenario: Windows lock directory

- **GIVEN** dashcap is running on Windows
- **WHEN** the interface lock is acquired
- **THEN** the lock file is created under `C:\ProgramData\dashcap\locks\`

## ADDED Requirements

### Requirement: Platform default path functions exist in storage package

The `internal/storage` package SHALL expose `DefaultDataDir()` and `DefaultLockDir()` as package-level functions that return the correct path for the current platform.

#### Scenario: DefaultDataDir returns platform-correct path

- **GIVEN** the code is compiled for a target platform
- **WHEN** `storage.DefaultDataDir()` is called
- **THEN** it returns the data directory path defined in DESIGN.md §9 for that platform

#### Scenario: DefaultLockDir returns platform-correct path

- **GIVEN** the code is compiled for a target platform
- **WHEN** `storage.DefaultLockDir()` is called
- **THEN** it returns the lock directory path defined in DESIGN.md §8.2 for that platform

### Requirement: main.go uses no hardcoded platform paths

`cmd/dashcap/main.go` SHALL NOT contain literal path strings for data or lock directories. All default paths SHALL come from `storage.DefaultDataDir()` and `storage.DefaultLockDir()`.

#### Scenario: No hardcoded Unix paths in main.go

- **GIVEN** the source code of `cmd/dashcap/main.go`
- **THEN** the strings `/var/lib/dashcap` and `/run/dashcap` do not appear as literals
