## Requirements

### Requirement: Dev environment compiles with CGO and libpcap
The devcontainer Dockerfile SHALL install `libpcap-dev` so that `CGO_ENABLED=1 go build ./...` succeeds without manual intervention after container creation.

#### Scenario: Fresh devcontainer build compiles successfully
- **WHEN** a developer rebuilds the devcontainer from the updated Dockerfile
- **THEN** `go build ./...` completes without errors and produces the `bin/dashcap` binary

#### Scenario: Missing libpcap-dev causes clear error
- **WHEN** `libpcap-dev` is not installed and `go build` is run
- **THEN** the build fails with `pcap.h: No such file or directory` (not a silent or misleading error)

### Requirement: Production Dockerfile is absent
The repository SHALL NOT contain a production `Dockerfile` because dashcap is distributed as a native binary, not a container image.

#### Scenario: No Dockerfile at repository root
- **WHEN** the repository root is listed
- **THEN** no `Dockerfile` is present

### Requirement: Makefile uses CGO_ENABLED=1
The `Makefile` `build` target SHALL set `CGO_ENABLED=1` explicitly to prevent CGO-disabled builds on CI environments that default to `CGO_ENABLED=0`.

#### Scenario: Make build produces CGO-enabled binary
- **WHEN** `make build` is executed in the devcontainer
- **THEN** a binary is produced at `bin/dashcap` that is linked against `libpcap.so`

### Requirement: Go module dependencies are declared
The `go.mod` file SHALL declare the following direct dependencies:
- `github.com/google/gopacket` >= v1.1.19 (packet capture and pcapng writing)
- `github.com/spf13/cobra` >= v1.9.0 (CLI framework)
- `golang.org/x/sys` (platform syscalls: flock, fallocate, statfs)

#### Scenario: Dependencies resolve cleanly
- **WHEN** `go mod tidy` is run
- **THEN** `go.mod` and `go.sum` are consistent with no unused or missing dependencies

### Requirement: Internal package directory structure exists
The repository SHALL contain the directory structure defined in DESIGN.md §11:
`cmd/dashcap/`, `internal/config/`, `internal/storage/`, `internal/capture/`, `internal/buffer/`, `internal/persist/`, `internal/trigger/`, `internal/api/`.

#### Scenario: All package directories present
- **WHEN** the repository is checked out
- **THEN** every directory listed in DESIGN.md §11 exists with at least one `.go` file

### Requirement: Package skeletons define exported interfaces
Each internal package SHALL expose its exported interface types and key structs so that cross-package imports compile, even if internal implementations are stubs.

#### Scenario: Module compiles with stub implementations
- **WHEN** `go build ./...` is run
- **THEN** all packages compile without errors, including packages that import other internal packages

#### Scenario: Lint passes on skeleton code
- **WHEN** `golangci-lint run` is executed
- **THEN** zero lint errors are reported on the skeleton code
