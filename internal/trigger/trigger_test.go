package trigger_test

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/gopacket/layers"

	"dashcap/internal/buffer"
	"dashcap/internal/config"
	"dashcap/internal/trigger"
)

// triggerTestDisk implements storage.DiskOps for trigger tests using temp files.
type triggerTestDisk struct{}

func (triggerTestDisk) FreeBytes(_ string) (uint64, error)       { return 1 << 30, nil }
func (triggerTestDisk) TotalBytes(_ string) (uint64, error)      { return 100 << 30, nil }
func (triggerTestDisk) Preallocate(f *os.File, size int64) error { return f.Truncate(size) }
func (triggerTestDisk) LockFile(_ *os.File) error                { return nil }
func (triggerTestDisk) UnlockFile(_ *os.File) error              { return nil }

func newTriggerTestSetup(t *testing.T) (*config.Config, *buffer.RingManager) {
	t.Helper()
	cfg := &config.Config{
		Interface:         "test0",
		BufferSize:        3 * 1024,
		SegmentSize:       1024,
		SegmentCount:      3,
		DataDir:           t.TempDir(),
		SavedDir:          "saved",
		MinFreeAfterAlloc: 0,
		DefaultDuration:   5 * time.Minute,
	}
	ring, err := buffer.NewRingManager(cfg, triggerTestDisk{}, layers.LinkTypeEthernet)
	if err != nil {
		t.Fatalf("NewRingManager: %v", err)
	}
	t.Cleanup(func() { _ = ring.Close() })
	return cfg, ring
}

// waitForAllComplete polls until all n records in History() are non-pending.
func waitForAllComplete(t *testing.T, d *trigger.Dispatcher, n int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		h := d.History()
		if len(h) >= n {
			allDone := true
			for _, r := range h {
				if r.Status == trigger.StatusPending {
					allDone = false
					break
				}
			}
			if allDone {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Log("timeout waiting for all triggers to complete")
}

func TestTriggerReturnsNonEmptyID(t *testing.T) {
	cfg, ring := newTriggerTestSetup(t)
	d := trigger.NewDispatcher(cfg, ring)

	rec, err := d.Trigger("api", trigger.TriggerOpts{})
	if err != nil {
		t.Fatalf("Trigger: %v", err)
	}
	if rec.ID == "" {
		t.Error("trigger ID should be non-empty")
	}
	if rec.Source != "api" {
		t.Errorf("Source: got %q, want %q", rec.Source, "api")
	}
	// Wait for the save goroutine to finish so temp dir cleanup succeeds.
	waitForAllComplete(t, d, 1)
}

func TestHistoryNewestFirst(t *testing.T) {
	cfg, ring := newTriggerTestSetup(t)
	d := trigger.NewDispatcher(cfg, ring)

	for i := 0; i < 3; i++ {
		if _, err := d.Trigger("test", trigger.TriggerOpts{}); err != nil {
			t.Fatalf("Trigger %d: %v", i, err)
		}
		time.Sleep(time.Millisecond) // ensure distinct timestamps
	}

	waitForAllComplete(t, d, 3)

	history := d.History()
	if len(history) != 3 {
		t.Fatalf("History len: got %d, want 3", len(history))
	}
	for i := 0; i < len(history)-1; i++ {
		if history[i].Timestamp.Before(history[i+1].Timestamp) {
			t.Errorf("history not newest-first: [%d].Timestamp %v < [%d].Timestamp %v",
				i, history[i].Timestamp, i+1, history[i+1].Timestamp)
		}
	}
}

func TestConcurrentTriggersSafe(t *testing.T) {
	cfg, ring := newTriggerTestSetup(t)
	d := trigger.NewDispatcher(cfg, ring)

	const n = 10
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := d.Trigger("test", trigger.TriggerOpts{}); err != nil {
				t.Errorf("Trigger: %v", err)
			}
		}()
	}
	wg.Wait()

	waitForAllComplete(t, d, n)

	if got := len(d.History()); got != n {
		t.Errorf("expected %d trigger records, got %d", n, got)
	}
}
