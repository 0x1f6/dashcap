## ADDED Requirements

### Requirement: Dedicated system user
The project SHALL define a `dashcap` system user with no login shell (`/usr/sbin/nologin`) and no home directory, used exclusively for running the dashcap service.

#### Scenario: User properties
- **WHEN** the `dashcap` user is created
- **THEN** it is a system account with shell `/usr/sbin/nologin`, no home directory, and primary group `dashcap`

### Requirement: Dedicated system group
The project SHALL define a `dashcap` system group. Members of this group SHALL be able to read the API token file and trigger captures via `dashcap client trigger`.

#### Scenario: Operator added to group
- **WHEN** an operator runs `sudo usermod -aG dashcap alice`
- **THEN** user `alice` can read `/etc/dashcap/api-token` and use `dashcap client trigger`

#### Scenario: Non-member cannot trigger
- **WHEN** a user who is not in the `dashcap` group attempts to read `/etc/dashcap/api-token`
- **THEN** the read fails with permission denied

### Requirement: sysusers.d drop-in
The project SHALL include a `dist/dashcap.sysusers` file in sysusers.d format that declares the `dashcap` user and group for automatic creation by `systemd-sysusers`.

#### Scenario: Package install creates user
- **WHEN** `systemd-sysusers` processes the drop-in during package installation
- **THEN** the `dashcap` system user and group are created

### Requirement: tmpfiles.d drop-in
The project SHALL include a `dist/dashcap.tmpfiles` file in tmpfiles.d format that declares `/etc/dashcap` (mode `0750`, owner `root:dashcap`), `/run/dashcap` (mode `0750`, owner `dashcap:dashcap`), and `/var/lib/dashcap` (mode `0750`, owner `dashcap:dashcap`).

#### Scenario: Directories created with correct permissions
- **WHEN** `systemd-tmpfiles --create` processes the drop-in
- **THEN** all three directories exist with the specified ownership and mode
