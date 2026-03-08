package persist

import (
	"net"
	"sort"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// MaxTrackedAddresses is the maximum number of unique addresses (IPs or MACs)
// tracked before new entries are dropped. Already-tracked addresses continue
// to be updated.
const MaxTrackedAddresses = 100_000

// TopN is the number of top addresses kept in the finalized stats.
const TopN = 20

// AddrCount pairs an address string with its packet count.
type AddrCount struct {
	Addr  string `json:"addr"`
	Count int64  `json:"count"`
}

// CaptureStats holds lightweight packet-header statistics collected during
// the segment-merge pass.
type CaptureStats struct {
	TotalPackets       int64            `json:"total_packets"`
	TotalBytes         int64            `json:"total_bytes"`
	FirstPacket        time.Time        `json:"first_packet"`
	LastPacket         time.Time        `json:"last_packet"`
	Duration           string           `json:"duration"`
	Protocols          map[string]int64 `json:"protocols"`
	TopSrcIPs          []AddrCount      `json:"top_src_ips"`
	TopDstIPs          []AddrCount      `json:"top_dst_ips"`
	TopSrcMACs         []AddrCount      `json:"top_src_macs"`
	TopDstMACs         []AddrCount      `json:"top_dst_macs"`
	UniqueIPs          int              `json:"unique_ips"`
	UniqueMACs         int              `json:"unique_macs"`
	TruncatedAddresses int64            `json:"truncated_addresses,omitempty"`
}

// StatsCollector accumulates packet-header statistics from raw packet data.
// It is not safe for concurrent use.
type StatsCollector struct {
	totalPackets int64
	totalBytes   int64
	firstPacket  time.Time
	lastPacket   time.Time

	protocols map[string]int64
	srcIPs    map[string]int64
	dstIPs    map[string]int64
	srcMACs   map[string]int64
	dstMACs   map[string]int64

	truncated int64

	parser  *gopacket.DecodingLayerParser
	decoded []gopacket.LayerType
	eth     layers.Ethernet
	ip4     layers.IPv4
	ip6     layers.IPv6
	tcp     layers.TCP
	udp     layers.UDP
	icmp4   layers.ICMPv4
	icmp6   layers.ICMPv6
	arp     layers.ARP
	dns     layers.DNS
}

// NewStatsCollector creates a StatsCollector ready for use.
func NewStatsCollector() *StatsCollector {
	sc := &StatsCollector{
		protocols: make(map[string]int64),
		srcIPs:    make(map[string]int64),
		dstIPs:    make(map[string]int64),
		srcMACs:   make(map[string]int64),
		dstMACs:   make(map[string]int64),
	}
	sc.parser = gopacket.NewDecodingLayerParser(
		layers.LayerTypeEthernet,
		&sc.eth, &sc.ip4, &sc.ip6,
		&sc.tcp, &sc.udp,
		&sc.icmp4, &sc.icmp6,
		&sc.arp, &sc.dns,
	)
	sc.parser.IgnoreUnsupported = true
	return sc
}

// Add processes a single packet's raw data and capture info, updating
// internal counters. Decode errors are silently ignored for stats purposes —
// the packet is still counted in totals.
func (sc *StatsCollector) Add(data []byte, ci gopacket.CaptureInfo) {
	sc.totalPackets++
	sc.totalBytes += int64(ci.CaptureLength)

	ts := ci.Timestamp
	if sc.firstPacket.IsZero() || ts.Before(sc.firstPacket) {
		sc.firstPacket = ts
	}
	if ts.After(sc.lastPacket) {
		sc.lastPacket = ts
	}

	sc.decoded = sc.decoded[:0]
	_ = sc.parser.DecodeLayers(data, &sc.decoded)

	for _, lt := range sc.decoded {
		sc.protocols[lt.String()]++

		switch lt {
		case layers.LayerTypeEthernet:
			sc.trackMAC(sc.srcMACs, sc.eth.SrcMAC)
			sc.trackMAC(sc.dstMACs, sc.eth.DstMAC)
		case layers.LayerTypeIPv4:
			sc.trackIP(sc.srcIPs, sc.ip4.SrcIP)
			sc.trackIP(sc.dstIPs, sc.ip4.DstIP)
		case layers.LayerTypeIPv6:
			sc.trackIP(sc.srcIPs, sc.ip6.SrcIP)
			sc.trackIP(sc.dstIPs, sc.ip6.DstIP)
		}
	}
}

func (sc *StatsCollector) trackIP(m map[string]int64, ip net.IP) {
	key := ip.String()
	if _, ok := m[key]; ok {
		m[key]++
		return
	}
	if len(sc.srcIPs)+len(sc.dstIPs) >= MaxTrackedAddresses {
		sc.truncated++
		return
	}
	m[key] = 1
}

func (sc *StatsCollector) trackMAC(m map[string]int64, mac net.HardwareAddr) {
	key := mac.String()
	if _, ok := m[key]; ok {
		m[key]++
		return
	}
	if len(sc.srcMACs)+len(sc.dstMACs) >= MaxTrackedAddresses {
		sc.truncated++
		return
	}
	m[key] = 1
}

// Finalize returns the collected statistics. Address maps are sorted by
// count descending and truncated to TopN entries.
func (sc *StatsCollector) Finalize() CaptureStats {
	uniqueIPs := countUniqueKeys(sc.srcIPs, sc.dstIPs)
	uniqueMACs := countUniqueKeys(sc.srcMACs, sc.dstMACs)

	var dur string
	if !sc.firstPacket.IsZero() && !sc.lastPacket.IsZero() {
		dur = sc.lastPacket.Sub(sc.firstPacket).String()
	}

	return CaptureStats{
		TotalPackets:       sc.totalPackets,
		TotalBytes:         sc.totalBytes,
		FirstPacket:        sc.firstPacket,
		LastPacket:         sc.lastPacket,
		Duration:           dur,
		Protocols:          sc.protocols,
		TopSrcIPs:          topN(sc.srcIPs),
		TopDstIPs:          topN(sc.dstIPs),
		TopSrcMACs:         topN(sc.srcMACs),
		TopDstMACs:         topN(sc.dstMACs),
		UniqueIPs:          uniqueIPs,
		UniqueMACs:         uniqueMACs,
		TruncatedAddresses: sc.truncated,
	}
}

func topN(m map[string]int64) []AddrCount {
	if len(m) == 0 {
		return nil
	}
	list := make([]AddrCount, 0, len(m))
	for addr, count := range m {
		list = append(list, AddrCount{Addr: addr, Count: count})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Count != list[j].Count {
			return list[i].Count > list[j].Count
		}
		return list[i].Addr < list[j].Addr
	})
	if len(list) > TopN {
		list = list[:TopN]
	}
	return list
}

func countUniqueKeys(maps ...map[string]int64) int {
	seen := make(map[string]struct{})
	for _, m := range maps {
		for k := range m {
			seen[k] = struct{}{}
		}
	}
	return len(seen)
}
