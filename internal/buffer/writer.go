// Package buffer implements the pcapng segment writer and ring manager.
package buffer

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
)

// SHBInfo holds metadata embedded in the pcapng Section Header Block.
type SHBInfo struct {
	Version   string // e.g. "v1.2.0"
	Hostname  string // OS hostname
	Interface string // capture interface name
}

// countingWriter wraps an io.Writer and counts every byte passing through.
type countingWriter struct {
	w io.Writer
	n int64
}

func (cw *countingWriter) Write(p []byte) (int, error) {
	n, err := cw.w.Write(p)
	cw.n += int64(n)
	return n, err
}

// SegmentWriter writes packets into a single pcapng segment file.
type SegmentWriter struct {
	path    string
	f       *os.File
	cw      *countingWriter
	ng      *pcapgo.NgWriter
	start   time.Time
	end     time.Time
	packets int64
}

// NewSegmentWriter opens (or creates) the file at path and initialises a
// pcapng NgWriter with the supplied link-layer type. Pre-allocated files are
// not truncated on open; writing starts from offset 0.
// The SHBInfo metadata is embedded in the pcapng Section Header Block.
func NewSegmentWriter(path string, linkType layers.LinkType, shb SHBInfo) (*SegmentWriter, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, fmt.Errorf("segment writer open %q: %w", path, err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("segment writer seek %q: %w", path, err)
	}
	cw := &countingWriter{w: f}

	intf := pcapgo.DefaultNgInterface
	intf.LinkType = linkType

	opts := pcapgo.NgWriterOptions{
		SectionInfo: pcapgo.NgSectionInfo{
			Hardware:    runtime.GOARCH,
			OS:          runtime.GOOS,
			Application: "dashcap " + shb.Version,
			Comment:     fmt.Sprintf("host=%s interface=%s", shb.Hostname, shb.Interface),
		},
	}

	ng, err := pcapgo.NewNgWriterInterface(cw, intf, opts)
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("segment writer ng init %q: %w", path, err)
	}
	return &SegmentWriter{
		path:  path,
		f:     f,
		cw:    cw,
		ng:    ng,
		start: time.Now(),
	}, nil
}

// WritePacket appends a packet to the segment.
func (w *SegmentWriter) WritePacket(ci gopacket.CaptureInfo, data []byte) error {
	// The NgWriter registers a single interface at index 0. Packets captured
	// via libpcap may carry the OS interface index, so pin it to 0.
	ci.InterfaceIndex = 0
	if err := w.ng.WritePacket(ci, data); err != nil {
		return fmt.Errorf("segment write %q: %w", w.path, err)
	}
	// Flush so the countingWriter sees all bytes (pcapgo buffers internally).
	if err := w.ng.Flush(); err != nil {
		return fmt.Errorf("segment flush %q: %w", w.path, err)
	}
	w.end = ci.Timestamp
	w.packets++
	return nil
}

// Flush flushes buffered data to the underlying file.
func (w *SegmentWriter) Flush() error {
	return w.ng.Flush()
}

// Close flushes the pcapng writer, truncates the file to the exact number of
// bytes written (removing any trailing pre-allocated or stale content), and
// closes the file.
func (w *SegmentWriter) Close() error {
	if err := w.ng.Flush(); err != nil {
		_ = w.f.Close()
		return fmt.Errorf("segment flush %q: %w", w.path, err)
	}
	if err := w.f.Truncate(w.cw.n); err != nil {
		_ = w.f.Close()
		return fmt.Errorf("segment truncate %q: %w", w.path, err)
	}
	return w.f.Close()
}

// StartTime returns when the first packet in this segment was captured.
func (w *SegmentWriter) StartTime() time.Time { return w.start }

// EndTime returns the timestamp of the most recently written packet.
func (w *SegmentWriter) EndTime() time.Time { return w.end }

// BytesWritten returns the total bytes written to the file, including pcapng framing.
func (w *SegmentWriter) BytesWritten() int64 { return w.cw.n }

// PacketCount returns the number of packets written to this segment.
func (w *SegmentWriter) PacketCount() int64 { return w.packets }

// Path returns the filesystem path of the segment file.
func (w *SegmentWriter) Path() string { return w.path }
