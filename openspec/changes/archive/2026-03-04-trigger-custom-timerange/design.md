## Context

The trigger API (`POST /api/v1/trigger`) currently accepts no request body and always uses the global `PreDuration` config (default 5m) to determine the time window persisted from the ring buffer. The `Dispatcher.Trigger()` method kicks off an async `save()` goroutine that queries `ring.SegmentsInWindow(from, now)` where `from = now - PreDuration`.

The config field is named `PreDuration` / CLI flag `--pre-duration`, which is accurate for the current behavior but will be confusing once per-trigger overrides exist.

## Goals / Non-Goals

**Goals:**
- Allow callers to specify a custom time range per trigger call via `duration` or `since` parameters in the request body.
- Fall back to the configured default duration when no range is specified.
- Rename `PreDuration` → `DefaultDuration` / `--pre-duration` → `--default-duration` for clarity.
- Persist all available data even when the requested range exceeds ring buffer contents (best-effort).
- Include a warning in both the API response and metadata when data is incomplete.

**Non-Goals:**
- Post-trigger capture (Phase 2 feature, out of scope).
- Filtering packets by timestamp within a segment (segments are included/excluded as a whole based on their time range overlap).
- Backward-compatible alias for `--pre-duration` (clean break).

## Decisions

### 1. Request body format

Accept an optional JSON body with two mutually exclusive fields:

```json
{ "duration": "30m" }
```
or
```json
{ "since": "2025-01-01T12:00:00Z" }
```

If both are provided, return `400 Bad Request`. If the body is empty or missing, use `DefaultDuration`.

**Rationale**: `duration` covers the common case ("last X minutes"). `since` enables forensic use cases with a known event timestamp. Keeping them mutually exclusive avoids ambiguity.

**Alternatives considered**: Query parameters instead of JSON body — rejected because POST with body is more consistent and extensible.

### 2. Rename PreDuration → DefaultDuration

Rename everywhere: config struct field, CLI flag, YAML key, metadata JSON key, comments.

- `Config.PreDuration` → `Config.DefaultDuration`
- `--pre-duration` → `--default-duration`
- `pre_duration` (YAML/JSON) → `default_duration`

**Rationale**: With per-trigger overrides, "pre-duration" no longer accurately describes the field's role. "Default duration" is self-documenting.

### 3. Dispatcher.Trigger signature change

Change `Trigger(source string)` to `Trigger(source string, opts TriggerOpts)` where:

```go
type TriggerOpts struct {
    Duration *time.Duration  // override default duration
    Since    *time.Time      // absolute start time
}
```

The `save()` method computes `from` based on: `opts.Since` if set, otherwise `now - opts.Duration` if set, otherwise `now - cfg.DefaultDuration`.

### 4. Best-effort persistence and warning

When `ring.SegmentsInWindow(from, now)` returns segments but the earliest segment's `StartTime` is after `from`, the data is incomplete. In this case:

- Persist what is available (do NOT fail).
- Add a `warning` field to `TriggerRecord` indicating data is shorter than requested.
- Write the actual covered time range into metadata.

When zero segments are returned, keep existing behavior: mark as failed with error "no segments to save".

### 5. Metadata enrichment

Extend `TriggerMeta` with:
- `requested_duration` — what was requested (or "default")
- `actual_from` / `actual_to` — the actual time range of persisted data
- `warning` — set when data is incomplete
- Rename `pre_duration` → `default_duration`

## Risks / Trade-offs

- **BREAKING: CLI flag rename** — Users with scripts using `--pre-duration` must update. Mitigated by clear error message if old flag is used. Low risk given early stage of the project.
- **Segment granularity** — Time range precision is limited to segment boundaries; a "last 1 minute" request may include more data than 1 minute if the segment started earlier. This is acceptable — more data is better than less.
- **`since` with future timestamp** — Validation rejects `since` timestamps in the future (400 error).
- **Large duration requests** — A very large duration simply returns all available ring buffer data with a warning. No performance risk since segment count is bounded.
