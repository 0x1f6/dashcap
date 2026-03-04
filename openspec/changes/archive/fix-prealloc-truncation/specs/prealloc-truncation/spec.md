## MODIFIED Requirements

### Requirement: NewSegmentWriter preserves pre-allocated disk space
`NewSegmentWriter` SHALL open segment files without truncating them, so that disk blocks reserved by `preallocSegment()` are not released on open.

#### Scenario: Pre-allocated file is not truncated on open
- **GIVEN** a segment file pre-allocated to 1 MB via `preallocSegment()`
- **WHEN** `NewSegmentWriter` opens the file
- **THEN** the file size on disk remains 1 MB (not 0)

#### Scenario: Writer starts writing from offset 0
- **GIVEN** a pre-allocated segment file containing old data
- **WHEN** `NewSegmentWriter` opens the file and writes a pcapng SHB
- **THEN** the SHB is written at byte offset 0, overwriting the old content

### Requirement: Close produces a valid pcapng file by truncating to written size
`SegmentWriter.Close()` SHALL truncate the file to `BytesWritten()` bytes after flushing, removing any trailing pre-allocated or stale content beyond the valid pcapng data.

#### Scenario: Closed segment is readable by pcapng readers
- **GIVEN** a SegmentWriter opened on a 1 MB pre-allocated file with 3 packets written (total pcapng size ~500 bytes)
- **WHEN** `Close()` is called and the file is opened with `pcapgo.NewNgReader`
- **THEN** the reader returns exactly 3 packets and reaches EOF without errors

#### Scenario: File size matches BytesWritten after Close
- **GIVEN** a SegmentWriter with packets written and then closed
- **WHEN** the file is stat'd on disk
- **THEN** the file size equals `BytesWritten()`

### Requirement: Rotation preserves pre-allocation on the next segment
`RingManager.Rotate()` SHALL open the next segment without destroying its pre-allocated disk space, and the rotated-out segment SHALL be a valid pcapng file.

#### Scenario: Next segment retains pre-allocated size until writing begins
- **GIVEN** a RingManager with 3 pre-allocated segments of 1 MB each
- **WHEN** `Rotate()` is called
- **THEN** the newly opened segment file has size >= 1 MB on disk (pre-allocation intact) and the closed segment has size equal to its `BytesWritten()`

### Requirement: BytesWritten == file size invariant is preserved
`SegmentWriter.BytesWritten()` SHALL continue to equal the file's stat size after `Close()`.

#### Scenario: Existing BytesWritten cross-check still passes
- **GIVEN** a SegmentWriter with packets written and then closed
- **WHEN** `BytesWritten()` is compared to `os.Stat(path).Size()`
- **THEN** they are equal
