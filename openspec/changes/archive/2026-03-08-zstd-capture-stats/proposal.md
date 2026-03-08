## Why

Saved captures are stored as uncompressed pcapng, wasting disk space on data that is rarely re-read. Additionally, the current `metadata.json` only contains trigger context — a user must open the capture in Wireshark to judge whether it's worth analysing. Lightweight triage statistics (protocol distribution, top talkers, unique MACs) at save time would let users decide quickly without loading the full capture.

Both goals can be achieved in a single pass through the packet stream during the existing segment-merge step, keeping memory constant and adding negligible latency.

## What Changes

- **Streaming zstd compression** of the merged pcapng output during `SaveCapture`. The saved file becomes `capture.pcapng.zst` instead of `capture.pcapng`. Compression happens via an `io.Writer` pipeline — no temp files, constant memory.
- **Capture statistics** collected while packets are read for merging. Each packet header is inspected (L2/L3/L4 only — no payload parsing) to accumulate:
  - Total packet count & byte count
  - Per-protocol packet counts (e.g. TCP, UDP, ICMP, ARP, DNS, …)
  - Unique source/destination IP addresses (v4 & v6) with packet counts
  - Unique MAC addresses with packet counts
  - Time span (first/last packet timestamp, duration)
- **Extended `metadata.json`** with a `stats` object containing the above.
- **`capture.pcapng` → `capture.pcapng.zst`** filename change in metadata and API responses. **BREAKING** for any tooling that expects uncompressed `capture.pcapng` at the saved path.

## Capabilities

### New Capabilities
- `zstd-compression`: Streaming zstd compression of saved pcapng captures during the segment-merge pass.
- `capture-stats`: Lightweight packet-header statistics (protocol, IP, MAC distributions) collected during merge and persisted in metadata.

### Modified Capabilities
- `pcap-concat`: The merge pipeline changes from a plain pcapng writer to a split-stream architecture (stats branch + compression branch). Output filename changes from `capture.pcapng` to `capture.pcapng.zst`.

## Impact

- **Code**: `internal/persist/persist.go` (merge pipeline, metadata struct), new stats collector, new zstd writer integration.
- **Dependencies**: New Go dependency `github.com/klauspost/compress/zstd` (pure-Go, no CGO, widely used).
- **API**: `/trigger` response and `/triggers` list will reflect the new filename and extended metadata.
- **Storage**: Saved captures shrink significantly (pcapng compresses well — typically 3-5× with zstd).
- **Compatibility**: Wireshark ≥ 3.6 can open `.pcapng.zst` directly. For older versions, users decompress with `zstd -d`.
