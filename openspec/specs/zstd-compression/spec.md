## ADDED Requirements

### Requirement: Saved captures are zstd-compressed
`SaveCapture` SHALL write the merged pcapng output through a streaming zstd encoder, producing a file named `capture.pcapng.zst`. The compression SHALL happen inline during the segment-merge pass without buffering the entire uncompressed output.

#### Scenario: Normal save produces compressed file
- **WHEN** a trigger fires and `SaveCapture` merges segments
- **THEN** the saved directory SHALL contain `capture.pcapng.zst` (not `capture.pcapng`)
- **AND** the file SHALL be a valid zstd-compressed stream that decompresses to valid pcapng

#### Scenario: Compressed output is smaller than uncompressed equivalent
- **WHEN** a trigger saves a capture with typical network traffic
- **THEN** `capture.pcapng.zst` SHALL be smaller than the equivalent uncompressed pcapng

### Requirement: Compression uses constant memory
The zstd encoder SHALL operate in streaming mode with a fixed window size. Memory usage SHALL NOT scale with the size of the capture being saved.

#### Scenario: Large capture does not increase memory
- **WHEN** `SaveCapture` processes a 500 MB capture (across multiple segments)
- **THEN** the zstd encoder's memory allocation SHALL remain constant (bounded by window size, not input size)

### Requirement: Compressed output is a valid zstd frame
The encoder SHALL be properly finalized (frame closed) after all packets are written. The resulting file SHALL be decompressible by standard zstd tools (`zstd -d`, `zstdcat`) and by Wireshark ≥ 3.6.

#### Scenario: Decompression with standard tools
- **WHEN** `capture.pcapng.zst` is decompressed with `zstd -d`
- **THEN** the resulting file SHALL be identical to what an uncompressed pcapng writer would have produced

#### Scenario: Empty capture (no packets)
- **WHEN** `SaveCapture` is called but no packets are written (segments matched but empty)
- **THEN** the system SHALL still produce a valid zstd-compressed pcapng file with SHB and IDB but no packets

### Requirement: Compression errors fail the save
If the zstd encoder encounters an error during writing or finalization, `SaveCapture` SHALL return an error and clean up the partial output file.

#### Scenario: Write error during compression
- **WHEN** the zstd encoder returns an error during `Write` or `Close`
- **THEN** `SaveCapture` SHALL return an error
- **AND** the partial `capture.pcapng.zst` file SHALL be removed
