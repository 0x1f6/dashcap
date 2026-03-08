## ADDED Requirements

### Requirement: Install-service subcommand
The CLI SHALL provide a `dashcap install-service` subcommand that installs all systemd service files for standalone binary deployments. It SHALL require root privileges and be idempotent.

#### Scenario: Successful installation
- **WHEN** `sudo dashcap install-service` is run on a Linux system with systemd
- **THEN** the following are created/updated:
  - `dashcap` system user and group (if not present)
  - `/etc/systemd/system/dashcap@.service` (template unit)
  - `/usr/lib/sysusers.d/dashcap.conf` (sysusers drop-in)
  - `/usr/lib/tmpfiles.d/dashcap.conf` (tmpfiles drop-in)
  - `/etc/dashcap/` directory with `0750 root:dashcap`
  - `systemctl daemon-reload` is executed

#### Scenario: Not running as root
- **WHEN** `dashcap install-service` is run without root privileges
- **THEN** the command exits with an error message indicating root is required

#### Scenario: Idempotent re-run
- **WHEN** `sudo dashcap install-service` is run and all files already exist
- **THEN** files are overwritten with current versions, user/group are left unchanged, and the command succeeds

#### Scenario: Non-systemd system
- **WHEN** `dashcap install-service` is run on a system without systemd
- **THEN** the command exits with an error message indicating systemd is required

### Requirement: Embedded service files
The `dashcap install-service` command SHALL use `go:embed` to embed the service files from the `dist/` directory into the binary, so no external files are needed at install time.

#### Scenario: Binary is self-contained
- **WHEN** `dashcap install-service` is run from a standalone binary
- **THEN** all service files are extracted from the embedded data without requiring the `dist/` directory

### Requirement: Post-install instructions
After successful installation, the command SHALL print next-step instructions showing how to enable and start the service for a specific interface.

#### Scenario: Instructions after install
- **WHEN** installation completes successfully
- **THEN** output includes commands like `systemctl enable --now dashcap@<interface>` and instructions for adding users to the `dashcap` group

### Requirement: Uninstall guidance
The `dashcap install-service --help` output SHALL document the manual steps to remove the service (stop, disable, remove files, remove user).

#### Scenario: Help shows uninstall steps
- **WHEN** user runs `dashcap install-service --help`
- **THEN** output includes a section describing how to uninstall the service
