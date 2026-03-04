## 1. Infrastructure Changes

- [x] 1.1 Add `libpcap-dev` RUN layer to `.devcontainer/Dockerfile`
- [x] 1.2 Delete production `Dockerfile` from repository root
- [x] 1.3 Add `CGO_ENABLED=1` to `build` target in `Makefile`
- [x] 1.4 Add `CGO_ENABLED=1` to `cross` / platform-specific targets in `Makefile`
- [x] 1.5 Install `libpcap-dev` in the running container: `sudo apt-get install -y libpcap-dev`

## 2. Go Module Dependencies

- [x] 2.1 Run `go get github.com/google/gopacket@v1.1.19`
- [x] 2.2 Run `go get github.com/spf13/cobra@latest`
- [x] 2.3 Run `go get golang.org/x/sys@latest`
- [x] 2.4 Run `go mod tidy` and verify `go.sum` is consistent

## 3. Directory Structure

- [x] 3.1 Create `cmd/dashcap/` directory
- [x] 3.2 Create `internal/config/` directory
- [x] 3.3 Create `internal/storage/` directory
- [x] 3.4 Create `internal/capture/` directory
- [x] 3.5 Create `internal/buffer/` directory
- [x] 3.6 Create `internal/persist/` directory
- [x] 3.7 Create `internal/trigger/` directory
- [x] 3.8 Create `internal/api/` directory
- [x] 3.9 Create `configs/` directory with `dashcap.example.yaml`

## 4. Package Skeletons

- [x] 4.1 Create `internal/config/config.go` — `Config` struct with all Phase 1 fields, `Defaults()` constructor
- [x] 4.2 Create `internal/storage/storage.go` — `DiskOps` interface definition
- [x] 4.3 Create `internal/storage/disk_unix.go` (`//go:build linux || darwin`) — `flock`, `fallocate`/`statfs` stubs
- [x] 4.4 Create `internal/storage/disk_windows.go` (`//go:build windows`) — Win32 API stubs
- [x] 4.5 Create `internal/capture/capture.go` — `Source` interface definition
- [x] 4.6 Create `internal/capture/pcap.go` — `PcapSource` struct implementing `Source` (stub body)
- [x] 4.7 Create `internal/buffer/writer.go` — `SegmentWriter` struct with `WritePacket` / `Close` stubs
- [x] 4.8 Create `internal/buffer/ring.go` — `RingManager` struct with `SegmentMeta`, `Rotate`, `SegmentsInWindow` stubs
- [x] 4.9 Create `internal/persist/persist.go` — `SaveCapture` function stub + `TriggerMeta` struct
- [x] 4.10 Create `internal/trigger/trigger.go` — `Dispatcher` struct, `TriggerRecord` struct, `Trigger` method stub
- [x] 4.11 Create `internal/api/server.go` — `Server` struct with HTTP handler stubs for all Phase 1 endpoints
- [x] 4.12 Create `cmd/dashcap/main.go` — `version`/`commit`/`buildTime` vars, Cobra root command with all Phase 1 flags

## 5. Verification

- [x] 5.1 Run `go build ./...` — zero errors
- [x] 5.2 Run `make build` — binary produced at `bin/dashcap`
- [x] 5.3 Run `go vet ./...` — zero issues
- [x] 5.4 Run `golangci-lint run` — zero lint errors
- [x] 5.5 Run `make test` — all tests pass (no tests yet, but `go test ./...` must not panic)
