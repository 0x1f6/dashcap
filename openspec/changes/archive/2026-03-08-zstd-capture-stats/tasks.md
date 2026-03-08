## 1. Dependencies & Setup

- [x] 1.1 Add `github.com/klauspost/compress` dependency via `go get`
- [x] 1.2 Verify import compiles: create a throwaway test that opens a `zstd.NewWriter` and closes it

## 2. Stats Collector

- [x] 2.1 Create `internal/persist/stats.go` with `CaptureStats`, `AddrCount` structs and `StatsCollector` type
- [x] 2.2 Implement `StatsCollector.Add(data []byte, ci gopacket.CaptureInfo)` — lazy-decode packet, extract L2/L3/L4 layer types, source/dest IPs and MACs, update internal maps and counters
- [x] 2.3 Implement memory bounding: stop inserting new addresses when map size exceeds 100k, increment `TruncatedAddresses` counter
- [x] 2.4 Implement `StatsCollector.Finalize() CaptureStats` — sort address maps, truncate to top-20, compute duration
- [x] 2.5 Write unit tests for `StatsCollector`: mixed protocol traffic, IPv4/IPv6, MAC extraction, truncation threshold, malformed packet resilience

## 3. Zstd Compression in SaveCapture

- [x] 3.1 Modify `concatSegments` to accept an `io.Writer` instead of creating its own file (decouple file management from merge logic)
- [x] 3.2 In `SaveCapture`, create zstd encoder wrapping the output file, pass encoder to `concatSegments`, close encoder before file
- [x] 3.3 Change output filename from `capture.pcapng` to `capture.pcapng.zst` in `SaveCapture` and `TriggerMeta`
- [x] 3.4 Write integration test: save capture → decompress → verify valid pcapng with expected packet count

## 4. Stats Integration in SaveCapture

- [x] 4.1 Integrate `StatsCollector` into the merge loop in `concatSegments` — call `Add()` for each packet read
- [x] 4.2 Extend `TriggerMeta` with a `Stats *CaptureStats` field (`json:"stats,omitempty"`)
- [x] 4.3 Wire up: `concatSegments` returns `CaptureStats`, `SaveCapture` populates `TriggerMeta.Stats` before writing metadata.json
- [x] 4.4 Write integration test: save capture → read metadata.json → verify stats fields (total_packets, protocols, top IPs/MACs, time span)

## 5. API & Existing Test Updates

- [x] 5.1 Update any API response references from `capture.pcapng` to `capture.pcapng.zst`
- [x] 5.2 Update existing persist/trigger tests to expect `.pcapng.zst` output and extended metadata
- [x] 5.3 Run full test suite, fix any breakage from the filename/metadata changes

## 6. Documentation

- [x] 6.1 Update DESIGN.md open questions section (compression is no longer open)
- [x] 6.2 Add zstd/Wireshark compatibility note to README
