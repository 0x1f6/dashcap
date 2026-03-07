## ADDED Requirements

### Requirement: Release workflow triggers on version tags
The release workflow SHALL trigger only when a tag matching `v*` is pushed.

#### Scenario: Version tag triggers release
- **WHEN** a tag like `v1.0.0` is pushed
- **THEN** the release workflow SHALL execute

#### Scenario: Non-version tag does not trigger release
- **WHEN** a tag not matching `v*` is pushed
- **THEN** the release workflow SHALL NOT execute

### Requirement: Release builds Linux amd64 binary
The release workflow SHALL produce a Linux amd64 binary with version information embedded via LDFLAGS (version, commit, build time).

#### Scenario: Linux binary is built with version info
- **WHEN** the release workflow runs for tag `v1.2.3`
- **THEN** a `dashcap-linux-amd64` binary SHALL be produced
- **AND** the binary SHALL report version `v1.2.3` when executed with `--version`

### Requirement: Release builds Windows amd64 binary
The release workflow SHALL cross-compile a Windows amd64 binary using MinGW and WinPcap/Npcap headers.

#### Scenario: Windows binary is built
- **WHEN** the release workflow runs
- **THEN** a `dashcap-windows-amd64.exe` binary SHALL be produced

### Requirement: Release generates checksums
The release workflow SHALL generate SHA256 checksums for all produced binaries.

#### Scenario: Checksums file is created
- **WHEN** binaries are built
- **THEN** a `checksums.txt` file SHALL be produced containing SHA256 hashes for each binary

### Requirement: Release creates GitHub Release with artifacts
The release workflow SHALL create a GitHub Release for the tag and attach all binaries and the checksums file.

#### Scenario: GitHub Release is created
- **WHEN** the release workflow completes successfully
- **THEN** a GitHub Release SHALL exist for the tag
- **AND** the release SHALL contain the Linux binary, Windows binary, and checksums file

### Requirement: Dependabot keeps dependencies updated
A `.github/dependabot.yml` config SHALL be present to monitor Go modules and GitHub Actions for updates.

#### Scenario: Dependabot monitors Go modules
- **WHEN** a Go module dependency has an update available
- **THEN** Dependabot SHALL create a pull request to update it

#### Scenario: Dependabot monitors GitHub Actions
- **WHEN** a GitHub Actions action has a new version
- **THEN** Dependabot SHALL create a pull request to update it
