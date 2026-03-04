## MODIFIED Requirements

### Requirement: BytesWritten tracks total file bytes, not just payload
`SegmentWriter.BytesWritten()` SHALL return the total number of bytes written to the underlying file, including all pcapng framing (SHB, IDB, EPB headers, padding), not only the raw packet payload.

#### Scenario: BytesWritten exceeds sum of packet payloads
- **GIVEN** a SegmentWriter with two 100-byte packets written
- **WHEN** `BytesWritten()` is called
- **THEN** the returned value is greater than 200 (accounting for pcapng SHB, IDB, and two EPB headers)

#### Scenario: BytesWritten matches actual file size
- **GIVEN** a SegmentWriter with packets written and then flushed
- **WHEN** the file is stat'd on disk
- **THEN** `BytesWritten()` equals the file's size reported by the OS

### Requirement: Segment rotation respects actual file size
The capture loop SHALL rotate segments based on total file bytes written so that segments do not exceed `segmentSize` on disk.

#### Scenario: Segment does not grow beyond segmentSize
- **GIVEN** a segment size of 4096 bytes
- **WHEN** packets are written until rotation triggers
- **THEN** the segment file on disk is at most `segmentSize` plus the size of the last packet written (the packet that caused the threshold to be met)

### Requirement: PacketCount remains unchanged
`SegmentWriter.PacketCount()` SHALL continue to return the number of packets written, unaffected by the byte-counting change.

#### Scenario: PacketCount is still accurate
- **GIVEN** a SegmentWriter with N packets written
- **WHEN** `PacketCount()` is called
- **THEN** it returns N
