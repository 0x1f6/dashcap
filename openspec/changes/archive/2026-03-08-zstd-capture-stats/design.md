## Context

`SaveCapture` in `internal/persist/persist.go` currently iterates over ring segments, reads every packet, and writes them into a single uncompressed `capture.pcapng`. The merge loop already touches every packet — this is the natural point to add both compression and statistics without extra I/O passes. The `metadata.json` beside the capture only stores trigger context today.

## Goals / Non-Goals

**Goals:**
- Compress saved captures with zstd streaming compression during the existing merge pass — zero additional I/O.
- Collect lightweight packet-header statistics (protocol, IP, MAC distributions) during the same pass.
- Persist statistics in the existing `metadata.json` alongside trigger context.
- Maintain constant memory usage regardless of capture size.

**Non-Goals:**
- Compressing ring buffer segments (latency-sensitive write path — not touched).
- Deep packet inspection or payload analysis (only L2/L3/L4 headers).
- Decompression tooling within dashcap (Wireshark and `zstd -d` handle this).
- Changing the ring buffer or capture pipeline — only the persist/save path is affected.
- Backward-compatible reading of old uncompressed captures (clean break).

## Decisions

### 1. Compression library: `github.com/klauspost/compress/zstd`

Pure-Go, no CGO dependency, widely adopted (used by CockroachDB, Prometheus, etc.). Alternatives considered:
- **`github.com/valyala/gozstd`**: CGO wrapper — faster but breaks cross-compilation and complicates CI.
- **gzip (`compress/gzip`)**: stdlib, but ~3× worse compression ratio at comparable speed. zstd wins on both ratio and decompression speed.
- **Built-in pcapng compression blocks**: Part of the spec but poorly supported by tools. Wireshark handles `.pcapng.zst` natively since 3.6.

Decision: `klauspost/compress` — best trade-off of performance, portability, and ecosystem support.

### 2. Pipeline architecture: stacked `io.Writer`

```
pcapng.NgWriter
  → zstd.Encoder (streaming, constant memory)
    → *os.File (capture.pcapng.zst)
```

The pcapng writer writes into the zstd encoder which writes compressed blocks to disk. No branching or splitting needed — statistics are collected from the *read* side (packet data + CaptureInfo), not the write side.

Alternatives considered:
- **`io.TeeReader` split-stream**: Unnecessary complexity — stats don't need the encoded pcapng bytes, they need parsed packet headers which are already available from `ngr.ReadPacketData()`.
- **Post-processing pass**: Would double I/O and require the full uncompressed file on disk temporarily.

Decision: Single-pass with stats collected on read, compression on write. Simplest possible architecture.

### 3. Stats collection: header-only parsing via `gopacket.DecodeOptions{Lazy: true, NoCopy: true}`

Use gopacket's lazy decode to extract L2 (Ethernet), L3 (IPv4/IPv6), and L4 (TCP/UDP/ICMP) layer types from each packet. `Lazy` defers full parsing until a layer is accessed, `NoCopy` avoids allocations. We only inspect `LayerType()` and extract addresses — no payload parsing.

Alternatives considered:
- **Manual byte-offset parsing**: Faster but fragile, hard to maintain across encapsulations (VLAN, tunnels).
- **Full gopacket decode**: Unnecessary overhead — we don't need reassembly or application-layer parsing.

Decision: Lazy decode — fast enough for the save path (not real-time capture), handles edge cases correctly.

### 4. Stats data structure

```go
type CaptureStats struct {
    TotalPackets int64
    TotalBytes   int64
    FirstPacket  time.Time
    LastPacket   time.Time
    Protocols    map[string]int64    // e.g. "TCP": 12345
    TopSrcIPs    []AddrCount         // sorted, top N
    TopDstIPs    []AddrCount         // sorted, top N
    TopSrcMACs   []AddrCount         // sorted, top N
    TopDstMACs   []AddrCount         // sorted, top N
    UniqueIPs    int                 // total unique IPs seen
    UniqueMACs   int                 // total unique MACs seen
}
```

During collection, full maps track all addresses. At serialization time, maps are sorted and truncated to top-N (e.g., 20) to keep metadata.json readable. Total unique counts are preserved.

### 5. Zstd encoder settings

- Compression level: `zstd.SpeedDefault` (level 3) — good ratio without significant CPU cost.
- Window size: default (8 MB) — sufficient for streaming, constant memory.
- Encoder is created once per `SaveCapture` call and closed after flush.
- The encoder's `Close()` writes the final zstd frame, so it must be called before the file is closed.

### 6. File naming: `capture.pcapng.zst`

Conventional double extension signals both the format and compression. Wireshark recognizes this natively. `metadata.json` `capture_path` field updates accordingly.

## Risks / Trade-offs

- **[CPU overhead on save]** → zstd at default level adds ~5-10% wall-clock time to the merge. Acceptable since saves are infrequent and not latency-critical. The smaller output also means less disk I/O.
- **[Memory for stats maps]** → In pathological cases (DDoS with millions of unique IPs), the address maps could grow large. → Mitigation: cap map entries at a configurable limit (e.g., 100k). Beyond that, increment a `truncated` counter and stop inserting new keys. This bounds memory at ~10 MB worst case.
- **[Breaking change for tooling]** → Any scripts expecting `capture.pcapng` will break. → Mitigation: document in release notes. The `metadata.json` `capture_path` field is the authoritative reference — tools should use it, not hardcode filenames.
- **[Wireshark compatibility]** → Wireshark < 3.6 cannot open `.pcapng.zst` directly. → Mitigation: users can decompress with `zstd -d`. Document minimum Wireshark version in README.
- **[gopacket decode failures]** → Malformed packets may fail lazy decode. → Mitigation: count packet in totals regardless; skip protocol/address extraction on decode error. Never fail the save due to a stats error.
