## 1. Dispatcher: Debounce Logic

- [x] 1.1 Add `DefaultDebounceInterval` constant (5s) and `ErrDebounced` sentinel error to `internal/trigger/trigger.go`
- [x] 1.2 Add `lastTriggerTime time.Time` field to `Dispatcher` struct
- [x] 1.3 Implement debounce check in `Dispatcher.Trigger()`: reject with `ErrDebounced` if within debounce window, update `lastTriggerTime` on accept
- [x] 1.4 Add unit tests for debounce: first trigger accepted, trigger within window rejected, trigger after window accepted

## 2. Dispatcher: Get by ID

- [x] 2.1 Add `Dispatcher.Get(id string) *TriggerRecord` method that searches history and returns a snapshot copy or nil
- [x] 2.2 Add unit tests for Get: existing ID returns record, non-existent ID returns nil

## 3. API: Trigger Status Endpoint

- [x] 3.1 Add `GET /api/v1/trigger/{id}` route registration in `server.go`
- [x] 3.2 Implement `handleTriggerStatus` handler: extract ID from path, call `Dispatcher.Get`, return 404 if nil
- [x] 3.3 Handle pending status: return 202 with trigger record and `retry_after` field
- [x] 3.4 Handle completed status: read `metadata.json` from `saved_path`, return 200 with record + `metadata` field
- [x] 3.5 Handle failed status: return 200 with record including `error` field
- [x] 3.6 Add unit tests for all status endpoint scenarios (completed, pending, failed, not found, metadata unreadable)

## 4. API: Debounce Rejection

- [x] 4.1 Update `handleTrigger` to check for `ErrDebounced` from dispatcher and return 429 with `Retry-After` header
- [x] 4.2 Add unit test for 429 debounce response

## 5. Client Library

- [x] 5.1 Add `TriggerStatus(id string)` method to the REST client in `internal/client/client.go`
- [x] 5.2 Add client test for trigger status method
