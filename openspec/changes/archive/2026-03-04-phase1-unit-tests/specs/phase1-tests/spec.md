## ADDED Requirements

### Requirement: Config package is unit-tested
The `internal/config` package SHALL have tests that verify default values, validation rules, and `SegmentCount` derivation.

#### Scenario: Defaults are sane
- **WHEN** `config.Defaults()` is called
- **THEN** `BufferSize` is 2 GB, `SegmentSize` is 100 MB, `SegmentCount` is 20, `APIPort` is 9800, and `Promiscuous` is true

#### Scenario: Validate derives SegmentCount
- **WHEN** `Validate()` is called on a config with `BufferSize = 1 GB` and `SegmentSize = 100 MB`
- **THEN** `SegmentCount` is set to 10 and `Validate` returns nil

#### Scenario: Validate rejects missing interface
- **WHEN** `Validate()` is called with an empty `Interface` field
- **THEN** an error is returned containing "interface"

#### Scenario: Validate rejects too-small buffer
- **WHEN** `BufferSize` is less than `SegmentSize`
- **THEN** `Validate()` returns an error

### Requirement: SegmentWriter writes valid pcapng
The `internal/buffer.SegmentWriter` SHALL produce a readable pcapng file and track byte/packet counters accurately.

#### Scenario: WritePacket increments counters
- **WHEN** two packets of 100 bytes each are written to a `SegmentWriter`
- **THEN** `PacketCount()` returns 2 and `BytesWritten()` returns 200

#### Scenario: Close produces readable pcapng
- **WHEN** packets are written and `Close()` is called
- **THEN** the resulting file can be read back by `pcapgo.NgReader` without error

#### Scenario: StartTime is set on creation
- **WHEN** a new `SegmentWriter` is created
- **THEN** `StartTime()` returns a time within the last second

### Requirement: RingManager rotates correctly
The `internal/buffer.RingManager` SHALL advance the active segment index on `Rotate()` and expose segment metadata.

#### Scenario: Rotate advances segment index
- **WHEN** `Rotate()` is called on a ring with 3 segments
- **THEN** the next `CurrentWriter()` is backed by `segment_001.pcapng`

#### Scenario: Rotate wraps around
- **WHEN** `Rotate()` is called N times on a ring of N segments
- **THEN** the active segment index returns to 0

#### Scenario: SegmentsInWindow filters by time
- **WHEN** two segments have non-overlapping time ranges and a window covers only the second
- **THEN** `SegmentsInWindow` returns exactly that second segment

#### Scenario: Insufficient disk space is rejected
- **WHEN** `NewRingManager` is called with a fake `DiskOps` that reports less free space than required
- **THEN** an error containing "insufficient disk space" is returned

### Requirement: Persist.SaveCapture creates correct layout
The `internal/persist.SaveCapture` SHALL create a timestamped directory, copy segments, and write `metadata.json`.

#### Scenario: Saved directory is created
- **WHEN** `SaveCapture` is called with a list of segment paths
- **THEN** a new directory exists under `savedDir` named `<timestamp>_<source>`

#### Scenario: metadata.json is valid JSON
- **WHEN** `SaveCapture` completes successfully
- **THEN** `metadata.json` in the saved directory is valid JSON with the correct `trigger_id`, `source`, and `interface` fields

#### Scenario: Segment files are copied
- **WHEN** `SaveCapture` is called with one segment path
- **THEN** a file with the same basename exists in the saved directory and has identical content

### Requirement: Trigger.Dispatcher records history
The `internal/trigger.Dispatcher` SHALL record trigger events and return them newest-first from `History()`.

#### Scenario: Trigger returns a pending record immediately
- **WHEN** `Trigger("api")` is called
- **THEN** it returns a `TriggerRecord` with `Status == "pending"` and a non-empty `ID`

#### Scenario: History is newest-first
- **WHEN** three triggers are fired in sequence
- **THEN** `History()[0].Timestamp` >= `History()[1].Timestamp` >= `History()[2].Timestamp`

#### Scenario: Concurrent triggers are safe
- **WHEN** 10 goroutines each call `Trigger("api")` simultaneously
- **THEN** `History()` returns exactly 10 records with no data races (verified with `-race`)

### Requirement: API handlers return correct HTTP responses
The `internal/api` HTTP handlers SHALL return correct status codes and JSON bodies for all Phase 1 endpoints.

#### Scenario: GET /api/v1/health returns 200 with status ok
- **WHEN** a GET request is sent to `/api/v1/health`
- **THEN** the response is HTTP 200 with body `{"status":"ok"}`

#### Scenario: GET /api/v1/status returns interface name
- **WHEN** a GET request is sent to `/api/v1/status`
- **THEN** the response body contains the configured interface name

#### Scenario: POST /api/v1/trigger returns 202
- **WHEN** a POST request is sent to `/api/v1/trigger`
- **THEN** the response is HTTP 202 with a JSON body containing a non-empty `id` field

#### Scenario: GET /api/v1/ring returns segment array
- **WHEN** a GET request is sent to `/api/v1/ring`
- **THEN** the response body is a JSON array

### Requirement: Storage disk operations work on Linux
The `internal/storage` package's Linux implementation SHALL correctly report free bytes, preallocate files, and lock/unlock files.

#### Scenario: FreeBytes returns a positive value for a valid path
- **WHEN** `FreeBytes` is called with a path in `os.TempDir()`
- **THEN** it returns a value greater than zero and no error

#### Scenario: Preallocate sets file size
- **WHEN** `Preallocate` is called on a new empty file with size 1 MB
- **THEN** `f.Stat().Size()` returns 1 MB

#### Scenario: LockFile succeeds on an unlocked file
- **WHEN** `LockFile` is called on an open file in a temp directory
- **THEN** it returns nil

#### Scenario: UnlockFile succeeds after LockFile
- **WHEN** `LockFile` is called followed by `UnlockFile` on the same file
- **THEN** both return nil

### Requirement: CLI helper functions are unit-tested
The `parseSize` and `sanitize` helper functions in `cmd/dashcap` SHALL be tested via a `main_test.go` in the same package.

#### Scenario: parseSize parses GB suffix
- **WHEN** `parseSize("2GB", &n)` is called
- **THEN** `n` equals `2 * 1024 * 1024 * 1024`

#### Scenario: parseSize parses MB suffix
- **WHEN** `parseSize("100MB", &n)` is called
- **THEN** `n` equals `100 * 1024 * 1024`

#### Scenario: parseSize rejects garbage input
- **WHEN** `parseSize("notasize", &n)` is called
- **THEN** an error is returned

#### Scenario: sanitize replaces special characters
- **WHEN** `sanitize("Wi-Fi 2.4GHz")` is called
- **THEN** the result contains only alphanumeric characters, hyphens, and underscores
