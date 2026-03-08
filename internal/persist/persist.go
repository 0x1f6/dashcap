// Package persist handles merging triggered ring segments into permanent storage.
package persist

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"dashcap/internal/buffer"

	"github.com/google/gopacket/pcapgo"
	"github.com/klauspost/compress/zstd"
)

// SaveOpts carries time-range information from the trigger to the persist layer.
type SaveOpts struct {
	DefaultDuration   time.Duration
	RequestedDuration string    // e.g. "30m", "since:2025-…", or "default"
	ActualFrom        time.Time // effective start of the window
	ActualTo          time.Time // effective end of the window (trigger time)
	Warning           string    // set when data is incomplete
}

// TriggerMeta is written as metadata.json alongside saved captures.
type TriggerMeta struct {
	TriggerID         string        `json:"trigger_id"`
	Timestamp         time.Time     `json:"timestamp"`
	Source            string        `json:"source"`
	Interface         string        `json:"interface"`
	DefaultDuration   string        `json:"default_duration"`
	RequestedDuration string        `json:"requested_duration"`
	ActualFrom        time.Time     `json:"actual_from"`
	ActualTo          time.Time     `json:"actual_to"`
	Warning           string        `json:"warning,omitempty"`
	CapturePath       string        `json:"capture_path"`
	Stats             *CaptureStats `json:"stats,omitempty"`
}

// SaveCapture merges the given segments into a single zstd-compressed
// capture.pcapng.zst in a timestamped subdirectory of savedDir, collects
// packet statistics, and writes a metadata.json file.
func SaveCapture(savedDir, triggerID, source, iface string, opts SaveOpts, segments []buffer.SegmentMeta, shb buffer.SHBInfo) (string, error) {
	if len(segments) == 0 {
		return "", fmt.Errorf("persist: no segments to save")
	}

	ts := time.Now().UTC().Format("2006-01-02T15-04-05")
	destDir := filepath.Join(savedDir, fmt.Sprintf("%s_%s", ts, source))
	if err := os.MkdirAll(destDir, 0o750); err != nil {
		return "", fmt.Errorf("persist: mkdir %q: %w", destDir, err)
	}

	capturePath := filepath.Join(destDir, "capture.pcapng.zst")

	// Open output file
	out, err := os.OpenFile(capturePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return "", fmt.Errorf("persist: create output %q: %w", capturePath, err)
	}

	// Stack: pcapng writer → zstd encoder → file
	zw, err := zstd.NewWriter(out)
	if err != nil {
		_ = out.Close()
		return "", fmt.Errorf("persist: create zstd encoder: %w", err)
	}

	stats, err := concatSegments(zw, segments, shb)
	if err != nil {
		_ = zw.Close()
		_ = out.Close()
		_ = os.Remove(capturePath)
		return "", fmt.Errorf("persist: concat segments: %w", err)
	}

	// Close zstd encoder (writes final frame) before closing the file.
	if err := zw.Close(); err != nil {
		_ = out.Close()
		_ = os.Remove(capturePath)
		return "", fmt.Errorf("persist: close zstd encoder: %w", err)
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(capturePath)
		return "", fmt.Errorf("persist: close output file: %w", err)
	}

	meta := TriggerMeta{
		TriggerID:         triggerID,
		Timestamp:         time.Now().UTC(),
		Source:            source,
		Interface:         iface,
		DefaultDuration:   opts.DefaultDuration.String(),
		RequestedDuration: opts.RequestedDuration,
		ActualFrom:        opts.ActualFrom,
		ActualTo:          opts.ActualTo,
		Warning:           opts.Warning,
		CapturePath:       "capture.pcapng.zst",
		Stats:             stats,
	}
	metaPath := filepath.Join(destDir, "metadata.json")
	if err := writeJSON(metaPath, meta); err != nil {
		return "", fmt.Errorf("persist: write metadata: %w", err)
	}

	return destDir, nil
}

// concatSegments reads packets from all source segments (sorted by StartTime),
// writes them as pcapng into w, and collects packet statistics.
func concatSegments(w io.Writer, segments []buffer.SegmentMeta, shb buffer.SHBInfo) (*CaptureStats, error) {
	sort.Slice(segments, func(i, j int) bool {
		return segments[i].StartTime.Before(segments[j].StartTime)
	})

	var ngw *pcapgo.NgWriter
	sc := NewStatsCollector()

	for _, seg := range segments {
		f, err := os.Open(seg.Path)
		if err != nil {
			return nil, fmt.Errorf("open segment %q: %w", seg.Path, err)
		}

		var r io.Reader = f
		if seg.Bytes > 0 {
			r = io.LimitReader(f, seg.Bytes)
		}

		ngr, err := pcapgo.NewNgReader(r, pcapgo.NgReaderOptions{})
		if err != nil {
			_ = f.Close()
			return nil, fmt.Errorf("read segment %q: %w", seg.Path, err)
		}

		if ngw == nil {
			intf := pcapgo.DefaultNgInterface
			intf.LinkType = ngr.LinkType()
			opts := pcapgo.NgWriterOptions{
				SectionInfo: pcapgo.NgSectionInfo{
					Hardware:    runtime.GOARCH,
					OS:          runtime.GOOS,
					Application: "dashcap " + shb.Version,
					Comment:     fmt.Sprintf("host=%s interface=%s", shb.Hostname, shb.Interface),
				},
			}
			ngw, err = pcapgo.NewNgWriterInterface(w, intf, opts)
			if err != nil {
				_ = f.Close()
				return nil, fmt.Errorf("create pcapng writer: %w", err)
			}
		}

		for {
			data, ci, err := ngr.ReadPacketData()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				_ = f.Close()
				return nil, fmt.Errorf("read packet from %q: %w", seg.Path, err)
			}
			sc.Add(data, ci)
			ci.InterfaceIndex = 0
			if err := ngw.WritePacket(ci, data); err != nil {
				_ = f.Close()
				return nil, fmt.Errorf("write packet: %w", err)
			}
		}

		_ = f.Close()
	}

	if ngw != nil {
		if err := ngw.Flush(); err != nil {
			return nil, fmt.Errorf("flush output: %w", err)
		}
	}

	stats := sc.Finalize()
	return &stats, nil
}

func writeJSON(path string, v any) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
