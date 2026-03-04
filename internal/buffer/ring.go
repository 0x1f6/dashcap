package buffer

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"dashcap/internal/config"
	"dashcap/internal/storage"

	"github.com/google/gopacket/layers"
)

// SegmentMeta holds metadata about a single ring segment.
type SegmentMeta struct {
	Index     int
	Path      string
	StartTime time.Time
	EndTime   time.Time
	Packets   int64
	Bytes     int64
}

// RingManager pre-allocates and manages a fixed-size ring of pcapng segment files.
type RingManager struct {
	mu       sync.RWMutex
	cfg      *config.Config
	disk     storage.DiskOps
	linkType layers.LinkType
	segments []SegmentMeta
	current  int // index of the active segment
	writer   *SegmentWriter
}

// NewRingManager performs the disk safety check, pre-allocates all segment
// files, and returns a ready-to-use RingManager.
func NewRingManager(cfg *config.Config, disk storage.DiskOps, linkType layers.LinkType) (*RingManager, error) {
	ringDir := filepath.Join(cfg.DataDir, "ring")
	if err := os.MkdirAll(ringDir, 0o750); err != nil {
		return nil, fmt.Errorf("ring manager: create ring dir: %w", err)
	}

	// Disk safety check
	free, err := disk.FreeBytes(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("ring manager: free space check: %w", err)
	}
	totalDisk, err := disk.TotalBytes(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("ring manager: total space check: %w", err)
	}
	ringTotal := uint64(cfg.SegmentCount) * uint64(cfg.SegmentSize)
	minFreeAbs := uint64(cfg.MinFreeAfterAlloc)
	minFreePct := uint64(float64(totalDisk) * cfg.MinFreePercent / 100)
	effectiveMin := max(minFreeAbs, minFreePct)
	if free < ringTotal+effectiveMin {
		return nil, fmt.Errorf(
			"ring manager: insufficient disk space: %d bytes free, need %d (ring) + %d (safety margin, %.0f%% of %d)",
			free, ringTotal, effectiveMin, cfg.MinFreePercent, totalDisk,
		)
	}

	rm := &RingManager{
		cfg:      cfg,
		disk:     disk,
		linkType: linkType,
		segments: make([]SegmentMeta, cfg.SegmentCount),
	}

	// Pre-allocate segment files
	for i := range rm.segments {
		path := filepath.Join(ringDir, fmt.Sprintf("segment_%03d.pcapng", i))
		rm.segments[i] = SegmentMeta{Index: i, Path: path}
		if err := preallocSegment(path, cfg.SegmentSize, disk); err != nil {
			return nil, fmt.Errorf("ring manager: prealloc segment %d: %w", i, err)
		}
	}

	// Open the first segment for writing
	w, err := NewSegmentWriter(rm.segments[0].Path, linkType)
	if err != nil {
		return nil, fmt.Errorf("ring manager: open initial segment: %w", err)
	}
	rm.writer = w
	return rm, nil
}

// CurrentWriter returns the active SegmentWriter.
func (rm *RingManager) CurrentWriter() *SegmentWriter {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.writer
}

// snapshotSegments returns a copy of all segment metadata, including live
// stats from the currently active writer. Caller must hold rm.mu.
func (rm *RingManager) snapshotSegments() []SegmentMeta {
	out := make([]SegmentMeta, len(rm.segments))
	copy(out, rm.segments)
	if rm.writer != nil {
		cur := &out[rm.current]
		cur.StartTime = rm.writer.StartTime()
		cur.EndTime = rm.writer.EndTime()
		cur.Packets = rm.writer.PacketCount()
		cur.Bytes = rm.writer.BytesWritten()
	}
	return out
}

// Rotate closes the current segment, advances the ring index, and opens the
// next segment for writing (overwriting the oldest data).
func (rm *RingManager) Rotate() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Capture stats before closing
	meta := &rm.segments[rm.current]
	meta.StartTime = rm.writer.StartTime()
	meta.EndTime = rm.writer.EndTime()
	meta.Packets = rm.writer.PacketCount()
	meta.Bytes = rm.writer.BytesWritten()

	if err := rm.writer.Close(); err != nil {
		return fmt.Errorf("ring rotate: close segment %d: %w", rm.current, err)
	}

	rm.current = (rm.current + 1) % rm.cfg.SegmentCount
	next := &rm.segments[rm.current]

	w, err := NewSegmentWriter(next.Path, rm.linkType)
	if err != nil {
		return fmt.Errorf("ring rotate: open segment %d: %w", rm.current, err)
	}
	// Reset metadata for the overwritten slot
	next.StartTime = time.Time{}
	next.EndTime = time.Time{}
	next.Packets = 0
	next.Bytes = 0

	rm.writer = w
	return nil
}

// SegmentsInWindow returns all segments whose time range overlaps [from, to].
func (rm *RingManager) SegmentsInWindow(from, to time.Time) []SegmentMeta {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	var out []SegmentMeta
	for _, s := range rm.snapshotSegments() {
		if s.StartTime.IsZero() {
			continue // not yet written
		}
		if s.EndTime.Before(from) || s.StartTime.After(to) {
			continue
		}
		out = append(out, s)
	}
	return out
}

// Segments returns a snapshot of all segment metadata.
func (rm *RingManager) Segments() []SegmentMeta {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.snapshotSegments()
}

// Close flushes and closes the active segment.
func (rm *RingManager) Close() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	if rm.writer == nil {
		return nil
	}
	return rm.writer.Close()
}

// preallocSegment creates (or truncates) path and pre-allocates size bytes.
func preallocSegment(path string, size int64, disk storage.DiskOps) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return fmt.Errorf("open segment %q: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return disk.Preallocate(f, size)
}
