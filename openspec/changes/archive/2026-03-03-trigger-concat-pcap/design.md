## Context

When a trigger fires, `persist.SaveCapture()` copies each matching ring buffer segment as a separate `.pcapng` file into a timestamped directory under `saved/`. The result is multiple files (e.g., `segment_000.pcapng`, `segment_001.pcapng`) that users must manually merge before opening in Wireshark or feeding into analysis pipelines.

The ring buffer segments are standalone pcapng files, each with their own Section Header Block (SHB) and Interface Description Block (IDB), written via `gopacket/pcapgo.NgWriter`. The active segment may be pre-allocated beyond valid data, tracked by `SegmentMeta.Bytes`.

## Goals / Non-Goals

**Goals:**
- Produce a single `capture.pcapng` file per trigger instead of multiple segment files
- Preserve packet chronological order across segments
- Keep the change contained to the persistence layer — trigger and ring buffer remain unchanged
- Update `metadata.json` to reflect the single-file output

**Non-Goals:**
- Changing the ring buffer segment format or rotation logic
- Supporting alternative output formats (pcap legacy, CSV, etc.)
- Post-trigger capture (Phase 2 feature, separate concern)
- Packet filtering or deduplication during merge

## Decisions

### 1. Read packets from source segments, write to a single new pcapng file

**Choice**: Open each source segment with `pcapgo.NgReader`, read packets sequentially, and write them into a single output file via `pcapgo.NgWriter`.

**Alternatives considered**:
- **Binary concatenation of pcapng sections**: pcapng supports multiple sections in one file, so raw byte concatenation would technically work. However, this produces files with multiple SHBs and IDBs which some tools handle inconsistently. Reading and re-writing packets is cleaner and produces a single-section file.
- **mergecap/editcap subprocess**: External tool dependency, not available in all environments (especially embedded/container targets for dashcap).

**Rationale**: Using the already-imported `gopacket/pcapgo` library keeps dependencies unchanged. Reading packets individually ensures proper pcapng framing and a single SHB/IDB in the output. The overhead is acceptable since trigger persistence is not latency-critical.

### 2. Sort segments by StartTime before merging

**Choice**: Sort the `[]SegmentMeta` slice by `StartTime` before iterating. Within each segment, packets are already in order.

**Rationale**: Ring buffer segments may not be in chronological index order (due to wrap-around). Sorting by `StartTime` ensures the output file has packets in chronological order without needing a full sort of individual packets.

### 3. Use `capture.pcapng` as the single output filename

**Choice**: The merged output is always named `capture.pcapng` within the timestamped directory.

**Rationale**: Simple, predictable name. No need for a naming scheme since there's only one file.

### 4. Update TriggerMeta to use a single CapturePath field

**Choice**: Replace `SegmentPaths []string` with `CapturePath string` in the metadata JSON.

**Rationale**: The field semantics change fundamentally — it's one file now, not many. A clean rename avoids confusion. This is a **breaking change** to the metadata format but the format is not yet part of any public API contract.

### 5. Handle the active segment's pre-allocated padding

**Choice**: When reading from the active segment, limit reads to `SegmentMeta.Bytes` valid bytes (same approach as the existing `copyFile` logic).

**Rationale**: The active segment file may contain pre-allocated zeros beyond valid pcapng data. Reading beyond valid bytes would produce parse errors. Truncating or wrapping in an `io.LimitReader` before passing to `NgReader` handles this cleanly.

## Risks / Trade-offs

- **Memory usage**: Reading and writing packets one at a time keeps memory usage constant regardless of total capture size. No risk of loading entire segments into memory.
- **Temporary disk usage**: During concatenation, both source segments (ring) and destination file exist simultaneously. This is the same as the current copy approach — no regression.
- **Slightly more I/O**: Re-encoding packets through pcapgo adds minor CPU/IO overhead vs. raw byte copy. Acceptable since trigger saves are infrequent and not latency-sensitive.
- **pcapng feature loss**: If segments contain pcapng-specific metadata (comments, custom blocks), re-writing via NgWriter may not preserve them. Current segments are simple (SHB + IDB + packets only), so this is not a concern today.
