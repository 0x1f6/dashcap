## Context

dashcap Phase 1 & 2 are complete. The daemon currently runs in the foreground, requires manual token management, and has no formal service integration. Signal handling is limited to SIGTERM/SIGINT for shutdown (`cmd/dashcap/main.go:301-303`). The token is either auto-generated (printed to stderr) or set via `--api-token`/`DASHCAP_API_TOKEN`. There is no persistent token storage and no Unix user isolation.

DESIGN.md Section 8.5 sketches a systemd template unit. This change implements it with additions for `sd_notify`, token persistence, SIGUSR1 trigger, and a self-install command.

## Goals / Non-Goals

**Goals:**
- Run dashcap as a properly sandboxed systemd service with `Type=notify` readiness
- Dedicated `dashcap` system user/group with `CAP_NET_RAW`/`CAP_NET_ADMIN` — no root at runtime
- Group-based access control: only `dashcap` group members can trigger via CLI
- Persist auto-generated API token to a file readable by the `dashcap` group
- SIGUSR1 signal trigger for simple default-duration saves (e.g. from cron, scripts, `systemctl kill`)
- Self-install command (`dashcap install-service`) for binary-only deployments without a package manager
- All config files ready for consumption by RPM/DEB packaging

**Non-Goals:**
- Windows Service registration (separate Phase 3 task)
- macOS launchd integration
- Process resource limits (MemoryMax, OOMScoreAdjust) — can be added by operators; not embedded in the unit
- polkit rules for `systemctl kill` — standard systemd permissions are sufficient
- `sd_notify` WATCHDOG support (only READY notification)

## Decisions

### 1. Service type: `Type=notify` with `sd_notify(READY=1)`

**Choice:** Use `Type=notify` and send `READY=1` from Go after ring pre-allocation and capture start.

**Alternatives considered:**
- `Type=simple`: systemd considers the service ready immediately after fork. This is inaccurate — pre-allocation of a multi-GB ring buffer can take seconds. Services depending on dashcap (or operators using `systemctl start --wait`) would not get correct readiness.
- `Type=forking`: Requires daemonization. Unnecessary complexity in Go.

**Implementation:** Use `github.com/coreos/go-systemd/v22/daemon` for `SdNotify`. Guard the import with a build tag (`//go:build linux`) so it doesn't affect Windows/macOS builds. Call `daemon.SdNotify(false, daemon.SdNotifyReady)` in `run()` after ring pre-allocation and before entering the capture loop. When not running under systemd (no `NOTIFY_SOCKET`), the call is a no-op.

### 2. Trigger via SIGUSR1

**Choice:** Register a SIGUSR1 handler that calls `dispatcher.Trigger("signal", TriggerOpts{})` with default duration.

**Rationale:** Signals are the simplest mechanism that doesn't require network access or token management. Limited to default-duration triggers (no parameters), which is acceptable for the emergency/script use case. Combined with systemd, operators can use `systemctl kill --signal=USR1 dashcap@eth0`.

**Alternatives considered:**
- systemd socket activation: Doesn't fit — dashcap is a long-running capture process, not a request-driven service.
- D-Bus: Heavyweight dependency for a simple trigger. Overkill.
- Named pipe / Unix socket: More capable than signals but requires additional protocol. Deferred to Phase 4 (Unix socket API).

**Access control for signals:** Only the `dashcap` user (service account), root, or users with `systemctl kill` privileges (typically via systemd/polkit) can send signals. This is sufficient — the `dashcap` group controls API token access for the richer `dashcap client trigger` path.

### 3. API token file at `/etc/dashcap/api-token`

**Choice:** Server writes the auto-generated token to `/etc/dashcap/api-token` with permissions `0640 root:dashcap`. Client reads this file as a fallback when `--token` and `DASHCAP_API_TOKEN` are not set.

**Token resolution chain (client):** `--token` flag → `DASHCAP_API_TOKEN` env → `/etc/dashcap/api-token` file → error

**Token resolution chain (server):** `--api-token` flag → `DASHCAP_API_TOKEN` env → read existing `/etc/dashcap/api-token` → generate new token and write to file

**Rationale:** This mirrors established patterns (k3s `/etc/rancher/k3s/k3s.yaml`, Docker `/var/run/docker.sock` group access). Group-readable file permissions are the standard Unix access control mechanism. No custom auth system needed.

**Security:** The file is `0640 root:dashcap` — only root and dashcap group members can read it. The server writes it as root during startup (before dropping to dashcap user via systemd `User=`). When running standalone, the operator manages permissions manually.

**When `--no-auth` is set:** No token file is written.

### 4. Secure token logging

**Choice:** Only log the token value when it is auto-generated (last-resort fallback). For all other sources, log only the source mechanism — never the token value itself.

**Current behavior (insecure):** `main.go:206` unconditionally logs `slog.Info("API token generated", "token", cfg.APIToken)` regardless of whether the token came from a flag, env var, or was truly generated. This leaks tokens into log files (journald, stdout redirects), enabling privilege escalation if logs are accessible to non-authorized users.

**New behavior:**
- Token from `--api-token` flag → `slog.Info("API token configured", "source", "flag")`
- Token from `DASHCAP_API_TOKEN` env → `slog.Info("API token configured", "source", "env")`
- Token from file → `slog.Info("API token configured", "source", "file", "path", tokenFilePath)`
- Token auto-generated (fallback) → `slog.Warn("API token auto-generated (no persistent storage)", "token", token)` — this is the only case where the value is logged, because the operator has no other way to retrieve it.

**Rationale:** Tokens from flag/env/file are already known to the operator through their respective mechanisms. Logging them provides no value and creates a log-based credential leak vector. The auto-generated case is the exception: without logging, the token would be irrecoverable.

### 5. Dedicated system user `dashcap:dashcap`

**Choice:** Create a system user `dashcap` with group `dashcap`, no login shell, no home directory.

**Provisioning:**
- **Package install:** sysusers.d drop-in (`dist/dashcap.sysusers`) — consumed by `systemd-sysusers` during package install
- **Standalone install:** `dashcap install-service` creates the user via `useradd --system --no-create-home --shell /usr/sbin/nologin dashcap` (or uses existing)

**Capabilities:** `AmbientCapabilities=CAP_NET_RAW CAP_NET_ADMIN` in the unit file grants packet capture ability without root. `NoNewPrivileges=true` ensures these cannot be escalated.

### 6. `dashcap install-service` subcommand

**Choice:** Add a Cobra subcommand that installs all systemd artifacts and creates the system user. Requires root. Idempotent.

**Steps performed:**
1. Check running as root (UID 0)
2. Create `dashcap` system user/group if not present
3. Copy embedded unit file to `/etc/systemd/system/dashcap@.service`
4. Write sysusers.d config to `/usr/lib/sysusers.d/dashcap.conf`
5. Write tmpfiles.d config to `/usr/lib/tmpfiles.d/dashcap.conf`
6. Create `/etc/dashcap/` directory with `0750 root:dashcap`
7. Run `systemctl daemon-reload`
8. Print next-steps instructions (enable, start)

**Embedded files:** The unit file and configs are embedded in the binary via `go:embed` from the `dist/` directory. No external files needed at install time.

**Idempotency:** Each step checks if the target already exists and skips if current. Re-running is safe.

### 7. Service hardening directives

The unit file includes systemd sandboxing:

```ini
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
NoNewPrivileges=true
RestrictSUIDs=true
ProtectKernelTunables=true
ProtectControlGroups=true
RestrictNamespaces=true
LockPersonality=true
MemoryDenyWriteExecute=true
RestrictRealtime=true
```

`ReadWritePaths=` grants access only to the instance data directory. All other filesystem access is read-only.

### 8. File layout

```
dist/
├── dashcap@.service        # systemd template unit
├── dashcap.sysusers        # sysusers.d drop-in (u dashcap - "dashcap service")
└── dashcap.tmpfiles        # tmpfiles.d drop-in (directories, permissions)

/etc/dashcap/               # Config directory (0750 root:dashcap)
├── dashcap.yaml            # Config file (optional)
└── api-token               # Auto-generated API token (0640 root:dashcap)

/run/dashcap/               # Runtime directory (via RuntimeDirectory=)
└── dashcap-<iface>.lock    # Interface lock file

/var/lib/dashcap/<iface>/   # Data directory (via StateDirectory=)
├── ring/
└── saved/
```

## Risks / Trade-offs

- **[go-systemd dependency]** → Build-tag gated to Linux only. Zero impact on Windows/macOS builds. The library is well-maintained (CoreOS/Red Hat) and has no transitive dependencies beyond Go stdlib.
- **[Token file written as root]** → The server process starts as root briefly (systemd `ExecStartPre=` or early in `ExecStart` before `User=` takes effect). Actually, with `User=dashcap` in the unit, the process never runs as root. **Mitigation:** Use `ExecStartPre=+/usr/bin/dashcap write-token` (the `+` prefix runs as root) or have systemd create the file via tmpfiles.d with a placeholder, then the daemon overwrites with the actual token (since it owns the file via group).
  **Revised approach:** tmpfiles.d creates `/etc/dashcap/api-token` with `0640 root:dashcap`. The daemon (running as `dashcap` user in group `dashcap`) can read but not write it. For auto-generated tokens: the token is generated at startup and written to the file only if the daemon has write permission (standalone mode) or via `ExecStartPre=+` (systemd mode). Alternatively, the config file already supports `api.token`, which can be pre-set by the operator.
  **Final approach:** Keep it simple — the token file is created/updated by `ExecStartPre=+/usr/bin/dashcap token-init` which runs as root, generates a token if none exists, and writes it to the file with correct permissions. The main process then reads the token file at startup.
- **[SIGUSR1 limited to default trigger]** → Acceptable. Parameterized triggers use the API. SIGUSR1 is an emergency/convenience mechanism.
- **[install-service requires root]** → Expected for system service installation. Clear error message if not root.
- **[Embedded files increase binary size]** → Negligible (< 5 KB for all dist/ files).
