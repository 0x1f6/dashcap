## Why

The dashcap repository has build infrastructure (Makefile, devcontainer, CI) but zero Go source code and no working binary. The project-setup change bootstraps the Go module with the correct dependencies and creates the initial project structure so that active feature development can begin on the Phase 1 MVP.

## What Changes

- Add `libpcap-dev` to the devcontainer Dockerfile so that `gopacket/pcap` (CGO) compiles successfully in the dev environment
- Remove the production `Dockerfile` — dashcap is a single native binary, not a container image (was a template artefact)
- Update the `Makefile` to set `CGO_ENABLED=1` on all build targets
- Add Go module dependencies: `github.com/google/gopacket`, `github.com/spf13/cobra`, `golang.org/x/sys`
- Create the full internal package directory structure as specified in DESIGN.md §11
- Add stub/skeleton Go files for every package so the module compiles end-to-end (`go build ./...` passes)

## Capabilities

### New Capabilities

- `project-scaffold`: Repository structure, devcontainer build environment, and Go module dependency baseline that enables Phase 1 feature implementation

### Modified Capabilities

<!-- none -->

## Impact

- `.devcontainer/Dockerfile`: adds one `RUN apt-get install libpcap-dev` layer
- `Dockerfile`: deleted
- `Makefile`: `CGO_ENABLED=1` added to `build` and `cross` targets
- `go.mod` / `go.sum`: three new direct dependencies
- New directories: `cmd/dashcap/`, `internal/{config,storage,capture,buffer,persist,trigger,api}/`
- New Go stub files in each package (interfaces only, no business logic)
