## Context

The pcapng format supports metadata in the Section Header Block (SHB) via standardized options: `shb_userappl` (application name), `shb_comment` (free-text comment), `shb_hardware`, and `shb_os`. The `pcapgo.NgWriter` already supports these through `NgWriterOptions.SectionInfo` (`NgSectionInfo` struct with `Application`, `Comment`, `Hardware`, `OS` fields). Currently, `NewSegmentWriter` calls `pcapgo.NewNgWriter` which uses `DefaultNgWriterOptions` — setting `Application` to `"gopacket"` and populating `Hardware`/`OS` from `runtime.GOARCH`/`runtime.GOOS`.

The segment writer needs to accept metadata parameters and pass them as `NgWriterOptions` to `NewNgWriterInterface` instead of using the defaults.

## Goals / Non-Goals

**Goals:**
- Every pcapng file produced by dashcap (ring segments and merged saves) carries dashcap version, hostname, and interface name in the SHB
- Metadata is visible in Wireshark via File > Capture File Properties
- Zero additional dependencies — uses existing `pcapgo` API

**Non-Goals:**
- Custom pcapng option codes (dashcap-specific private enterprise numbers)
- Per-packet annotations or comments
- Reading/validating SHB metadata from existing captures

## Decisions

### Use `shb_userappl` for version, `shb_comment` for host/interface context

The pcapng spec defines `shb_userappl` as the producing application identifier. We set it to `dashcap <version>` (e.g. `dashcap v1.2.0`). This replaces the default `"gopacket"` value.

For hostname and interface, `shb_comment` is used with a structured format: `host=<hostname> interface=<iface>`. This is a single-line, grep-friendly format that is also human-readable in Wireshark.

**Alternative considered:** Encoding everything in `shb_comment` as JSON. Rejected because `shb_userappl` exists specifically for the application identifier, and JSON is harder to read at a glance in Wireshark's properties panel.

### Pass metadata via a `SHBInfo` struct to `NewSegmentWriter`

A new `SHBInfo` struct bundles the three values (version, hostname, interface). This keeps the constructor signature clean and extensible. `NewSegmentWriter` maps `SHBInfo` to `pcapgo.NgWriterOptions` and calls `NewNgWriterInterface` instead of `NewNgWriter`.

The `Hardware` and `OS` fields continue to use `runtime.GOARCH` and `runtime.GOOS` from the defaults — no reason to change those.

### Hostname resolved once at startup

`os.Hostname()` is called once during startup in `cmd/dashcap/main.go` and passed down. This avoids repeated syscalls and ensures consistent metadata across all segments in a session.

## Risks / Trade-offs

- **Slightly larger SHB**: Adding `shb_userappl` and `shb_comment` options adds ~50-100 bytes per SHB. Negligible compared to segment sizes (1-100 MB). → No mitigation needed.
- **Comment format not machine-parseable as JSON**: The `key=value` format is less structured than JSON but more readable in Wireshark. → Acceptable trade-off; metadata.json already provides structured data for programmatic use.
