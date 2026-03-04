// Package capture abstracts platform-specific packet capture backends.
package capture

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// Source is the interface that wraps a packet capture backend.
// Implementations must be safe to call from a single goroutine.
type Source interface {
	// ReadPacketData returns the next captured packet.
	// Returns io.EOF when the source is exhausted (live captures block until
	// the source is closed).
	ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error)

	// SetBPFFilter compiles and applies a BPF filter expression.
	// Pass an empty string to remove any active filter.
	SetBPFFilter(expr string) error

	// LinkType returns the data-link-layer type of captured packets.
	LinkType() layers.LinkType

	// Close shuts down the capture source and releases resources.
	Close()
}
