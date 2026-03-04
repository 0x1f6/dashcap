## 1. Rename PreDuration → DefaultDuration

- [x] 1.1 Rename `Config.PreDuration` to `Config.DefaultDuration` in `internal/config/config.go` (field, default, comments)
- [x] 1.2 Rename CLI flag `--pre-duration` to `--default-duration` in `cmd/dashcap/main.go`
- [x] 1.3 Update YAML config key from `pre_duration` to `default_duration` in `configs/dashcap.example.yaml`
- [x] 1.4 Update all references to `PreDuration` in `internal/trigger/trigger.go` and `internal/persist/persist.go`
- [x] 1.5 Update test files (`internal/api/auth_test.go`, `internal/api/server_test.go`, `internal/trigger/trigger_test.go`) to use `DefaultDuration`
- [x] 1.6 Update `README.md` and `DESIGN.md` references

## 2. Add TriggerOpts and request body parsing

- [x] 2.1 Define `TriggerOpts` struct in `internal/trigger/trigger.go` with `Duration *time.Duration` and `Since *time.Time` fields
- [x] 2.2 Define `TriggerRequest` struct in `internal/api/server.go` with `Duration string` and `Since string` JSON fields
- [x] 2.3 Update `handleTrigger` to parse optional JSON body into `TriggerRequest`, validate inputs (mutually exclusive, valid duration, since not in future), and return 400 on errors
- [x] 2.4 Change `Dispatcher.Trigger` signature to `Trigger(source string, opts TriggerOpts)` and update all call sites

## 3. Implement custom time window in save logic

- [x] 3.1 Update `Dispatcher.save()` to compute `from` based on `TriggerOpts`: use `opts.Since` if set, else `now - opts.Duration` if set, else `now - cfg.DefaultDuration`
- [x] 3.2 Add `Warning` field to `TriggerRecord` struct (`json:"warning,omitempty"`)
- [x] 3.3 After querying `SegmentsInWindow`, detect if earliest segment starts after `from` and set warning on the record
- [x] 3.4 Pass the effective time range info (requested duration, actual from/to, warning) to `persist.SaveCapture`

## 4. Extend metadata

- [x] 4.1 Add `RequestedDuration`, `ActualFrom`, `ActualTo`, `Warning` fields to `TriggerMeta` in `internal/persist/persist.go`
- [x] 4.2 Rename `PreDuration` to `DefaultDuration` in `TriggerMeta` JSON tags
- [x] 4.3 Update `SaveCapture` signature to accept the new time range information and populate the extended metadata
- [x] 4.4 Include warning in the API response (already in `TriggerRecord` from 3.2)

## 5. Tests

- [x] 5.1 Add API handler tests: trigger with duration, trigger with since, trigger with both (400), trigger with invalid duration (400), trigger with no body (default), trigger with future since (400)
- [x] 5.2 Add dispatcher tests: verify correct `from` calculation for each variant (duration, since, default)
- [x] 5.3 Add test for best-effort persistence: mock ring buffer returning partial data, verify warning is set and data is still saved
- [x] 5.4 Run full test suite and fix any breakage from the rename
