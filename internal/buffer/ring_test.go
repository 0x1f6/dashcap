package buffer

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/gopacket/layers"

	"dashcap/internal/config"
)

// fakeDisk is a storage.DiskOps implementation for ring buffer tests.
type fakeDisk struct {
	freeBytes  uint64
	totalBytes uint64
}

func (f *fakeDisk) FreeBytes(_ string) (uint64, error)  { return f.freeBytes, nil }
func (f *fakeDisk) TotalBytes(_ string) (uint64, error) { return f.totalBytes, nil }
func (f *fakeDisk) Preallocate(file *os.File, size int64) error {
	return file.Truncate(size)
}
func (f *fakeDisk) LockFile(_ *os.File) error   { return nil }
func (f *fakeDisk) UnlockFile(_ *os.File) error { return nil }

func newRingTestConfig(t *testing.T) *config.Config {
	t.Helper()
	return &config.Config{
		Interface:         "test0",
		BufferSize:        3 * 1024,
		SegmentSize:       1024,
		SegmentCount:      3,
		DataDir:           t.TempDir(),
		SavedDir:          "saved",
		MinFreeAfterAlloc: 0,
	}
}

func TestNewRingManagerInsufficientSpace(t *testing.T) {
	cfg := newRingTestConfig(t)
	disk := &fakeDisk{freeBytes: 0, totalBytes: 100 << 30}

	_, err := NewRingManager(cfg, disk, layers.LinkTypeEthernet)
	if err == nil {
		t.Fatal("expected error for insufficient disk space, got nil")
	}
	if !strings.Contains(err.Error(), "insufficient disk space") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestNewRingManagerPercentRejectsAllocation(t *testing.T) {
	cfg := newRingTestConfig(t)
	// 100 GB total, 5% = 5 GB minimum. Ring needs 3 KB (3 segments * 1024).
	// With 4 GB free, the absolute threshold (0) passes but the
	// percentage threshold (5 GB) should trigger rejection.
	cfg.MinFreePercent = 5
	cfg.MinFreeAfterAlloc = 0
	disk := &fakeDisk{freeBytes: 4 << 30, totalBytes: 100 << 30}

	_, err := NewRingManager(cfg, disk, layers.LinkTypeEthernet)
	if err == nil {
		t.Fatal("expected error for percentage-based rejection, got nil")
	}
	if !strings.Contains(err.Error(), "insufficient disk space") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestNewRingManagerCreatesSegmentFiles(t *testing.T) {
	cfg := newRingTestConfig(t)
	disk := &fakeDisk{freeBytes: 1 << 30, totalBytes: 100 << 30}

	rm, err := NewRingManager(cfg, disk, layers.LinkTypeEthernet)
	if err != nil {
		t.Fatalf("NewRingManager: %v", err)
	}
	defer func() { _ = rm.Close() }()

	for _, seg := range rm.Segments() {
		if _, err := os.Stat(seg.Path); os.IsNotExist(err) {
			t.Errorf("segment file not found: %s", seg.Path)
		}
	}
}

func TestRotateAdvancesSegment(t *testing.T) {
	cfg := newRingTestConfig(t)
	disk := &fakeDisk{freeBytes: 1 << 30, totalBytes: 100 << 30}

	rm, err := NewRingManager(cfg, disk, layers.LinkTypeEthernet)
	if err != nil {
		t.Fatalf("NewRingManager: %v", err)
	}
	defer func() { _ = rm.Close() }()

	if err := rm.Rotate(); err != nil {
		t.Fatalf("Rotate: %v", err)
	}

	if !strings.HasSuffix(rm.CurrentWriter().Path(), "segment_001.pcapng") {
		t.Errorf("after 1 rotation expected segment_001.pcapng, got: %s", rm.CurrentWriter().Path())
	}
}

func TestRotateWrapsAround(t *testing.T) {
	cfg := newRingTestConfig(t) // 3 segments
	disk := &fakeDisk{freeBytes: 1 << 30, totalBytes: 100 << 30}

	rm, err := NewRingManager(cfg, disk, layers.LinkTypeEthernet)
	if err != nil {
		t.Fatalf("NewRingManager: %v", err)
	}
	defer func() { _ = rm.Close() }()

	// Rotate N times on an N-segment ring → back to segment_000
	for i := 0; i < cfg.SegmentCount; i++ {
		if err := rm.Rotate(); err != nil {
			t.Fatalf("Rotate %d: %v", i, err)
		}
	}

	if !strings.HasSuffix(rm.CurrentWriter().Path(), "segment_000.pcapng") {
		t.Errorf("after full wrap expected segment_000.pcapng, got: %s", rm.CurrentWriter().Path())
	}
}

func TestSegmentsInWindow(t *testing.T) {
	cfg := newRingTestConfig(t)
	disk := &fakeDisk{freeBytes: 1 << 30, totalBytes: 100 << 30}

	rm, err := NewRingManager(cfg, disk, layers.LinkTypeEthernet)
	if err != nil {
		t.Fatalf("NewRingManager: %v", err)
	}
	defer func() { _ = rm.Close() }()

	// Move the current writer to segment 2 so that manual times on
	// segments 0 and 1 are not overwritten by the live writer snapshot.
	rm.current = 2

	// Directly assign time ranges to two segments (white-box, same package)
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	rm.segments[0].StartTime = t0
	rm.segments[0].EndTime = t0.Add(time.Minute)
	rm.segments[1].StartTime = t0.Add(2 * time.Hour)
	rm.segments[1].EndTime = t0.Add(3 * time.Hour)
	// segments[2] is the live writer → EndTime zero → filtered out

	// Window overlapping only segment 0
	got := rm.SegmentsInWindow(t0.Add(-time.Second), t0.Add(30*time.Second))
	if len(got) != 1 {
		t.Fatalf("expected 1 segment in window, got %d", len(got))
	}
	if got[0].Index != 0 {
		t.Errorf("expected index 0, got %d", got[0].Index)
	}

	// Window overlapping only segment 1
	got = rm.SegmentsInWindow(t0.Add(2*time.Hour+30*time.Minute), t0.Add(3*time.Hour+time.Second))
	if len(got) != 1 {
		t.Fatalf("expected 1 segment in window, got %d", len(got))
	}
	if got[0].Index != 1 {
		t.Errorf("expected index 1, got %d", got[0].Index)
	}
}

func TestRingManagerClose(t *testing.T) {
	cfg := newRingTestConfig(t)
	disk := &fakeDisk{freeBytes: 1 << 30, totalBytes: 100 << 30}

	rm, err := NewRingManager(cfg, disk, layers.LinkTypeEthernet)
	if err != nil {
		t.Fatalf("NewRingManager: %v", err)
	}
	if err := rm.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
