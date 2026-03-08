## MODIFIED Requirements

### Requirement: Trigger produces a single pcapng output file
When a trigger fires, `SaveCapture` SHALL merge all matching ring buffer segments into a single zstd-compressed pcapng file named `capture.pcapng.zst` in the destination directory. The pcapng writer SHALL write through a streaming zstd encoder to the output file.

#### Scenario: Single segment in trigger window
- **WHEN** a trigger fires and only one segment falls within the pre-duration window
- **THEN** the saved directory SHALL contain a single `capture.pcapng.zst` with all packets from that segment

#### Scenario: Multiple segments in trigger window
- **WHEN** a trigger fires and three segments fall within the pre-duration window
- **THEN** the saved directory SHALL contain a single `capture.pcapng.zst` with packets from all three segments in chronological order

#### Scenario: No segments in trigger window
- **WHEN** a trigger fires but no segments overlap with the pre-duration window
- **THEN** `SaveCapture` SHALL return an error indicating no segments were available

### Requirement: Metadata reflects single output file
The `metadata.json` written alongside the capture SHALL use a `capture_path` field (string) referencing the compressed output file.

#### Scenario: Metadata file content
- **WHEN** a trigger completes and metadata.json is written
- **THEN** the JSON SHALL contain a `capture_path` field with value `"capture.pcapng.zst"` and SHALL NOT contain a `segments` array field

### Requirement: Output file is valid pcapng
The merged output SHALL be a valid zstd-compressed pcapng file with a single Section Header Block and a single Interface Description Block. When decompressed, it SHALL be readable by standard tools (Wireshark, tshark, tcpdump). Wireshark ≥ 3.6 SHALL be able to open the `.pcapng.zst` file directly without manual decompression.

#### Scenario: Output readable by pcapng parser
- **WHEN** the merged `capture.pcapng.zst` is decompressed and opened with a pcapng reader
- **THEN** it SHALL parse without errors and contain the expected total packet count (sum of all source segments)
