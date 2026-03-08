## ADDED Requirements

### Requirement: systemd template unit file
The project SHALL include a systemd template unit file `dashcap@.service` in the `dist/` directory that allows per-interface instances via `systemctl start dashcap@<interface>`.

#### Scenario: Start service for eth0
- **WHEN** operator runs `systemctl start dashcap@eth0`
- **THEN** dashcap starts capturing on interface `eth0`

#### Scenario: Multiple interfaces
- **WHEN** operator runs `systemctl start dashcap@eth0` and `systemctl start dashcap@eth1`
- **THEN** two independent dashcap instances run, one per interface

### Requirement: Service type notify
The unit file SHALL use `Type=notify`. The dashcap daemon SHALL send `sd_notify(READY=1)` after ring buffer pre-allocation, capture source open, and BPF filter application are complete — immediately before entering the capture loop.

#### Scenario: Readiness signal timing
- **WHEN** dashcap starts as a systemd service
- **THEN** `systemctl start` blocks until ring pre-allocation is complete and returns only after `READY=1` is sent

#### Scenario: Not running under systemd
- **WHEN** dashcap is started directly (not via systemd) and `NOTIFY_SOCKET` is not set
- **THEN** the `sd_notify` call is a silent no-op and startup proceeds normally

### Requirement: Capability-based security
The unit file SHALL use `User=dashcap`, `Group=dashcap`, and `AmbientCapabilities=CAP_NET_RAW CAP_NET_ADMIN` so the daemon runs without root privileges.

#### Scenario: Process runs as dashcap user
- **WHEN** the service is started via systemd
- **THEN** the process UID/GID is `dashcap:dashcap`, not root

#### Scenario: Packet capture works without root
- **WHEN** the service runs as `dashcap` user with ambient capabilities
- **THEN** libpcap opens the interface successfully

### Requirement: Service hardening
The unit file SHALL include sandboxing directives: `ProtectSystem=strict`, `ProtectHome=true`, `PrivateTmp=true`, `NoNewPrivileges=true`, `ProtectKernelTunables=true`, `ProtectControlGroups=true`, `RestrictNamespaces=true`, `LockPersonality=true`, `MemoryDenyWriteExecute=true`, `RestrictRealtime=true`.

#### Scenario: Filesystem isolation
- **WHEN** the service is running
- **THEN** the process cannot write to any filesystem path except those listed in `ReadWritePaths=`

### Requirement: Directory management via systemd
The unit file SHALL use `RuntimeDirectory=dashcap` and `StateDirectory=dashcap/%i` to have systemd create and manage `/run/dashcap/` and `/var/lib/dashcap/<interface>/` with correct ownership.

#### Scenario: Directories created on start
- **WHEN** the service starts for the first time
- **THEN** `/run/dashcap/` and `/var/lib/dashcap/<interface>/` exist and are owned by `dashcap:dashcap`

### Requirement: Token initialization via ExecStartPre
The unit file SHALL include an `ExecStartPre=+/usr/bin/dashcap token-init` line (with `+` prefix for root execution) that generates an API token if `/etc/dashcap/api-token` does not exist, and sets file permissions to `0640 root:dashcap`.

#### Scenario: First start generates token
- **WHEN** service starts and `/etc/dashcap/api-token` does not exist
- **THEN** a new token is generated, written to the file with `0640 root:dashcap` permissions

#### Scenario: Subsequent starts preserve token
- **WHEN** service starts and `/etc/dashcap/api-token` already exists with content
- **THEN** the existing token is preserved unchanged

### Requirement: Graceful shutdown
The unit file SHALL set `KillSignal=SIGTERM` and `TimeoutStopSec=10`. The daemon SHALL handle SIGTERM by flushing the active segment and shutting down the API server.

#### Scenario: Clean stop
- **WHEN** `systemctl stop dashcap@eth0` is executed
- **THEN** dashcap flushes the active segment, stops the API server, and exits with code 0 within 10 seconds
