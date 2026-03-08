## Why

dashcap is ready for production deployment on Linux (Phase 1 & 2 complete). To run as a reliable system service, it needs proper systemd integration, a dedicated system user with least-privilege access, and a clear access-control model so that only authorized users can trigger captures. This is the foundation for packaging as RPM/DEB where the service is set up automatically on install — but must also work for standalone binary deployments via a self-install CLI flag.

## What Changes

- **systemd template unit** (`dashcap@.service`) for per-interface instances, with `sd_notify` readiness signaling, capability-based security (no root), and hardened sandboxing directives
- **Dedicated `dashcap` system user and group** — daemon runs as `dashcap:dashcap` with `CAP_NET_RAW`/`CAP_NET_ADMIN` granted via `AmbientCapabilities`
- **API token file with group-based access control** — token persisted at `/etc/dashcap/api-token` with `0640 root:dashcap` permissions; users in the `dashcap` group can read the token and use `dashcap client trigger`
- **SIGUSR1 signal trigger** — sending `SIGUSR1` to the dashcap process triggers a default-duration capture save; accessible via `systemctl kill --signal=USR1 dashcap@<iface>` (requires systemd/polkit privileges)
- **`sd_notify` integration** — report `READY=1` after ring buffer pre-allocation and capture start, enabling `Type=notify` for accurate service readiness
- **sysusers.d / tmpfiles.d drop-ins** — declarative system user creation and runtime directory setup, consumed by RPM/DEB packaging
- **`dashcap install-service` CLI command** — for standalone binary deployments (no package manager): checks if systemd unit exists, creates `dashcap` user/group if missing, installs unit + tmpfiles + sysusers configs, and runs `systemctl daemon-reload`. Requires root. Idempotent — safe to re-run.

## Capabilities

### New Capabilities
- `systemd-unit`: systemd template unit file, sd_notify integration, service hardening directives
- `system-user`: dedicated dashcap user/group, sysusers.d config, directory ownership
- `token-file`: API token persistence to file with group-based read access for CLI trigger authorization
- `signal-trigger`: SIGUSR1 handler for default-duration capture trigger
- `install-service`: `dashcap install-service` CLI command that installs systemd unit, sysusers/tmpfiles configs, and creates user/group for standalone deployments

### Modified Capabilities
- `cli-client`: read API token from file (`/etc/dashcap/api-token`) as fallback when `--token` and `DASHCAP_API_TOKEN` are not set

## Impact

- **New files**: `dist/dashcap@.service`, `dist/dashcap.sysusers`, `dist/dashcap.tmpfiles`, signal handler in `cmd/dashcap/`, install-service command
- **Modified code**: `cmd/dashcap/main.go` (sd_notify, signal handler, token-file write), `internal/client/client.go` (token-file fallback)
- **New dependency**: `github.com/coreos/go-systemd/v22/daemon` for sd_notify (build-tag gated to Linux)
- **Packaging**: sysusers.d/tmpfiles.d files ready for consumption by RPM spec / DEB maintainer scripts
- **Security model**: principle of least privilege — no root at runtime, group membership controls trigger access
