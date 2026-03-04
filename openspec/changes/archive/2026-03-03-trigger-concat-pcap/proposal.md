## Why

When a trigger fires, the persistence layer copies each ring buffer segment as a separate `.pcapng` file into the saved directory. This forces users to manually merge files before analysis in tools like Wireshark or tshark. A single concatenated pcap output per trigger makes saved captures immediately usable.

## What Changes

- Replace individual segment copying in `persist.SaveCapture()` with pcapng concatenation that merges all matching segments into one output file
- The saved directory will contain a single `capture.pcapng` instead of multiple `segment_NNN.pcapng` files
- Update `metadata.json` to reference the single output file instead of a list of segment paths
- Ensure packets are written in chronological order across segments

## Capabilities

### New Capabilities
- `pcap-concat`: Merge multiple pcapng ring buffer segments into a single pcapng output file during trigger persistence

### Modified Capabilities

## Impact

- `internal/persist/persist.go` — rewrite `SaveCapture()` to concatenate instead of copy
- `internal/persist/persist_test.go` — update tests for single-file output
- `internal/trigger/trigger.go` — no changes expected (passes segments to persist as before)
- `metadata.json` format changes — downstream consumers expecting `segmentPaths` array need to handle single file
- Dependency: `gopacket/pcapgo` already available for pcapng read/write
