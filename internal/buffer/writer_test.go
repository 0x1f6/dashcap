package buffer

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
)

func TestNewSegmentWriterStartTime(t *testing.T) {
	before := time.Now()
	path := filepath.Join(t.TempDir(), "test.pcapng")

	w, err := NewSegmentWriter(path, layers.LinkTypeEthernet)
	if err != nil {
		t.Fatalf("NewSegmentWriter: %v", err)
	}
	defer func() { _ = w.Close() }()

	if w.StartTime().Before(before) {
		t.Errorf("StartTime %v is before test start %v", w.StartTime(), before)
	}
	if time.Since(w.StartTime()) > time.Second {
		t.Errorf("StartTime is more than 1s in the past: %v", w.StartTime())
	}
}

func TestWritePacketCounters(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.pcapng")

	w, err := NewSegmentWriter(path, layers.LinkTypeEthernet)
	if err != nil {
		t.Fatalf("NewSegmentWriter: %v", err)
	}

	pkt := make([]byte, 100)
	ci := gopacket.CaptureInfo{
		Timestamp:     time.Now(),
		CaptureLength: 100,
		Length:        100,
	}
	for i := 0; i < 2; i++ {
		if err := w.WritePacket(ci, pkt); err != nil {
			t.Fatalf("WritePacket %d: %v", i, err)
		}
	}

	if w.PacketCount() != 2 {
		t.Errorf("PacketCount: got %d, want 2", w.PacketCount())
	}
	if w.BytesWritten() <= 200 {
		t.Errorf("BytesWritten: got %d, want > 200 (payload + pcapng framing)", w.BytesWritten())
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Cross-check: BytesWritten must match actual file size on disk.
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if w.BytesWritten() != fi.Size() {
		t.Errorf("BytesWritten %d != file size %d", w.BytesWritten(), fi.Size())
	}
}

func TestCloseProducesReadablePcapng(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.pcapng")

	w, err := NewSegmentWriter(path, layers.LinkTypeEthernet)
	if err != nil {
		t.Fatalf("NewSegmentWriter: %v", err)
	}

	pkt := make([]byte, 64)
	ci := gopacket.CaptureInfo{
		Timestamp:     time.Now(),
		CaptureLength: 64,
		Length:        64,
	}
	for i := 0; i < 3; i++ {
		if err := w.WritePacket(ci, pkt); err != nil {
			t.Fatalf("WritePacket %d: %v", i, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Read back and verify packet count
	rf, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rf.Close() }()

	r, err := pcapgo.NewNgReader(rf, pcapgo.NgReaderOptions{})
	if err != nil {
		t.Fatalf("NewNgReader: %v", err)
	}

	var count int
	for {
		_, _, err := r.ReadPacketData()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("ReadPacketData: %v", err)
		}
		count++
	}
	if count != 3 {
		t.Errorf("read %d packets, want 3", count)
	}
}

func TestNewSegmentWriterPreservesPrealloc(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prealloc.pcapng")

	// Simulate pre-allocation: create a 1 MB file filled with zeros.
	const preallocSize = 1 << 20
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := f.Truncate(preallocSize); err != nil {
		_ = f.Close()
		t.Fatalf("Truncate to prealloc size: %v", err)
	}
	_ = f.Close()

	// Open with NewSegmentWriter — must NOT truncate the file.
	w, err := NewSegmentWriter(path, layers.LinkTypeEthernet)
	if err != nil {
		t.Fatalf("NewSegmentWriter: %v", err)
	}

	fi, err := os.Stat(path)
	if err != nil {
		_ = w.Close()
		t.Fatalf("Stat: %v", err)
	}
	if fi.Size() < preallocSize {
		_ = w.Close()
		t.Fatalf("file size after open = %d, want >= %d (pre-allocation destroyed)", fi.Size(), preallocSize)
	}
	_ = w.Close()
}

func TestCloseOnPreallocatedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prealloc.pcapng")

	// Simulate pre-allocation: create a 1 MB file.
	const preallocSize = 1 << 20
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := f.Truncate(preallocSize); err != nil {
		_ = f.Close()
		t.Fatalf("Truncate to prealloc size: %v", err)
	}
	_ = f.Close()

	// Open, write packets, close.
	w, err := NewSegmentWriter(path, layers.LinkTypeEthernet)
	if err != nil {
		t.Fatalf("NewSegmentWriter: %v", err)
	}

	pkt := make([]byte, 64)
	ci := gopacket.CaptureInfo{
		Timestamp:     time.Now(),
		CaptureLength: 64,
		Length:        64,
	}
	for i := 0; i < 3; i++ {
		if err := w.WritePacket(ci, pkt); err != nil {
			t.Fatalf("WritePacket %d: %v", i, err)
		}
	}
	bytesWritten := w.BytesWritten()
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// (a) File size must equal BytesWritten (truncated on Close).
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if fi.Size() != bytesWritten {
		t.Errorf("file size %d != BytesWritten %d", fi.Size(), bytesWritten)
	}
	if fi.Size() >= preallocSize {
		t.Errorf("file size %d not truncated below prealloc size %d", fi.Size(), preallocSize)
	}

	// (b) File must be a valid pcapng with exactly 3 packets.
	rf, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rf.Close() }()

	r, err := pcapgo.NewNgReader(rf, pcapgo.NgReaderOptions{})
	if err != nil {
		t.Fatalf("NewNgReader: %v", err)
	}
	var count int
	for {
		_, _, err := r.ReadPacketData()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("ReadPacketData: %v", err)
		}
		count++
	}
	if count != 3 {
		t.Errorf("read %d packets, want 3", count)
	}
}
