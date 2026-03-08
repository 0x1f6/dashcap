## Context

dashcap exposes a REST API for triggering network captures. `POST /api/v1/trigger` starts an async save and returns a `TriggerRecord` with `status: "pending"`. The only way to check the result is `GET /api/v1/triggers`, which returns all records. There is no way to poll a specific trigger or retrieve capture metadata (stats, time window, etc.) via the API. Additionally, nothing prevents rapid-fire triggers from creating redundant captures.

Key existing structures:
- `TriggerRecord` (in-memory): `id`, `timestamp`, `source`, `status`, `saved_path`, `error`, `warning`
- `TriggerMeta` (on-disk `metadata.json`): full capture metadata including stats, time window, interface
- `Dispatcher` manages records in `history []* TriggerRecord` with a mutex

## Goals / Non-Goals

**Goals:**
- Allow polling a single trigger by ID, returning its record and — when completed — the capture metadata from `metadata.json`
- Return a clear "still processing" response for pending triggers so callers know to retry
- Prevent rapid-fire triggers by debouncing at the Dispatcher level (hardcoded 5s cooldown)

**Non-Goals:**
- Webhooks or push notifications for trigger completion
- Configurable debounce duration (hardcoded for now, can be made configurable later)
- Streaming/downloading the capture file via the API
- Pagination or filtering on the existing `/triggers` list endpoint

## Decisions

### 1. Lookup by trigger ID in Dispatcher

Add a `Dispatcher.Get(id string) *TriggerRecord` method that searches the history slice. Linear scan is fine — the history list is small (bounded by session lifetime, typically <100 entries).

**Alternative**: Map-based lookup — rejected because it adds a second data structure to keep in sync and the list is too small to matter.

### 2. Metadata enrichment from disk

When the trigger is completed and `saved_path` is set, the handler reads `metadata.json` from `saved_path` and returns an enriched response combining the record and metadata. This avoids duplicating metadata in memory.

**Response structure**: A `TriggerStatusResponse` wrapping the `TriggerRecord` fields plus an optional `metadata` field containing `TriggerMeta`. This keeps the base record shape familiar while adding detail.

**Alternative**: Return raw `metadata.json` content — rejected because it wouldn't include the trigger record fields (`status`, `error`).

### 3. HTTP status codes for trigger status

- `200 OK` with full record + metadata when `status == "completed"`
- `200 OK` with record when `status == "failed"` (error field explains failure)
- `202 Accepted` with record + `retry_after` hint when `status == "pending"` — signals the caller to poll again
- `404 Not Found` when the trigger ID doesn't exist

### 4. Debounce in Dispatcher.Trigger()

Track `lastTriggerTime time.Time` in the Dispatcher. On each `Trigger()` call, check if `time.Since(lastTriggerTime) < debounceInterval`. If too soon, return a sentinel error `ErrDebounced`. The API handler maps this to `429 Too Many Requests` with `Retry-After` header.

**Debounce interval**: 5 seconds, hardcoded as a package-level constant `DefaultDebounceInterval`.

**Alternative**: Middleware-level rate limiting — rejected because debouncing is domain logic (one capture at a time), not generic rate limiting.

## Risks / Trade-offs

- **Disk read on every status poll**: Reading `metadata.json` on each `GET /trigger/{id}` when completed. Acceptable because the file is small (<10KB) and polling frequency is low. Could cache later if needed.
- **Debounce is per-process**: No cross-instance coordination. Fine for dashcap's single-instance deployment model.
- **In-memory history loss on restart**: Trigger records are already ephemeral (lost on restart). This is existing behavior, not a new risk.
