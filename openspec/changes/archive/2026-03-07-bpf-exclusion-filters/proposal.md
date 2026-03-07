## Why

dashcap currently captures all traffic on the configured interface with no way to exclude known-benign, high-volume flows (e.g., backup traffic, DNS noise from a local resolver). This fills the ring buffer with irrelevant packets, reducing the effective retention window for security-relevant traffic. The `capture.Source` interface already defines `SetBPFFilter`, but nothing in the system wires it up — there is no way to configure or apply exclusion filters. This is the next step in the Phase 2 roadmap.

## What Changes

- Add an `exclusions` section to the YAML config file, where each exclusion has a `name` and a BPF `filter` expression (tcpdump syntax)
- Add an `--exclude` CLI flag for passing a single BPF exclusion expression without a config file
- Combine all configured exclusion filters into a single negated BPF expression and apply it at startup via `SetBPFFilter`
- Validate BPF filter syntax at startup; refuse to start if any filter is invalid
- Expose the active BPF filter expression in `GET /api/v1/status` response
- Log each configured exclusion at startup for operational visibility

## Capabilities

### New Capabilities

- `bpf-exclusion-filters`: Configuring, validating, compiling, and applying BPF exclusion filters from config file and CLI flags. Includes exposing the active filter in the status API.

### Modified Capabilities

_(none — no existing spec-level requirements change)_

## Impact

- **Config**: `config.Config` gains an `Exclusions` field; `fileConfig` gains an `exclusions` YAML section; `Validate()` gains BPF syntax checking
- **CLI**: New `--exclude` flag on the root command
- **Startup**: `run()` in `main.go` calls `SetBPFFilter` after opening the capture source
- **API**: `/api/v1/status` response includes a new `bpf_filter` field (empty string if no filter active)
- **Example config**: `configs/dashcap.example.yaml` updated with an `exclusions` section
- **No breaking changes**: Existing configs and CLI invocations continue to work unchanged
