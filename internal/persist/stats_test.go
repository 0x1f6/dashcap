package persist_test

import (
	"net"
	"testing"
	"time"

	"dashcap/internal/persist"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// buildEthernetIPv4TCP creates a minimal Ethernet/IPv4/TCP packet.
func buildEthernetIPv4TCP(t *testing.T, srcMAC, dstMAC net.HardwareAddr, srcIP, dstIP net.IP, srcPort, dstPort uint16) []byte {
	t.Helper()
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	ip := &layers.IPv4{SrcIP: srcIP, DstIP: dstIP, Version: 4, IHL: 5, TTL: 64, Protocol: layers.IPProtocolTCP}
	tcp := &layers.TCP{SrcPort: layers.TCPPort(srcPort), DstPort: layers.TCPPort(dstPort)}
	_ = tcp.SetNetworkLayerForChecksum(ip)
	err := gopacket.SerializeLayers(buf, opts,
		&layers.Ethernet{SrcMAC: srcMAC, DstMAC: dstMAC, EthernetType: layers.EthernetTypeIPv4},
		ip, tcp,
		gopacket.Payload([]byte("payload")),
	)
	if err != nil {
		t.Fatalf("serialize TCP packet: %v", err)
	}
	return buf.Bytes()
}

// buildEthernetIPv4UDP creates a minimal Ethernet/IPv4/UDP packet.
func buildEthernetIPv4UDP(t *testing.T, srcMAC, dstMAC net.HardwareAddr, srcIP, dstIP net.IP) []byte {
	t.Helper()
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	ip := &layers.IPv4{SrcIP: srcIP, DstIP: dstIP, Version: 4, IHL: 5, TTL: 64, Protocol: layers.IPProtocolUDP}
	udp := &layers.UDP{SrcPort: 12345, DstPort: 53}
	_ = udp.SetNetworkLayerForChecksum(ip)
	err := gopacket.SerializeLayers(buf, opts,
		&layers.Ethernet{SrcMAC: srcMAC, DstMAC: dstMAC, EthernetType: layers.EthernetTypeIPv4},
		ip, udp,
		gopacket.Payload([]byte("query")),
	)
	if err != nil {
		t.Fatalf("serialize UDP packet: %v", err)
	}
	return buf.Bytes()
}

// buildEthernetIPv6TCP creates a minimal Ethernet/IPv6/TCP packet.
func buildEthernetIPv6TCP(t *testing.T, srcMAC, dstMAC net.HardwareAddr, srcIP, dstIP net.IP) []byte {
	t.Helper()
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	ip6 := &layers.IPv6{SrcIP: srcIP, DstIP: dstIP, Version: 6, NextHeader: layers.IPProtocolTCP, HopLimit: 64}
	tcp := &layers.TCP{SrcPort: 443, DstPort: 8080}
	_ = tcp.SetNetworkLayerForChecksum(ip6)
	err := gopacket.SerializeLayers(buf, opts,
		&layers.Ethernet{SrcMAC: srcMAC, DstMAC: dstMAC, EthernetType: layers.EthernetTypeIPv6},
		ip6, tcp,
		gopacket.Payload([]byte("v6data")),
	)
	if err != nil {
		t.Fatalf("serialize IPv6/TCP packet: %v", err)
	}
	return buf.Bytes()
}

var (
	macA = net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0x01}
	macB = net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0x02}
	ipA  = net.IP{10, 0, 0, 1}
	ipB  = net.IP{10, 0, 0, 2}
	ipC  = net.IP{10, 0, 0, 3}
)

func TestStatsCollectorMixedTraffic(t *testing.T) {
	sc := persist.NewStatsCollector()

	t0 := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	tcpPkt := buildEthernetIPv4TCP(t, macA, macB, ipA, ipB, 1234, 80)
	udpPkt := buildEthernetIPv4UDP(t, macA, macB, ipA, ipC)

	// Add 3 TCP packets and 2 UDP packets
	for i := 0; i < 3; i++ {
		ci := gopacket.CaptureInfo{
			Timestamp:     t0.Add(time.Duration(i) * time.Second),
			CaptureLength: len(tcpPkt),
			Length:        len(tcpPkt),
		}
		sc.Add(tcpPkt, ci)
	}
	for i := 0; i < 2; i++ {
		ci := gopacket.CaptureInfo{
			Timestamp:     t0.Add(time.Duration(3+i) * time.Second),
			CaptureLength: len(udpPkt),
			Length:        len(udpPkt),
		}
		sc.Add(udpPkt, ci)
	}

	stats := sc.Finalize()

	if stats.TotalPackets != 5 {
		t.Errorf("TotalPackets: got %d, want 5", stats.TotalPackets)
	}
	if stats.Protocols["TCP"] != 3 {
		t.Errorf("TCP count: got %d, want 3", stats.Protocols["TCP"])
	}
	if stats.Protocols["UDP"] != 2 {
		t.Errorf("UDP count: got %d, want 2", stats.Protocols["UDP"])
	}
	if stats.Protocols["Ethernet"] != 5 {
		t.Errorf("Ethernet count: got %d, want 5", stats.Protocols["Ethernet"])
	}
	if stats.Protocols["IPv4"] != 5 {
		t.Errorf("IPv4 count: got %d, want 5", stats.Protocols["IPv4"])
	}
}

func TestStatsCollectorIPv6(t *testing.T) {
	sc := persist.NewStatsCollector()

	srcIPv6 := net.ParseIP("2001:db8::1")
	dstIPv6 := net.ParseIP("2001:db8::2")
	pkt := buildEthernetIPv6TCP(t, macA, macB, srcIPv6, dstIPv6)

	t0 := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	ci := gopacket.CaptureInfo{Timestamp: t0, CaptureLength: len(pkt), Length: len(pkt)}
	sc.Add(pkt, ci)

	stats := sc.Finalize()

	if stats.Protocols["IPv6"] != 1 {
		t.Errorf("IPv6 count: got %d, want 1", stats.Protocols["IPv6"])
	}
	if len(stats.TopSrcIPs) == 0 || stats.TopSrcIPs[0].Addr != "2001:db8::1" {
		t.Errorf("expected src IPv6 2001:db8::1, got %v", stats.TopSrcIPs)
	}
	if len(stats.TopDstIPs) == 0 || stats.TopDstIPs[0].Addr != "2001:db8::2" {
		t.Errorf("expected dst IPv6 2001:db8::2, got %v", stats.TopDstIPs)
	}
}

func TestStatsCollectorMACExtraction(t *testing.T) {
	sc := persist.NewStatsCollector()

	pkt := buildEthernetIPv4TCP(t, macA, macB, ipA, ipB, 80, 443)
	t0 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	ci := gopacket.CaptureInfo{Timestamp: t0, CaptureLength: len(pkt), Length: len(pkt)}
	sc.Add(pkt, ci)

	stats := sc.Finalize()

	if stats.UniqueMACs != 2 {
		t.Errorf("UniqueMACs: got %d, want 2", stats.UniqueMACs)
	}
	if len(stats.TopSrcMACs) != 1 || stats.TopSrcMACs[0].Addr != macA.String() {
		t.Errorf("TopSrcMACs: got %v, want %s", stats.TopSrcMACs, macA.String())
	}
	if len(stats.TopDstMACs) != 1 || stats.TopDstMACs[0].Addr != macB.String() {
		t.Errorf("TopDstMACs: got %v, want %s", stats.TopDstMACs, macB.String())
	}
}

func TestStatsCollectorTimeSpan(t *testing.T) {
	sc := persist.NewStatsCollector()

	pkt := buildEthernetIPv4TCP(t, macA, macB, ipA, ipB, 80, 443)

	t0 := time.Date(2025, 1, 1, 14, 0, 0, 0, time.UTC)
	t1 := t0.Add(5*time.Minute + 30*time.Second)

	ci0 := gopacket.CaptureInfo{Timestamp: t0, CaptureLength: len(pkt), Length: len(pkt)}
	ci1 := gopacket.CaptureInfo{Timestamp: t1, CaptureLength: len(pkt), Length: len(pkt)}
	sc.Add(pkt, ci0)
	sc.Add(pkt, ci1)

	stats := sc.Finalize()

	if !stats.FirstPacket.Equal(t0) {
		t.Errorf("FirstPacket: got %v, want %v", stats.FirstPacket, t0)
	}
	if !stats.LastPacket.Equal(t1) {
		t.Errorf("LastPacket: got %v, want %v", stats.LastPacket, t1)
	}
	if stats.Duration != "5m30s" {
		t.Errorf("Duration: got %q, want %q", stats.Duration, "5m30s")
	}
}

func TestStatsCollectorMalformedPacket(t *testing.T) {
	sc := persist.NewStatsCollector()

	garbage := []byte{0xff, 0xfe, 0x00}
	t0 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	ci := gopacket.CaptureInfo{Timestamp: t0, CaptureLength: len(garbage), Length: len(garbage)}
	sc.Add(garbage, ci)

	stats := sc.Finalize()

	if stats.TotalPackets != 1 {
		t.Errorf("TotalPackets: got %d, want 1 (malformed packet should still be counted)", stats.TotalPackets)
	}
	if stats.TotalBytes != int64(len(garbage)) {
		t.Errorf("TotalBytes: got %d, want %d", stats.TotalBytes, len(garbage))
	}
}

func TestStatsCollectorTruncation(t *testing.T) {
	sc := persist.NewStatsCollector()

	t0 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Generate packets with unique IPs beyond the threshold.
	// We use MaxTrackedAddresses/2 + 1 unique src IPs and the same for dst
	// to hit the combined limit.
	limit := persist.MaxTrackedAddresses/2 + 1

	for i := 0; i < limit; i++ {
		srcIP := net.IP{byte(10 + i>>16), byte(i >> 8), byte(i), 1}
		dstIP := net.IP{byte(10 + i>>16), byte(i >> 8), byte(i), 2}
		pkt := buildEthernetIPv4TCP(t, macA, macB, srcIP, dstIP, 80, 443)
		ci := gopacket.CaptureInfo{Timestamp: t0, CaptureLength: len(pkt), Length: len(pkt)}
		sc.Add(pkt, ci)
	}

	stats := sc.Finalize()

	if stats.TruncatedAddresses == 0 {
		t.Error("expected TruncatedAddresses > 0 after exceeding limit")
	}
	if stats.TotalPackets != int64(limit) {
		t.Errorf("TotalPackets: got %d, want %d (all packets should be counted regardless of truncation)", stats.TotalPackets, limit)
	}
	if len(stats.TopSrcIPs) > persist.TopN {
		t.Errorf("TopSrcIPs should be capped at %d, got %d", persist.TopN, len(stats.TopSrcIPs))
	}
}

func TestStatsCollectorTopNSorting(t *testing.T) {
	sc := persist.NewStatsCollector()

	t0 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Send 10 packets from ipA, 5 from ipB, 1 from ipC
	for i := 0; i < 10; i++ {
		pkt := buildEthernetIPv4TCP(t, macA, macB, ipA, ipB, 80, 443)
		ci := gopacket.CaptureInfo{Timestamp: t0, CaptureLength: len(pkt), Length: len(pkt)}
		sc.Add(pkt, ci)
	}
	for i := 0; i < 5; i++ {
		pkt := buildEthernetIPv4TCP(t, macA, macB, ipB, ipA, 80, 443)
		ci := gopacket.CaptureInfo{Timestamp: t0, CaptureLength: len(pkt), Length: len(pkt)}
		sc.Add(pkt, ci)
	}
	pkt := buildEthernetIPv4TCP(t, macA, macB, ipC, ipA, 80, 443)
	ci := gopacket.CaptureInfo{Timestamp: t0, CaptureLength: len(pkt), Length: len(pkt)}
	sc.Add(pkt, ci)

	stats := sc.Finalize()

	if len(stats.TopSrcIPs) != 3 {
		t.Fatalf("TopSrcIPs: got %d entries, want 3", len(stats.TopSrcIPs))
	}
	if stats.TopSrcIPs[0].Addr != ipA.String() || stats.TopSrcIPs[0].Count != 10 {
		t.Errorf("TopSrcIPs[0]: got %v, want %s:10", stats.TopSrcIPs[0], ipA)
	}
	if stats.TopSrcIPs[1].Addr != ipB.String() || stats.TopSrcIPs[1].Count != 5 {
		t.Errorf("TopSrcIPs[1]: got %v, want %s:5", stats.TopSrcIPs[1], ipB)
	}
	if stats.TopSrcIPs[2].Addr != ipC.String() || stats.TopSrcIPs[2].Count != 1 {
		t.Errorf("TopSrcIPs[2]: got %v, want %s:1", stats.TopSrcIPs[2], ipC)
	}
}
