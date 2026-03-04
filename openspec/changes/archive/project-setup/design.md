## Context

dashcap has a complete design document (DESIGN.md) and surrounding infrastructure (Makefile, devcontainer, golangci.yml, Forgejo CI) but zero Go source code. The `go.mod` declares only the module name and Go version. The project is ready to receive its first code, but several infrastructure gaps block compilation:

1. `libpcap-dev` is absent from the devcontainer image — `gopacket/pcap` uses CGO and requires `pcap.h` at build time.
2. The production `Dockerfile` targets `distroless/static` and `CGO_ENABLED=0`, which is incompatible with `gopacket/pcap`. The Dockerfile is also unnecessary — dashcap ships as a native binary, not a container image.
3. The `Makefile` build targets do not set `CGO_ENABLED=1`, which will cause silent CGO-disabled builds on systems where CGO defaults to off.
4. No Go packages exist yet — the module cannot be compiled or linted.

## Goals / Non-Goals

**Goals:**
- Dev environment compiles `go build ./...` without errors after this change
- All direct dependencies pinned in `go.mod` / `go.sum`
- Package skeletons define correct interfaces and exported types for each internal package
- `make build` produces a working binary
- `make lint` passes golangci-lint on the stub code

**Non-Goals:**
- Business logic implementation (capture, ring buffer, API, persistence) — that is Phase 1 feature work
- YAML configuration support (Phase 2 per DESIGN.md)
- Windows cross-compilation from Linux (requires MinGW + Npcap SDK; deferred to a dedicated CI runner)
- macOS support (Phase 5)

## Decisions

### CGO strategy: dynamic link against system libpcap

**Decision:** Build with `CGO_ENABLED=1`, link dynamically against the system's `libpcap.so`. The binary requires `libpcap0.8` on the target host.

**Rationale:** DESIGN.md §3.1 explicitly lists "No runtime dependencies beyond Npcap on Windows" — implying libpcap on Linux is an accepted system dependency (as with Wireshark and similar tools). Dynamic linking keeps the build simple; static linking against glibc is brittle and not necessary for the target use case (managed Linux hosts).

**Alternative considered:** Fully static binary (`-linkmode external -extldflags -static`). Rejected because it requires a musl-based toolchain or all-static glibc, significantly complicating the build environment.

### Production Dockerfile: delete it

**Decision:** Remove `Dockerfile` from the repository.

**Rationale:** dashcap is a daemon binary deployed via package manager, systemd, or direct binary copy — not a container image. The existing Dockerfile was generated from a generic Go devcontainer template and is a misleading artefact.

### Makefile: explicit CGO_ENABLED=1

**Decision:** Set `CGO_ENABLED=1` explicitly on `build`, `build-linux`, and `cross` Makefile targets.

**Rationale:** Some CI environments (e.g., GitHub Actions hosted runners) default `CGO_ENABLED=0`. Explicit setting prevents silent fallback to a CGO-disabled build that would fail at link time when pcap symbols are unresolved.

### Package skeleton approach: interfaces + unexported stubs

**Decision:** Each package exposes its exported interface/types but internal functions return `nil` / `errors.New("not implemented")` stubs.

**Rationale:** Allows `go build ./...` and `golangci-lint` to run and validate the package graph, while making clear which symbols are public API vs. implementation detail. Tests can be written against interfaces before implementation.

### No uuid dependency for trigger IDs

**Decision:** Use `fmt.Sprintf("%d", time.Now().UnixNano())` for trigger IDs in the skeleton.

**Rationale:** Avoids adding a dependency solely for the scaffold. A proper UUID library (`github.com/google/uuid`) can be added when `persist.go` is implemented.

## Risks / Trade-offs

- **libpcap-dev absent in existing container**: The running devcontainer does not have `libpcap-dev` installed yet (the Dockerfile change only applies on rebuild). → Mitigation: Run `sudo apt-get install -y libpcap-dev` once manually in the current container session before building.
- **gopacket v1.1.19 is old (2021)**: No newer release exists; it is effectively unmaintained but stable. → Mitigation: No action needed for Phase 1. If a critical bug is found, fork or switch to an alternative (e.g., `github.com/packetcap/go-pcap`) in a later phase.
- **Windows build not verified**: `disk_windows.go` uses Win32 APIs that cannot be tested from Linux. → Mitigation: Stub with `//go:build windows` build tag and `// TODO` comments; verify on a Windows CI runner in Phase 1 feature work.
- **golangci-lint on stubs**: Some linters (e.g., `unparam`, `unused`) may flag stub functions. → Mitigation: Use `//nolint` directives sparingly on known stub functions, or disable specific linters for the scaffold phase.

## Migration Plan

1. Install `libpcap-dev` in the current running container: `sudo apt-get install -y libpcap-dev`
2. Apply infrastructure changes (devcontainer Dockerfile, Makefile, delete production Dockerfile)
3. Create directory structure and stub files
4. Run `go get` to add dependencies, then `go mod tidy`
5. Verify: `make build` → binary in `bin/dashcap`; `make lint` → zero errors; `go vet ./...` → clean

No rollback complexity — this is additive scaffolding with no existing code to break.

## Open Questions

- Should `internal/trigger/signal_unix.go` (SIGUSR1) be included as a stub now (Phase 4 feature) or deferred? → Defer; keep trigger package minimal for Phase 1.
- Should `configs/dashcap.example.yaml` be created now? → Yes, as a static file with no Go code dependency — useful for documentation even before Phase 2 YAML parsing.
