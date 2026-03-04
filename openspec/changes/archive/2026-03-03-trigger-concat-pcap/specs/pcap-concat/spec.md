## ADDED Requirements

### Requirement: Trigger produces a single pcapng output file
When a trigger fires, `SaveCapture` SHALL merge all matching ring buffer segments into a single pcapng file named `capture.pcapng` in the destination directory, instead of copying individual segment files.

#### Scenario: Single segment in trigger window
- **WHEN** a trigger fires and only one segment falls within the pre-duration window
- **THEN** the saved directory SHALL contain a single `capture.pcapng` with all packets from that segment

#### Scenario: Multiple segments in trigger window
- **WHEN** a trigger fires and three segments fall within the pre-duration window
- **THEN** the saved directory SHALL contain a single `capture.pcapng` with packets from all three segments in chronological order

#### Scenario: No segments in trigger window
- **WHEN** a trigger fires but no segments overlap with the pre-duration window
- **THEN** `SaveCapture` SHALL return an error indicating no segments were available

### Requirement: Packets are written in chronological order
The merged output file SHALL contain packets ordered by segment StartTime. Segments SHALL be sorted by their StartTime before reading, so packets from earlier segments appear before packets from later segments.

#### Scenario: Ring buffer wraparound preserves order
- **WHEN** the ring buffer has wrapped around (e.g., segments 18, 19, 0, 1 are in the window)
- **THEN** the output file SHALL contain packets starting from segment 18's data through segment 1's data in chronological order

### Requirement: Active segment reads respect valid byte boundary
When merging the currently active segment (which may be pre-allocated beyond written data), the system SHALL read only `SegmentMeta.Bytes` valid bytes from that segment file.

#### Scenario: Active segment with pre-allocated padding
- **WHEN** the active segment has 50 MB of valid pcapng data in a 100 MB pre-allocated file
- **THEN** the merger SHALL read only the first 50 MB and produce valid pcapng output without trailing garbage data

### Requirement: Metadata reflects single output file
The `metadata.json` written alongside the capture SHALL use a `capture_path` field (string) referencing the single output file, replacing the previous `segments` array field.

#### Scenario: Metadata file content
- **WHEN** a trigger completes and metadata.json is written
- **THEN** the JSON SHALL contain a `capture_path` field with value `"capture.pcapng"` and SHALL NOT contain a `segments` array field

### Requirement: Output file is valid pcapng
The merged `capture.pcapng` SHALL be a valid pcapng file with a single Section Header Block and a single Interface Description Block, readable by standard tools (Wireshark, tshark, tcpdump).

#### Scenario: Output readable by pcapng parser
- **WHEN** the merged `capture.pcapng` is opened with a pcapng reader
- **THEN** it SHALL parse without errors and contain the expected total packet count (sum of all source segments)
