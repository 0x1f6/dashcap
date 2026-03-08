## Why

The API currently returns trigger records only via `GET /api/v1/triggers` (all records). Users who trigger a capture need a way to poll for a specific trigger's status and retrieve its capture metadata once complete. Additionally, rapid successive triggers can waste resources — debouncing prevents redundant captures within a short window.

## What Changes

- **New endpoint `GET /api/v1/trigger/{id}`**: Returns the trigger record for the given ID. When the capture is completed, the response includes full capture metadata (from `metadata.json`). When still pending, the response signals the client to wait.
- **Trigger debouncing**: The dispatcher rejects new triggers if a trigger was accepted within the last N seconds (hardcoded default: 5s). Returns HTTP 429 with a retry-after hint.

## Capabilities

### New Capabilities
- `trigger-status`: Individual trigger status lookup endpoint with metadata enrichment and pending/completed/failed status handling
- `trigger-debounce`: Rate-limiting of trigger requests via time-based debouncing in the dispatcher

### Modified Capabilities

_(none — existing trigger behavior is unchanged, new functionality is additive)_

## Impact

- **API**: New route `GET /api/v1/trigger/{id}` on the HTTP server; modified `POST /api/v1/trigger` response (429 on debounce)
- **Code**: `internal/api/server.go` (new handler + debounce rejection), `internal/trigger/trigger.go` (debounce logic in Dispatcher), `internal/client/client.go` (new client method)
- **Dependencies**: None — uses existing `TriggerRecord`, `TriggerMeta`, and `metadata.json` infrastructure
