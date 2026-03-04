// Package capture — gopacket/pcap implementation.
package capture

import (
	"fmt"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// PcapSource is a capture.Source backed by gopacket/pcap (libpcap / Npcap).
type PcapSource struct {
	handle *pcap.Handle
}

// OpenLive opens a live capture on iface.
//   - snaplen 0 → full packet (65535 bytes)
//   - promiscuous enables promiscuous mode
func OpenLive(iface string, snaplen int, promiscuous bool) (*PcapSource, error) {
	if snaplen <= 0 {
		snaplen = 65535
	}
	handle, err := pcap.OpenLive(iface, int32(snaplen), promiscuous, pcap.BlockForever)
	if err != nil {
		return nil, fmt.Errorf("pcap open %q: %w", iface, err)
	}
	return &PcapSource{handle: handle}, nil
}

// ReadPacketData implements Source.
func (s *PcapSource) ReadPacketData() ([]byte, gopacket.CaptureInfo, error) {
	return s.handle.ReadPacketData()
}

// SetBPFFilter implements Source.
func (s *PcapSource) SetBPFFilter(expr string) error {
	return s.handle.SetBPFFilter(expr)
}

// LinkType implements Source.
func (s *PcapSource) LinkType() layers.LinkType {
	return s.handle.LinkType()
}

// Close implements Source.
func (s *PcapSource) Close() {
	s.handle.Close()
}
