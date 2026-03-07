## Context

dashcap captures all traffic on a network interface into a ring buffer. The `capture.Source` interface already defines `SetBPFFilter(expr string) error`, and the `PcapSource` implementation delegates to `pcap.Handle.SetBPFFilter`. However, nothing in the system calls this method — there is no way to configure or apply exclusion filters.

The DESIGN.md specifies exclusion filters as named BPF expressions in the YAML config (section 7.1). The `fileConfig` YAML struct in `internal/config/load.go` does not yet have an exclusions field, and `config.Config` has no filter-related fields.

BPF filters in libpcap are *inclusion* filters (match = keep). To implement *exclusion* semantics, individual exclusion expressions must be negated and combined with AND.

## Goals / Non-Goals

**Goals:**

- Allow users to exclude traffic from capture via named BPF filter rules in the YAML config
- Provide a `--exclude` CLI flag for simple single-expression exclusions without a config file
- Validate all BPF expressions at startup and fail fast on syntax errors
- Expose the active compiled filter in the `/status` API response
- Match the exclusions config schema from DESIGN.md section 7.1

**Non-Goals:**

- Hot-reload of filters via API (Phase 4 — `PUT /api/v1/filters`)
- A dedicated `/filters` API endpoint
- CLI flags for multiple named exclusions (config file covers that)
- Filter statistics (packets dropped by filter) — libpcap doesn't expose per-filter stats portably

## Decisions

### 1. Filter compilation: combine into a single BPF expression

Individual exclusion rules are combined into one BPF string: `not (<expr1>) and not (<expr2>) and ...`. This single expression is passed to `SetBPFFilter`, which compiles it into kernel-level cBPF.

**Rationale:** libpcap supports exactly one active BPF filter. Combining at the expression level is the standard approach and ensures kernel-level filtering (no userspace overhead). Each sub-expression is parenthesized to avoid operator precedence issues.

**Alternative considered:** Apply multiple pcap filters in sequence — not supported by libpcap; only one filter can be active.

### 2. Config model: named exclusions list

Each exclusion has a `name` (for logging/identification) and a `filter` (BPF expression in tcpdump syntax). The `--exclude` CLI flag provides a single unnamed exclusion for quick use.

```yaml
exclusions:
  - name: backup_traffic
    filter: "host 10.0.0.50 and port 443"
  - name: dns_noise
    filter: "udp port 53 and host 10.0.0.1"
```

CLI exclusions and config file exclusions are merged. The `--exclude` flag adds an entry named `cli` to the list.

### 3. Validation strategy: compile a test filter at startup

BPF syntax is validated by attempting `pcap.CompileBPFFilter` (or opening a temporary handle and calling `SetBPFFilter`) at startup, before entering the capture loop. If the combined expression fails to compile, dashcap exits with a clear error indicating which exclusion is invalid.

**Approach:** Validate each exclusion individually first (to pinpoint which one is invalid), then validate the combined expression. This gives the best error messages.

### 4. Where to apply the filter

The filter is applied in `run()` in `main.go`, immediately after `capture.OpenLive` and before the capture loop starts. This keeps the capture source unaware of exclusion logic.

### 5. Status API: expose filter as a string

The active BPF filter string is added to the `/api/v1/status` response as `"bpf_filter": "<combined expression>"`. Empty string when no filter is active. This is simple and sufficient for operational visibility.

## Risks / Trade-offs

- **Complex combined expressions may hit BPF instruction limits** → libpcap/kernel have a maximum BPF program size (typically 4096 instructions on Linux). Many complex exclusions could exceed this. Mitigation: document the limitation; the error from `SetBPFFilter` is descriptive. This is a known libpcap constraint, not something we can work around.

- **`--exclude` flag only supports one expression** → Users who need multiple exclusions should use the config file. The flag is intentionally simple. Mitigation: users can write compound BPF expressions in the single flag value (e.g., `--exclude "host 10.0.0.1 or port 53"`).

- **No runtime feedback on excluded packet counts** → libpcap provides `pcap.Stats()` which includes `PacketsDropped` (kernel drops), but not filter-excluded counts. This is a libpcap limitation. Mitigation: log the active filter at startup so operators know what's excluded.
