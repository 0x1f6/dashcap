// Package trigger dispatches save requests from multiple input sources.
package trigger

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"dashcap/internal/buffer"
	"dashcap/internal/config"
	"dashcap/internal/persist"
)

// DefaultDebounceInterval is the minimum time between accepted triggers.
const DefaultDebounceInterval = 5 * time.Second

// DebounceError is returned when a trigger is rejected due to debouncing.
type DebounceError struct {
	RetryAfter time.Duration
}

func (e *DebounceError) Error() string {
	secs := int((e.RetryAfter + time.Second - 1) / time.Second)
	return fmt.Sprintf("trigger debounced, retry after %d seconds", secs)
}

// IsDebounceError reports whether err is a DebounceError.
func IsDebounceError(err error) (*DebounceError, bool) {
	var de *DebounceError
	if errors.As(err, &de) {
		return de, true
	}
	return nil, false
}

// Status values for TriggerRecord.
const (
	StatusPending   = "pending"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)

// TriggerOpts holds optional per-trigger overrides for the time window.
type TriggerOpts struct {
	Duration *time.Duration // override default duration
	Since    *time.Time     // absolute start time
}

// TriggerRecord records the outcome of a single trigger event.
type TriggerRecord struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"`
	Status    string    `json:"status"`
	SavedPath string    `json:"saved_path,omitempty"`
	Error     string    `json:"error,omitempty"`
	Warning   string    `json:"warning,omitempty"`
}

// Dispatcher multiplexes trigger signals from all input channels.
type Dispatcher struct {
	mu               sync.Mutex
	cfg              *config.Config
	ring             *buffer.RingManager
	shb              buffer.SHBInfo
	history          []*TriggerRecord
	counter          int64
	lastTriggerTime  time.Time
	DebounceInterval time.Duration
}

// NewDispatcher creates a Dispatcher backed by the given ring buffer.
func NewDispatcher(cfg *config.Config, ring *buffer.RingManager, shb buffer.SHBInfo) *Dispatcher {
	return &Dispatcher{cfg: cfg, ring: ring, shb: shb, DebounceInterval: DefaultDebounceInterval}
}

// Trigger initiates a save of the capture window. source identifies the
// caller (e.g. "api"). opts may override the default duration.
// Returns a snapshot copy of the record at the moment of creation (Status = "pending").
func (d *Dispatcher) Trigger(source string, opts TriggerOpts) (*TriggerRecord, error) {
	d.mu.Lock()

	// Debounce check.
	if d.DebounceInterval > 0 && !d.lastTriggerTime.IsZero() {
		elapsed := time.Since(d.lastTriggerTime)
		if elapsed < d.DebounceInterval {
			remaining := d.DebounceInterval - elapsed
			d.mu.Unlock()
			return nil, &DebounceError{RetryAfter: remaining}
		}
	}
	d.lastTriggerTime = time.Now()

	d.counter++
	id := fmt.Sprintf("%d-%d", time.Now().UnixNano(), d.counter)
	rec := &TriggerRecord{
		ID:        id,
		Timestamp: time.Now().UTC(),
		Source:    source,
		Status:    StatusPending,
	}
	d.history = append(d.history, rec)
	// Take a copy under the lock before the save goroutine can mutate rec.Status.
	cp := *rec
	d.mu.Unlock()

	slog.Info("trigger fired", "id", id, "source", source)
	go d.save(rec, opts)
	return &cp, nil
}

// History returns a snapshot of all trigger records (newest first).
func (d *Dispatcher) History() []*TriggerRecord {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]*TriggerRecord, len(d.history))
	for i, r := range d.history {
		cp := *r
		out[len(d.history)-1-i] = &cp
	}
	return out
}

// Get returns a snapshot copy of the TriggerRecord with the given ID,
// or nil if no such record exists.
func (d *Dispatcher) Get(id string) *TriggerRecord {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, r := range d.history {
		if r.ID == id {
			cp := *r
			return &cp
		}
	}
	return nil
}

func (d *Dispatcher) save(rec *TriggerRecord, opts TriggerOpts) {
	now := rec.Timestamp

	// Compute the start of the window based on opts.
	var from time.Time
	var requestedDuration string
	switch {
	case opts.Since != nil:
		from = *opts.Since
		requestedDuration = "since:" + opts.Since.UTC().Format(time.RFC3339)
	case opts.Duration != nil:
		from = now.Add(-*opts.Duration)
		requestedDuration = opts.Duration.String()
	default:
		from = now.Add(-d.cfg.DefaultDuration)
		requestedDuration = "default"
	}

	segments := d.ring.SegmentsInWindow(from, now)
	slog.Debug("trigger window selected", "id", rec.ID, "from", from, "to", now, "segments", len(segments))

	// Best-effort: detect if data is incomplete.
	var warning string
	if len(segments) > 0 {
		sorted := make([]buffer.SegmentMeta, len(segments))
		copy(sorted, segments)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].StartTime.Before(sorted[j].StartTime)
		})
		if sorted[0].StartTime.After(from) {
			warning = fmt.Sprintf("requested data from %s but earliest available data starts at %s",
				from.UTC().Format(time.RFC3339), sorted[0].StartTime.UTC().Format(time.RFC3339))
		}
	}

	savedDir := filepath.Join(d.cfg.DataDir, d.cfg.SavedDir)
	saveOpts := persist.SaveOpts{
		DefaultDuration:   d.cfg.DefaultDuration,
		RequestedDuration: requestedDuration,
		ActualFrom:        from,
		ActualTo:          now,
		Warning:           warning,
	}
	savedPath, err := persist.SaveCapture(savedDir, rec.ID, rec.Source, d.cfg.Interface, saveOpts, segments, d.shb)

	d.mu.Lock()
	defer d.mu.Unlock()
	if err != nil {
		rec.Status = StatusFailed
		rec.Error = err.Error()
		slog.Info("trigger failed", "id", rec.ID, "error", err)
		return
	}
	rec.Status = StatusCompleted
	rec.SavedPath = savedPath
	rec.Warning = warning
	slog.Info("trigger completed", "id", rec.ID, "path", savedPath, "segments", len(segments))
}
