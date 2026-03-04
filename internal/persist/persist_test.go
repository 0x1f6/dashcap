package persist_test

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"dashcap/internal/buffer"
	"dashcap/internal/persist"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
)

func defaultSaveOpts() persist.SaveOpts {
	return persist.SaveOpts{
		DefaultDuration:   5 * time.Minute,
		RequestedDuration: "default",
		ActualFrom:        time.Now().Add(-5 * time.Minute),
		ActualTo:          time.Now(),
	}
}

// writePcapngSegment creates a valid pcapng file with the given packets and
// returns its path. Each packet is a raw byte payload with the given timestamp.
func writePcapngSegment(t *testing.T, dir, name string, packets []pcapPacket) string {
	t.Helper()
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	w, err := pcapgo.NewNgWriter(f, layers.LinkTypeEthernet)
	if err != nil {
		t.Fatal(err)
	}

	for _, pkt := range packets {
		ci := gopacket.CaptureInfo{
			Timestamp:      pkt.ts,
			CaptureLength:  len(pkt.data),
			Length:         len(pkt.data),
			InterfaceIndex: 0,
		}
		if err := w.WritePacket(ci, pkt.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	return path
}

type pcapPacket struct {
	ts   time.Time
	data []byte
}

// readAllPackets reads all packets from a pcapng file and returns them.
func readAllPackets(t *testing.T, path string) []pcapPacket {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	r, err := pcapgo.NewNgReader(f, pcapgo.NgReaderOptions{})
	if err != nil {
		t.Fatal(err)
	}

	var out []pcapPacket
	for {
		data, ci, err := r.ReadPacketData()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		cp := make([]byte, len(data))
		copy(cp, data)
		out = append(out, pcapPacket{ts: ci.Timestamp, data: cp})
	}
	return out
}

func TestSaveCaptureCreatesDirectory(t *testing.T) {
	savedDir := filepath.Join(t.TempDir(), "saved")
	segDir := t.TempDir()

	t0 := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	segPath := writePcapngSegment(t, segDir, "segment_000.pcapng", []pcapPacket{
		{ts: t0, data: []byte("payload")},
	})

	segs := []buffer.SegmentMeta{{Index: 0, Path: segPath, StartTime: t0}}
	gotPath, err := persist.SaveCapture(savedDir, "tid1", "api", "eth0", defaultSaveOpts(), segs)
	if err != nil {
		t.Fatalf("SaveCapture: %v", err)
	}

	info, err := os.Stat(gotPath)
	if err != nil {
		t.Fatalf("saved directory not found: %v", err)
	}
	if !info.IsDir() {
		t.Error("savedPath should be a directory")
	}
	if !strings.Contains(filepath.Base(gotPath), "api") {
		t.Errorf("directory name should contain source 'api', got: %s", filepath.Base(gotPath))
	}
}

func TestSaveCaptureProducesSingleFile(t *testing.T) {
	savedDir := filepath.Join(t.TempDir(), "saved")
	segDir := t.TempDir()

	t0 := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	segPath := writePcapngSegment(t, segDir, "segment_000.pcapng", []pcapPacket{
		{ts: t0, data: []byte("hello")},
	})

	segs := []buffer.SegmentMeta{{Index: 0, Path: segPath, StartTime: t0}}
	gotPath, err := persist.SaveCapture(savedDir, "tid2", "api", "eth0", defaultSaveOpts(), segs)
	if err != nil {
		t.Fatalf("SaveCapture: %v", err)
	}

	capturePath := filepath.Join(gotPath, "capture.pcapng")
	if _, err := os.Stat(capturePath); err != nil {
		t.Fatalf("capture.pcapng not found: %v", err)
	}

	// Old individual segment files should NOT exist
	if _, err := os.Stat(filepath.Join(gotPath, "segment_000.pcapng")); err == nil {
		t.Error("individual segment file should not exist, expected single capture.pcapng")
	}
}

func TestSaveCaptureMultipleSegmentsMerged(t *testing.T) {
	savedDir := filepath.Join(t.TempDir(), "saved")
	segDir := t.TempDir()

	t0 := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	t1 := t0.Add(1 * time.Second)
	t2 := t0.Add(2 * time.Second)

	seg0 := writePcapngSegment(t, segDir, "segment_000.pcapng", []pcapPacket{
		{ts: t0, data: []byte("pkt-0")},
	})
	seg1 := writePcapngSegment(t, segDir, "segment_001.pcapng", []pcapPacket{
		{ts: t1, data: []byte("pkt-1")},
	})
	seg2 := writePcapngSegment(t, segDir, "segment_002.pcapng", []pcapPacket{
		{ts: t2, data: []byte("pkt-2")},
	})

	segs := []buffer.SegmentMeta{
		{Index: 0, Path: seg0, StartTime: t0},
		{Index: 1, Path: seg1, StartTime: t1},
		{Index: 2, Path: seg2, StartTime: t2},
	}
	gotPath, err := persist.SaveCapture(savedDir, "tid3", "api", "eth0", defaultSaveOpts(), segs)
	if err != nil {
		t.Fatalf("SaveCapture: %v", err)
	}

	pkts := readAllPackets(t, filepath.Join(gotPath, "capture.pcapng"))
	if len(pkts) != 3 {
		t.Fatalf("expected 3 packets, got %d", len(pkts))
	}
	for i, want := range []string{"pkt-0", "pkt-1", "pkt-2"} {
		if string(pkts[i].data) != want {
			t.Errorf("packet %d: got %q, want %q", i, string(pkts[i].data), want)
		}
	}
}

func TestSaveCaptureChronologicalOrderWraparound(t *testing.T) {
	savedDir := filepath.Join(t.TempDir(), "saved")
	segDir := t.TempDir()

	// Simulate ring wraparound: segment 2 is oldest, then 0, then 1
	tEarly := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	tMid := tEarly.Add(10 * time.Second)
	tLate := tEarly.Add(20 * time.Second)

	seg0 := writePcapngSegment(t, segDir, "segment_000.pcapng", []pcapPacket{
		{ts: tMid, data: []byte("mid")},
	})
	seg1 := writePcapngSegment(t, segDir, "segment_001.pcapng", []pcapPacket{
		{ts: tLate, data: []byte("late")},
	})
	seg2 := writePcapngSegment(t, segDir, "segment_002.pcapng", []pcapPacket{
		{ts: tEarly, data: []byte("early")},
	})

	// Pass in index order (0, 1, 2) — but chronological is (2, 0, 1)
	segs := []buffer.SegmentMeta{
		{Index: 0, Path: seg0, StartTime: tMid},
		{Index: 1, Path: seg1, StartTime: tLate},
		{Index: 2, Path: seg2, StartTime: tEarly},
	}
	gotPath, err := persist.SaveCapture(savedDir, "tid4", "api", "eth0", defaultSaveOpts(), segs)
	if err != nil {
		t.Fatalf("SaveCapture: %v", err)
	}

	pkts := readAllPackets(t, filepath.Join(gotPath, "capture.pcapng"))
	if len(pkts) != 3 {
		t.Fatalf("expected 3 packets, got %d", len(pkts))
	}
	expected := []string{"early", "mid", "late"}
	for i, want := range expected {
		if string(pkts[i].data) != want {
			t.Errorf("packet %d: got %q, want %q", i, string(pkts[i].data), want)
		}
	}
}

func TestSaveCaptureEmptySegmentsError(t *testing.T) {
	savedDir := filepath.Join(t.TempDir(), "saved")

	_, err := persist.SaveCapture(savedDir, "tid5", "api", "eth0", defaultSaveOpts(), nil)
	if err == nil {
		t.Fatal("expected error for empty segments, got nil")
	}
	if !strings.Contains(err.Error(), "no segments") {
		t.Errorf("error should mention 'no segments', got: %v", err)
	}
}

func TestSaveCaptureMergedOutputReadable(t *testing.T) {
	savedDir := filepath.Join(t.TempDir(), "saved")
	segDir := t.TempDir()

	t0 := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	t1 := t0.Add(1 * time.Second)

	seg0 := writePcapngSegment(t, segDir, "segment_000.pcapng", []pcapPacket{
		{ts: t0, data: []byte("alpha")},
		{ts: t0.Add(100 * time.Millisecond), data: []byte("beta")},
	})
	seg1 := writePcapngSegment(t, segDir, "segment_001.pcapng", []pcapPacket{
		{ts: t1, data: []byte("gamma")},
	})

	segs := []buffer.SegmentMeta{
		{Index: 0, Path: seg0, StartTime: t0},
		{Index: 1, Path: seg1, StartTime: t1},
	}
	gotPath, err := persist.SaveCapture(savedDir, "tid6", "api", "eth0", defaultSaveOpts(), segs)
	if err != nil {
		t.Fatalf("SaveCapture: %v", err)
	}

	pkts := readAllPackets(t, filepath.Join(gotPath, "capture.pcapng"))
	if len(pkts) != 3 {
		t.Fatalf("expected 3 packets, got %d", len(pkts))
	}
	if string(pkts[0].data) != "alpha" {
		t.Errorf("first packet: got %q, want %q", string(pkts[0].data), "alpha")
	}
	if string(pkts[2].data) != "gamma" {
		t.Errorf("last packet: got %q, want %q", string(pkts[2].data), "gamma")
	}
}

func TestSaveCaptureMetadataHasCapturePath(t *testing.T) {
	savedDir := filepath.Join(t.TempDir(), "saved")
	segDir := t.TempDir()

	t0 := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	segPath := writePcapngSegment(t, segDir, "segment_000.pcapng", []pcapPacket{
		{ts: t0, data: []byte("data")},
	})

	segs := []buffer.SegmentMeta{{Index: 0, Path: segPath, StartTime: t0}}
	gotPath, err := persist.SaveCapture(savedDir, "my-trigger-id", "signal", "eth1", defaultSaveOpts(), segs)
	if err != nil {
		t.Fatalf("SaveCapture: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(gotPath, "metadata.json"))
	if err != nil {
		t.Fatalf("read metadata.json: %v", err)
	}

	var meta map[string]any
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("invalid JSON in metadata.json: %v", err)
	}

	if meta["capture_path"] != "capture.pcapng" {
		t.Errorf("capture_path: got %v, want capture.pcapng", meta["capture_path"])
	}
	if _, ok := meta["segments"]; ok {
		t.Error("metadata should not contain 'segments' field")
	}
	if meta["trigger_id"] != "my-trigger-id" {
		t.Errorf("trigger_id: got %v, want my-trigger-id", meta["trigger_id"])
	}
	if meta["source"] != "signal" {
		t.Errorf("source: got %v, want signal", meta["source"])
	}
}

func TestSaveCaptureReturnsPathUnderSavedDir(t *testing.T) {
	savedDir := filepath.Join(t.TempDir(), "saved")
	segDir := t.TempDir()

	t0 := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	segPath := writePcapngSegment(t, segDir, "segment_000.pcapng", []pcapPacket{
		{ts: t0, data: []byte("x")},
	})

	segs := []buffer.SegmentMeta{{Index: 0, Path: segPath, StartTime: t0}}
	gotPath, err := persist.SaveCapture(savedDir, "tid7", "api", "eth0", defaultSaveOpts(), segs)
	if err != nil {
		t.Fatalf("SaveCapture: %v", err)
	}
	if !strings.HasPrefix(gotPath, savedDir) {
		t.Errorf("savedPath %q should be under %q", gotPath, savedDir)
	}
}
