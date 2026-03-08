# dashcap — A Network Packet Dashcam

> Full-Packet Capture with Ring Buffer and On-Demand Persistence

**Version:** 1.0 · **Date:** March 2026 · **Status:** Phase 1 & 2 complete

---

## 1. Executive Summary

dashcap is a lightweight, cross-platform tool that performs continuous full-packet capture on a network interface, storing captured data in a ring buffer of fixed-size pcapng segments. When triggered by an external signal, dashcap preserves the relevant capture window for later analysis. The concept is analogous to a dashcam for network traffic: always recording, but only saving footage when an incident occurs.

The tool bridges the gap between full packet capture (expensive, storage-intensive) and NetFlow/metadata-only approaches (cheap but lacking detail) by providing on-demand access to full packet data around security-relevant events, without the cost of storing everything permanently.

---

## 2. Problem Statement & Motivation

### 2.1 The Visibility Gap

Security teams face a fundamental tension in network monitoring. Full packet capture provides complete visibility but requires massive storage and generates data that is rarely needed. NetFlow and metadata approaches are storage-efficient but lack the detail needed for incident response and forensic analysis. When a security incident occurs, analysts often find themselves wishing they had full packet data from the time of the incident — data that was never captured or was already rotated out.

### 2.2 The Dashcam Analogy

A dashcam solves an analogous problem for drivers: it records continuously but only saves footage when triggered by an event (collision, hard braking, manual button press). Most footage is silently overwritten. dashcap applies this same principle to network packets: continuously capture all traffic into a ring buffer, and when a security event triggers a save, preserve the relevant time window — including traffic from *before* the trigger occurred (pre-trigger window).

### 2.3 Target Use Cases

- **SIEM/SOAR Integration:** An alert in Splunk or Elastic triggers dashcap to save the packet data around the time of the alert, providing analysts with full packet context for investigation.
- **Incident Response:** During active IR, an analyst can trigger a save to capture the current network state and recent history on any endpoint.
- **Threat Hunting:** Periodic or anomaly-driven triggers preserve packet windows for offline analysis with Wireshark or similar tools.
- **Endpoint Forensics:** Running on endpoints (workstations, servers), dashcap provides a forensic artifact that traditional network taps cannot — traffic from the endpoint's own perspective, including localhost and internal communications.

---

## 3. Goals & Non-Goals

### 3.1 Goals

- **Cross-Platform:** Run on Linux, Windows, and macOS. Windows support is the highest priority alongside Linux.
- **Minimal Footprint:** Predictable, pre-allocated disk usage. Must never compromise system stability.
- **Wireshark Compatibility:** Output in pcapng format, directly openable in Wireshark and compatible with standard packet analysis tools.
- **External Trigger Integration:** REST API for SIEM/SOAR integration and local trigger mechanisms.
- **Multi-Instance:** Multiple independent instances for different interfaces on the same host, with proper isolation and locking.
- **Configurable Exclusions:** BPF-based traffic filters to exclude known-benign high-volume traffic from capture, preserving ring buffer space for relevant packets.
- **Flexible Buffer Limits:** Buffer size configurable by total size, number of segments, or time-based retention duration.
- **Single Binary Deployment:** Single binary with only libpcap (Linux/macOS) or Npcap (Windows) as runtime dependency. Easy to deploy across a fleet.

### 3.2 Non-Goals

- **Real-Time Analysis:** dashcap does not inspect or analyze packet contents. It captures and saves. Analysis is done externally (Wireshark, Zeek, etc.).
- **Central Management:** No built-in fleet management, central configuration, or multi-host orchestration. Each instance is independent. Fleet management is left to existing tools (Ansible, SCCM, etc.).
- **Long-Term Storage:** dashcap is not a PCAP archival system. Saved captures should be moved to appropriate storage by external processes.
- **Packet Modification:** No packet rewriting, injection, or man-in-the-middle capabilities.
- **Global Resource Coordination:** Multiple instances do not share resource budgets or coordinate disk usage. Each instance manages its own pre-allocated space independently.

---

## 4. Architecture Overview

### 4.1 Data Flow

```
  Network Interface
        │
        ▼
  ┌───────────────────────┐
  │   Capture Engine       │   libpcap (Linux/macOS) / Npcap (Windows)
  │   + BPF Exclusions     │   Kernel-level filtering before userspace copy
  └───────────┬────────────┘
              │ raw packets
              ▼
  ┌───────────────────────┐
  │   Segment Writer       │   Writes pcapng segments of fixed size
  │   (Ring Rotation)      │   Overwrites oldest segment when ring is full
  └───────────┬────────────┘
              │ pcapng files
              ▼
  ┌───────────────────────┐
  │   Disk Ring            │   Pre-allocated segment files on disk
  │   segment_000.pcapng   │   Fixed footprint, no growth at runtime
  │   segment_001.pcapng   │
  │   ...                  │
  └───────────┬────────────┘
              │ on trigger
              ▼
  ┌───────────────────────┐
  │   Persistence Layer    │   Merges segments into zstd-compressed capture
  │   + Metadata           │   Writes trigger metadata + capture statistics
  └───────────────────────┘
```

### 4.2 Control Plane

```
  ┌──────────────────┐
  │   REST API        │
  │   (TCP)           │
  └────────┬──────────┘
           │
           ▼
  ┌──────────────────────────────────────────┐
  │          Trigger Dispatcher               │
  │   Receives requests, initiates persistence│
  └──────────────────────────────────────────┘
```

---

## 5. Core Design Decisions

### 5.1 Programming Language: Go

Go was chosen for the following reasons:

- **Performance:** Compiled language with low-latency goroutine scheduling, suitable for continuous packet processing.
- **gopacket Ecosystem:** Mature bindings for libpcap/Npcap (cross-platform) with a unified API.
- **Single Binary:** Compiles to a single binary. The only runtime dependency is libpcap (Linux/macOS) or Npcap (Windows). Ideal for deployment across heterogeneous endpoints.
- **Concurrency Model:** Goroutines and channels naturally model the parallel concerns of capture, buffer management, API serving, and trigger handling.
- **Cross-Compilation:** Native cross-compilation (GOOS/GOARCH) simplifies multi-platform builds, though cgo is required for libpcap bindings.

Alternatives considered: Python was rejected due to insufficient performance for high-throughput packet capture. Rust was considered but deemed excessive in complexity for this use case without proportional benefit.

### 5.2 Capture Backend

The capture backend uses `gopacket/pcap` which wraps libpcap (Linux/macOS) and Npcap (Windows), providing a single cross-platform API. The backend is abstracted behind a `capture.Source` interface, allowing alternative implementations in the future.

| Platform | Backend                   | Filter Mechanism   |
| -------- | ------------------------- | ------------------ |
| Linux    | libpcap                   | BPF filter         |
| Windows  | Npcap (WinPcap successor) | WinPcap BPF filter |
| macOS    | libpcap (BPF native)      | BPF native         |

**Windows Dependency:** Npcap must be installed on Windows hosts. This is the same dependency required by Wireshark and is standard in security tooling environments. Bundled installation (requiring an OEM license) may be considered for future fleet deployment.

*Future consideration:* On Linux, `gopacket/afpacket` could provide zero-copy capture via AF_PACKET v3 with kernel-managed MMAP ring buffers for higher throughput scenarios.

### 5.3 Output Format: pcapng

pcapng (PCAP Next Generation) was chosen over legacy pcap for the following advantages:

- Support for multiple network interfaces in a single file
- Extensible metadata via custom option blocks — ideal for embedding trigger information, hostname, interface details, and dashcap version
- Comment support on packets and sections, enabling annotation of triggered captures
- Native Wireshark support with full feature compatibility
- Section Header Blocks can carry dashcap-specific metadata without breaking compatibility

### 5.4 Ring Buffer Strategy: Pre-Allocated Disk Segments

The ring buffer is implemented as a fixed number of pre-allocated pcapng segment files on disk:

- **Pre-Allocation at Startup:** All segment files are created with their full size using `fallocate` (Linux), `fcntl(F_PREALLOCATE)` (macOS), or `SetEndOfFile` (Windows) when the instance starts. The total disk footprint is known and constant from the first second of operation.
- **No Runtime Growth:** Since all files are pre-allocated, the instance never increases its disk usage during operation. New data overwrites existing segments in-place.
- **Rotation by Overwrite:** When the ring is full, the oldest segment is truncated, re-allocated, and rewritten. No file creation or deletion during steady-state operation.
- **Startup Safety Check:** Before pre-allocating, the instance checks that sufficient free disk space remains after allocation. If the allocation would leave less than a configurable minimum (default: 1 GB or 5% of partition, whichever is larger), the instance refuses to start with a clear error message.

This approach eliminates the need for runtime disk monitoring, global coordination between instances, or complex cleanup logic. Each instance's footprint is exactly: `segment_count × segment_size`.

### 5.5 Trigger Mechanism

Triggers are issued exclusively via the REST API. This keeps the trigger interface uniform and allows full control over parameters (time window, metadata) that simpler mechanisms like signals cannot express.

When a trigger is received:

1. **Timestamp Recording:** The exact time of the trigger is recorded.
2. **Window Selection:** All segments covering the requested time window are identified. The window is either the configured default duration (e.g., 5 minutes before the trigger), an explicit duration, or an absolute start time (`since`).
3. **Persistence:** The identified segments are merged into a single zstd-compressed `capture.pcapng.zst` in a timestamped directory under `saved/`. Packet-header statistics (protocol distribution, top IPs/MACs) are collected during the merge pass.
4. **Metadata:** A JSON metadata file is written alongside the capture, containing trigger timestamp, source, requested duration, actual time window, capture path, and capture statistics.

---

## 6. Component Design

### 6.1 Capture Engine

The capture engine is defined by a Go interface (`capture.Source`) that abstracts the platform-specific capture mechanism. Each implementation provides methods to start capture, apply BPF filters, read packets, and close the capture handle.

The primary implementation uses `gopacket/pcap` for cross-platform compatibility (libpcap on Linux/macOS, Npcap on Windows).

BPF exclusion filters are compiled and applied at the capture source level, ensuring excluded traffic never enters userspace. Filters use standard tcpdump/BPF syntax. Hot-reloading of filters via the API is planned.

### 6.2 Segment Writer

The segment writer receives packets from the capture engine and writes them into the current active segment file in pcapng format:

- Each segment file has a maximum size (configurable, default 100 MB).
- When the current segment reaches its size limit, the writer advances to the next segment in the ring, re-initializing the file with a new Section Header Block.
- The writer tracks start and end timestamps, packet count, and byte count for each segment. This metadata is used for trigger window calculations.
- Embeds dashcap metadata (hostname, interface name, version) in pcapng Section Header Block options (`shb_userappl`, `shb_comment`), making captures self-describing and visible in Wireshark's section properties.

### 6.3 Ring Manager

The ring manager maintains the ring of segment files and handles rotation:

- Pre-allocates `N` segment files at startup (`N = buffer_size / segment_size`).
- Tracks the current write position (active segment index).
- Maintains a segment metadata table: index, file path, start timestamp, end timestamp, packet count, byte count.
- On rotation, resets the oldest segment's metadata and signals the writer to begin overwriting it.

Ring state is not persisted across restarts. After a restart, all segments are treated as empty. This is acceptable because a restart inherently creates a capture gap, making pre-restart data unreliable for trigger windows.

### 6.4 Trigger Dispatcher

The trigger dispatcher receives trigger requests from the REST API and orchestrates the save pipeline:

- Calculates which ring segments fall within the requested time window based on segment timestamps.
- Hands off the segment list to the persistence layer.
- Returns a trigger ID and status to the caller.
- Debouncing with a 5-second cooldown between triggers. Duplicate triggers within the cooldown window are rejected with a 429 response.

### 6.5 Persistence Layer

- Creates a timestamped directory under `saved/` (e.g., `saved/2026-02-28T14-30-00_api/`).
- Merges relevant ring segments into a single zstd-compressed `capture.pcapng.zst` file, sorted chronologically by segment start time. This handles ring buffer wraparound correctly — segments are reordered before merging.
- Collects packet-header statistics (protocol distribution, top IPs/MACs, time span) during the merge pass with constant memory overhead.
- Writes a `metadata.json` file containing trigger context, capture statistics, and the path to the merged capture file.

**Important:** Saved captures are outside the ring buffer's pre-allocated space. Each save operation consumes additional disk space. Cleanup of old saved captures is the responsibility of external tooling or a configurable retention policy (future phase).

### 6.6 REST API

The REST API provides programmatic control over the dashcap instance, served via HTTP on a configurable TCP port.

| Method | Endpoint               | Description                                                             |
| ------ | ---------------------- | ----------------------------------------------------------------------- |
| `GET`  | `/api/v1/health`       | Liveness check — returns `{"status": "ok"}`                             |
| `GET`  | `/api/v1/status`       | Instance status: interface, uptime, packet/byte counts                  |
| `POST` | `/api/v1/trigger`      | Trigger a save. Optional JSON body with `duration` or `since` overrides |
| `GET`  | `/api/v1/trigger/{id}` | Per-trigger status, metadata, and capture statistics (200/202/404)      |
| `GET`  | `/api/v1/triggers`     | List all trigger records (newest first)                                 |
| `GET`  | `/api/v1/ring`         | Per-segment metadata: index, path, timestamps, packet/byte counts       |

*Planned:*
- `PUT /api/v1/filters` — Update BPF exclusion filters (hot reload)

The API uses standard HTTP status codes and returns JSON. All endpoints except `/health` require bearer-token authentication (enabled by default). The token is auto-generated at startup or can be set via `--api-token` flag or `DASHCAP_API_TOKEN` environment variable. TLS is supported via `--tls-cert` / `--tls-key` flags. Authentication can be disabled with `--no-auth`.

---

## 7. Configuration

dashcap supports both CLI flags and a YAML configuration file. CLI flags always take precedence over config file values. Each instance reads its own configuration — there is no global shared config.

### 7.1 Configuration File

```yaml
# /etc/dashcap/dashcap.yaml (Linux)
# C:\ProgramData\dashcap\dashcap.yaml (Windows)

interface: eth0

buffer:
  size: 2GB              # Total ring buffer size
  segment_size: 100MB    # Individual segment size
  # Derived: segment_count = size / segment_size = 20

trigger:
  default_duration: 5m   # Default time window to save on trigger

safety:
  min_free_after_alloc: 5GB  # Min free disk after preallocation
  min_free_percent: 15       # Alternative: min % free after prealloc

api:
  tcp_port: 9800               # 0 = disabled, >0 = enable TCP REST API
  token: ""                    # Bearer token for API auth (empty = auto-generated)
  no_auth: false               # Disable API authentication entirely
  tls_cert: ""                 # Path to TLS certificate file
  tls_key: ""                  # Path to TLS private key file

capture:
  snaplen: 0             # 0 = full packets, >0 = truncate
  promiscuous: true

# BPF exclusion filters — exclude known-benign high-volume traffic from capture.
# Each entry has a name (for logging) and a BPF filter expression (tcpdump syntax).
# exclusions:
#   - name: backup_traffic
#     filter: "host 10.0.0.50 and port 443"
#   - name: dns_noise
#     filter: "udp port 53 and host 10.0.0.1"

storage:
  data_dir: /var/lib/dashcap/eth0   # Base directory for this instance

logging:
  level: info            # debug | info (debug enables verbose output)
```

### 7.2 CLI Flags

```bash
dashcap --interface eth0 --buffer-size 5GB --segment-size 100MB
dashcap --interface eth0 --api-port 9800
dashcap --interface "Wi-Fi" --config C:\dashcap\config.yaml
```

---

## 8. Multi-Instance Design

dashcap supports multiple independent instances on the same host, one per network interface. Each instance is a separate OS process with its own config, data directory, and API endpoint.

### 8.1 Instance Identity

Each instance is uniquely identified by its network interface name, sanitized for use in file paths and used as the key for locking, data directories, and service names.

### 8.2 Interface Locking

To prevent two instances from capturing on the same interface, each acquires an exclusive file lock at startup:

| Platform | Lock File                                               | Mechanism                               |
| -------- | ------------------------------------------------------- | --------------------------------------- |
| Linux    | `/run/dashcap/dashcap-{interface}.lock`                 | `flock()` — auto-released on crash      |
| Windows  | `C:\ProgramData\dashcap\locks\dashcap-{interface}.lock` | `LockFileEx()` — auto-released on crash |
| macOS    | `/run/dashcap/dashcap-{interface}.lock`                 | `flock()`                               |

The lock is acquired via `flock()` (Unix) or `LockFileEx()` (Windows). If already held, the new instance exits with an error identifying the interface and lock file path.

### 8.3 Data Directory Layout

```
/var/lib/dashcap/                  # Linux base
C:\ProgramData\dashcap\            # Windows base
├── eth0/                          # Per-interface directory
│   ├── ring/                      # Active ring segments
│   │   ├── segment_000.pcapng
│   │   ├── segment_001.pcapng
│   │   └── segment_002.pcapng
│   ├── saved/                     # Triggered captures
│   │   └── 2026-02-28T14-30-00_api/
│   │       ├── capture.pcapng.zst  # Merged capture, zstd-compressed
│   │       └── metadata.json
├── eth1/
│   └── ...
```

### 8.4 API Isolation

Each instance exposes its own API endpoint via TCP (configured with `--api-port`). The built-in client connects to a specific instance by host and port:

```bash
dashcap client status                                 # Default: localhost:9800
dashcap client status --host 10.0.0.5 --port 8080     # Remote instance
dashcap client trigger --host 10.0.0.5 --port 8080    # Trigger on remote
```

*Planned:* Unix socket support (Linux/macOS) for local communication without port conflicts.

### 8.5 Service Integration

**Linux (systemd template unit) — implemented:**

The template unit `dashcap@.service` uses `Type=notify` with `sd_notify(READY=1)`, runs as a dedicated `dashcap` user with ambient capabilities, and includes comprehensive sandboxing. API token initialization runs via `ExecStartPre=+` (as root) before the main process starts.

Install via package (RPM/DEB) or standalone:
```bash
sudo dashcap install-service          # standalone binary install
sudo systemctl enable --now dashcap@eth0
sudo usermod -aG dashcap <operator>   # grant trigger access
```

**Trigger mechanisms:**
- API: `dashcap client trigger` (reads token from `/etc/dashcap/api-token`)
- Signal: `systemctl kill --signal=USR1 dashcap@eth0` (default-duration only)

**Access control:** Members of the `dashcap` group can read the API token file (`0640 root:dashcap`) and trigger captures. Non-members cannot.

```bash
journalctl -u dashcap@eth0 -f
```

**Windows (service per interface)** *(planned — Phase 3):*

```powershell
dashcap.exe install --interface "Ethernet 2"
# → Registers service: dashcap_ethernet_2

dashcap.exe install --interface "Wi-Fi"
# → Registers service: dashcap_wi-fi
```

Each service would be a separate instance of the same binary with different arguments, using Go's `golang.org/x/sys/windows/svc` package.

---

## 9. Platform Abstraction

Platform-specific code is isolated using Go build tags (`//go:build linux`, `//go:build windows`, `//go:build darwin`). Each difference is encapsulated behind a common interface:

| Concern               | Linux                   | Windows                         | macOS                  |
| --------------------- | ----------------------- | ------------------------------- | ---------------------- |
| Capture               | libpcap                 | Npcap                           | libpcap                |
| File locking          | `flock()`               | `LockFileEx()`                  | `flock()`              |
| Pre-allocation        | `fallocate()`           | `SetEndOfFile()`                | `fcntl(F_PREALLOCATE)` |
| Free disk space       | `statfs()`              | `GetDiskFreeSpaceEx()`          | `statfs()`             |
| Local API *(planned)* | Unix socket             | —                               | Unix socket            |
| Service *(planned)*   | systemd (template unit) | Windows Service (SCM)           | launchd (plist)        |
| Default data dir      | `/var/lib/dashcap/`     | `C:\ProgramData\dashcap\`       | `/var/lib/dashcap/`    |
| Default lock dir      | `/run/dashcap/`         | `C:\ProgramData\dashcap\locks\` | `/run/dashcap/`        |
| Permissions           | `CAP_NET_RAW`           | Administrator / Npcap group     | root / BPF group       |

---

## 10. Disk Safety & Resource Management

Disk safety is achieved through **pre-allocation** rather than runtime monitoring. This is a deliberate design choice favoring simplicity and predictability.

### 10.1 Pre-Allocation Model

- **At Startup:** The ring manager calculates total required space (`segment_count × segment_size`) and pre-allocates all segment files.
- **Safety Check:** Before allocation, available disk space is queried. If the space remaining after allocation would fall below the configured minimum (`min_free_after_alloc` or `min_free_percent`), the instance refuses to start.
- **During Operation:** No additional disk space is consumed by the ring buffer. Segment rotation overwrites existing files in-place.

### 10.2 Saved Captures

Saved captures (triggered persistence) are the only source of disk growth at runtime. Each save merges the relevant ring segments into a new zstd-compressed `capture.pcapng.zst` file. Zstd compression typically reduces pcapng size by 3–5×, significantly lowering disk consumption per save. Cleanup of old saved captures is left to external tooling or a future retention policy.

### 10.3 Process Resource Limits

- On Linux: the systemd unit can set `MemoryMax` and `OOMScoreAdjust=500` so dashcap is killed before critical system services under memory pressure.
- On Windows: job objects can set memory limits for the service process.

---

## 11. Project Structure

```
dashcap/
├── cmd/dashcap/               # CLI entry point (Cobra)
│   ├── main.go                # Root command + capture daemon
│   └── client.go              # `dashcap client` subcommand group
├── internal/
│   ├── api/                   # REST API server
│   │   ├── server.go          # HTTP handlers + router
│   │   └── auth.go            # Bearer-token authentication middleware
│   ├── buffer/                # Ring buffer and segment management
│   │   ├── ring.go            # Ring manager (rotation, prealloc)
│   │   └── writer.go          # pcapng segment writer
│   ├── capture/               # Capture engine abstraction
│   │   ├── capture.go         # Source interface definition
│   │   └── pcap.go            # libpcap/Npcap implementation
│   ├── client/                # HTTP client for REST API (used by `dashcap client`)
│   │   └── client.go
│   ├── config/                # Runtime configuration + validation
│   │   ├── config.go          # Config struct, defaults, validation
│   │   ├── load.go            # YAML config file loading
│   │   └── size.go            # Human-readable size parsing (e.g. "2GB")
│   ├── persist/               # Save/export logic
│   │   ├── persist.go         # Segment merge + zstd compression
│   │   └── stats.go           # Capture statistics collection
│   ├── storage/               # Disk operations (platform-specific)
│   │   ├── storage.go         # Interface (prealloc, flock, free space)
│   │   ├── disk_unix.go       # flock, statfs (Linux/macOS)
│   │   ├── disk_windows.go    # LockFileEx, GetDiskFreeSpaceEx, SetEndOfFile
│   │   ├── prealloc_linux.go  # fallocate
│   │   └── prealloc_darwin.go # fcntl(F_PREALLOCATE)
│   └── trigger/               # Trigger dispatcher
│       └── trigger.go         # Receives API triggers, orchestrates saves
├── configs/
│   └── dashcap.example.yaml   # Example configuration
├── DESIGN.md                  # Architecture and design document
├── Makefile                   # Build, test, lint, cross-compile targets
├── go.mod
├── go.sum
└── README.md
```

---

## 12. Development Phases

### Phase 1 — MVP (Core Capture Loop) *(complete)*

*Goal: A working capture-to-disk pipeline with API trigger on Linux and Windows.*

- `gopacket/pcap` capture backend (cross-platform via libpcap/Npcap)
- pcapng segment writer with fixed-size segments and accurate byte tracking
- Ring buffer with pre-allocated segments and rotation
- REST API with `/health`, `/status`, `/trigger`, `/triggers`, `/ring` endpoints
- Bearer-token API authentication (auto-generated or user-supplied, TLS optional)
- Triggered saves merge segments into a single zstd-compressed `capture.pcapng.zst` (chronologically sorted)
- Capture statistics (protocol distribution, top IPs/MACs, time span) in metadata.json
- Custom trigger time windows (`duration` or `since` parameter)
- Interface locking via file locks
- Built-in CLI client (`dashcap client`) with human-readable and JSON output modes
- Structured logging via `log/slog` with `--debug` flag
- Startup disk space safety check (absolute + percentage-based thresholds)
- Platform-aware paths (Linux, macOS, Windows)
- Graceful shutdown with capture flush on SIGTERM/SIGINT
- Builds for Linux (amd64) and Windows (amd64)
- GitHub Actions CI (lint, test, build) and release workflows (cross-compile + GitHub Releases)
- Dependabot for Go module and Actions version updates

### Phase 2 — Configuration & Filters *(complete)*

*Goal: Production-ready configuration and traffic filtering.*

- YAML configuration file support with CLI flag precedence
- BPF exclusion filters (compile and apply from config/CLI, expose active filter in `/status`)
- `GET /api/v1/trigger/{id}` endpoint to retrieve trigger status, metadata, and capture statistics
- Trigger debouncing (5-second cooldown, 429 response)
- Embedding dashcap metadata in pcapng Section Header Block options (`shb_userappl`, `shb_comment`)

### Phase 3 — Production Hardening

*Goal: Reliable operation as a system service.*

*Done (Linux systemd):*
- systemd template unit (`dashcap@.service`) with `Type=notify` and `sd_notify(READY=1)` integration
- Dedicated `dashcap` system user/group with `CAP_NET_RAW`/`CAP_NET_ADMIN` (no root at runtime)
- API token file persistence (`/etc/dashcap/api-token`) with group-based access control (`0640 root:dashcap`)
- SIGUSR1 signal trigger for default-duration captures (works with `systemctl kill`)
- `dashcap install-service` self-install command for standalone binary deployments (embedded via `go:embed`)
- `dashcap token-init` subcommand for token file initialization (`ExecStartPre=+`)
- sysusers.d / tmpfiles.d drop-ins for RPM/DEB packaging
- Secure token logging (only auto-generated tokens logged; flag/env/file sources log mechanism only)
- Service hardening (ProtectSystem=strict, PrivateTmp, NoNewPrivileges, etc.)

*Remaining:*
- Windows Service registration and lifecycle management
- Process resource limits (systemd `MemoryMax`, Windows job objects)

### Phase 4 — Advanced Features

*Goal: Enhanced capabilities for power users and fleet deployments.*

- Hot-reload of BPF exclusion filters via API (`PUT /api/v1/filters`)
- Unix socket API endpoint (Linux/macOS)
- API rate limiting with configurable thresholds (`Retry-After` header)
- Saved capture retention policies (auto-cleanup by age/count)
- Prometheus metrics endpoint

### Phase 5 — Ecosystem Integration

*Goal: Deep integration with security tooling.*

- Splunk/Elastic webhook trigger receivers
- Automatic trigger via external anomaly detection
- Pluggable persistence targets (local directory, S3, SMB, SCP)
- ARM64 builds for embedded/IoT use cases

---

## 13. Open Questions & Future Considerations

### 13.1 Open Questions

- **~~Compression:~~** Resolved — saved captures are now zstd-compressed (`capture.pcapng.zst`) using streaming compression during the merge pass. Ring segments remain uncompressed for write latency.
- **Encryption at Rest:** Saved captures contain potentially sensitive packet data. Asymmetric encryption (e.g., age/NaCl with a public key) would allow encrypting on the endpoint without storing the decryption key locally — useful for fleet deployments where the endpoint may be compromised. Symmetric encryption would require key management on the host.

### 13.2 Future Directions

- **Central Dashboard:** A lightweight web UI or CLI tool aggregating status from multiple dashcap instances across hosts.
- **Intelligent Triggers:** Integration with ML-based anomaly detection to automatically trigger saves based on traffic patterns.
- **Selective Capture Modes:** Beyond full capture, support modes like "headers only" or "first N bytes per packet" to extend buffer duration.