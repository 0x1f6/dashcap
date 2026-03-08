## ADDED Requirements

### Requirement: Packet statistics are collected during merge
During the segment-merge pass, `SaveCapture` SHALL inspect each packet's headers (L2/L3/L4) and accumulate statistics. Statistics collection SHALL NOT prevent a successful save — decode errors for individual packets SHALL be silently skipped for stats purposes while the packet is still written to the output.

#### Scenario: Stats reflect all merged packets
- **WHEN** a trigger merges 3 segments containing 10,000 total packets
- **THEN** the collected stats SHALL report `total_packets: 10000`
- **AND** `total_bytes` SHALL equal the sum of all packet capture lengths

#### Scenario: Malformed packet does not block stats
- **WHEN** a packet cannot be decoded by gopacket (truncated or unknown encapsulation)
- **THEN** the packet SHALL still be counted in `total_packets` and `total_bytes`
- **AND** the save SHALL complete successfully

### Requirement: Protocol distribution is tracked
The stats SHALL include a map of protocol names to packet counts. Protocols SHALL be identified from decoded layer types (Ethernet, IPv4, IPv6, ARP, TCP, UDP, ICMPv4, ICMPv6, DNS, and any other layer gopacket identifies).

#### Scenario: Mixed traffic protocol counts
- **WHEN** a capture contains 5000 TCP packets, 3000 UDP packets, and 2000 ICMP packets
- **THEN** `protocols` SHALL contain at least `{"TCP": 5000, "UDP": 3000, "ICMPv4": 2000}`
- **AND** lower-layer protocols (e.g. `"Ethernet"`, `"IPv4"`) SHALL also be counted

### Requirement: IP address distribution is tracked
The stats SHALL track unique source and destination IP addresses (both v4 and v6) with per-address packet counts. The metadata SHALL include the top-N addresses sorted by packet count descending, plus a total unique count.

#### Scenario: Top source IPs in metadata
- **WHEN** a capture has traffic from 500 unique source IPs
- **THEN** `top_src_ips` SHALL list the top 20 source IPs by packet count in descending order
- **AND** `unique_ips` SHALL report the total number of distinct IPs seen (src + dst combined)

#### Scenario: IPv6 addresses are included
- **WHEN** a capture contains IPv6 traffic
- **THEN** IPv6 addresses SHALL appear in `top_src_ips` / `top_dst_ips` alongside IPv4 addresses

### Requirement: MAC address distribution is tracked
The stats SHALL track unique source and destination MAC addresses with per-address packet counts. The metadata SHALL include the top-N MACs sorted by packet count descending, plus a total unique count.

#### Scenario: Top MAC addresses in metadata
- **WHEN** a capture has traffic from 50 unique source MACs
- **THEN** `top_src_macs` SHALL list the top 20 source MACs by packet count in descending order
- **AND** `unique_macs` SHALL report the total number of distinct MACs seen (src + dst combined)

### Requirement: Time span is recorded
The stats SHALL record the timestamp of the first and last packet in the merged capture, plus the computed duration.

#### Scenario: Time span reflects actual packet timestamps
- **WHEN** a capture spans from 14:00:00 to 14:05:30
- **THEN** `first_packet` SHALL be `14:00:00`, `last_packet` SHALL be `14:05:30`, and `duration` SHALL be `5m30s`

### Requirement: Stats are persisted in metadata.json
The collected statistics SHALL be written as a `stats` object within the existing `metadata.json` file alongside the trigger metadata.

#### Scenario: metadata.json contains stats object
- **WHEN** a save completes successfully
- **THEN** `metadata.json` SHALL contain a top-level `stats` key
- **AND** the `stats` object SHALL include `total_packets`, `total_bytes`, `protocols`, `first_packet`, `last_packet`, `duration`, `top_src_ips`, `top_dst_ips`, `top_src_macs`, `top_dst_macs`, `unique_ips`, and `unique_macs`

### Requirement: Address map memory is bounded
During collection, if the number of unique addresses (IPs or MACs) exceeds a threshold (100,000), the collector SHALL stop inserting new addresses and increment a `truncated_addresses` counter. Already-tracked addresses SHALL continue to be updated.

#### Scenario: High-cardinality traffic
- **WHEN** a capture contains traffic from 200,000 unique source IPs
- **THEN** the stats SHALL track at most 100,000 unique IPs
- **AND** `truncated_addresses` SHALL be greater than zero
- **AND** `SaveCapture` SHALL complete without excessive memory usage
