## 1. Add platform default path functions

- [x] 1.1 In `internal/storage/disk_unix.go`, add `DefaultDataDir() string` returning `"/var/lib/dashcap"`
- [x] 1.2 In `internal/storage/disk_unix.go`, add `DefaultLockDir() string` returning `"/run/dashcap"`
- [x] 1.3 In `internal/storage/disk_windows.go`, add `DefaultDataDir() string` returning `C:\ProgramData\dashcap`
- [x] 1.4 In `internal/storage/disk_windows.go`, add `DefaultLockDir() string` returning `C:\ProgramData\dashcap\locks`

## 2. Update main.go to use platform functions

- [x] 2.1 In `cmd/dashcap/main.go`, replace `filepath.Join("/var/lib/dashcap", ...)` (line ~95) with `filepath.Join(storage.DefaultDataDir(), ...)`
- [x] 2.2 In `cmd/dashcap/main.go`, replace `filepath.Join("/run/dashcap", ...)` (line ~112) with `filepath.Join(storage.DefaultLockDir(), ...)`
- [x] 2.3 Add `import` for `internal/storage` in `main.go` if not already present

## 3. Verify

- [x] 3.1 Run `go build ./...` to confirm compilation on the host platform
- [x] 3.2 Run `GOOS=windows go vet ./...` to verify Windows build tags resolve correctly
- [x] 3.3 Grep `main.go` for leftover hardcoded `/var/lib/dashcap` or `/run/dashcap` strings — expect zero matches
