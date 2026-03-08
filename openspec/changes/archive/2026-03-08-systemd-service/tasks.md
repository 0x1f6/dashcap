## 1. systemd distribution files

- [x] 1.1 Create `dist/dashcap@.service` template unit with Type=notify, User=dashcap, AmbientCapabilities, hardening directives, RuntimeDirectory, StateDirectory, ExecStartPre token-init
- [x] 1.2 Create `dist/dashcap.sysusers` — sysusers.d drop-in declaring `dashcap` system user and group
- [x] 1.3 Create `dist/dashcap.tmpfiles` — tmpfiles.d drop-in for `/etc/dashcap`, `/run/dashcap`, `/var/lib/dashcap`

## 2. sd_notify integration

- [x] 2.1 Add `github.com/coreos/go-systemd/v22` dependency
- [x] 2.2 Create `internal/notify/notify_linux.go` with `Ready()` calling `daemon.SdNotify(false, SdNotifyReady)`, gated with `//go:build linux`
- [x] 2.3 Create `internal/notify/notify_other.go` as no-op stub for non-Linux builds (`//go:build !linux`)
- [x] 2.4 Call `notify.Ready()` in `run()` after ring pre-allocation and capture source open, before capture loop

## 3. SIGUSR1 signal trigger

- [x] 3.1 Add SIGUSR1 to signal channel in `run()` (separate from SIGTERM/SIGINT channel)
- [x] 3.2 Implement signal listener goroutine that calls `dispatcher.Trigger("signal", TriggerOpts{})` on SIGUSR1, logging debounce rejections at debug level
- [x] 3.3 Add test for signal trigger appearing in dispatcher history with source "signal"

## 4. Token file — server side

- [x] 4.1 Add `TokenFile` field to `config.Config` with default `/etc/dashcap/api-token`, `--token-file` CLI flag, and `api.token_file` YAML key
- [x] 4.2 Create `dashcap token-init` Cobra subcommand: generate token if file missing/empty, write with 0640 permissions, create parent dir, require root
- [x] 4.3 Embed `token-init` logic so it can also be called from `ExecStartPre=+`
- [x] 4.4 Update token resolution in `run()`: flag → env → read token file → auto-generate with warning
- [x] 4.5 Fix token logging: log only source mechanism (flag/env/file) without token value; only log the token value when auto-generated as last-resort fallback
- [x] 4.6 Add test for token-init idempotency (existing file preserved, empty file regenerated)

## 5. Token file — client side

- [x] 5.1 Add `--token-file` persistent flag to `clientCmd()` with default `/etc/dashcap/api-token`
- [x] 5.2 Update `clientFlags.newClient()` to read token from file as fallback after env var check
- [x] 5.3 Add test verifying fallback chain: flag → env → file → no token

## 6. install-service command

- [x] 6.1 Embed `dist/` files via `go:embed` in a new `cmd/dashcap/install.go` (Linux-only build tag)
- [x] 6.2 Implement `dashcap install-service` Cobra subcommand: root check, create user/group, write unit + sysusers + tmpfiles, create /etc/dashcap, daemon-reload
- [x] 6.3 Add no-op stub for non-Linux builds that prints "only supported on Linux with systemd"
- [x] 6.4 Print post-install instructions (enable/start commands, group membership)

## 7. Documentation and config updates

- [x] 7.1 Update README.md: add systemd service section, install-service usage, group-based access control, SIGUSR1 trigger
- [x] 7.2 Update DESIGN.md Phase 3 status and section 8.5 to reflect implemented service integration
- [x] 7.3 Update `configs/dashcap.example.yaml` with `api.token_file` field
- [x] 7.4 Update OpenAPI spec with token-init subcommand notes (if applicable)

## 8. Integration verification

- [x] 8.1 Verify `make build` succeeds with new go-systemd dependency on Linux
- [x] 8.2 Verify build without CGO on non-Linux platforms still compiles (no-op notify stubs)
- [x] 8.3 Verify `dashcap install-service` is idempotent in a test environment
- [x] 8.4 Verify SIGUSR1 trigger + API trigger coexistence (both sources in trigger history)
