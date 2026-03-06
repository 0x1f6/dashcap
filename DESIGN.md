# dashcap — A Network Packet Dashcam

> Full-Packet Capture with Ring Buffer and On-Demand Persistence

**Version:** 1.0 · **Date:** February 2026 · **Status:** Draft

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
- **Single Binary Deployment:** No runtime dependencies beyond Npcap on Windows. Easy to deploy across a fleet.

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
  │   Capture Engine       │   AF_PACKET (Linux) / Npcap (Windows) / libpcap (macOS)
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
  │   Persistence Layer    │   Copies/hardlinks triggered segments
  │   + Metadata           │   Writes trigger metadata (time, source, window)
  └───────────────────────┘
```

### 4.2 Control Plane

```
  ┌──────────────────┐     ┌─────────────────┐
  │   REST API        │     │  Signal Handler  │
  │   (TCP / Socket)  │     │  (Unix/Pipe)     │
  └────────┬──────────┘     └────────┬─────────┘
           │                         │
           ▼                         ▼
  ┌──────────────────────────────────────────┐
  │          Trigger Dispatcher               │
  │   Receives signals, initiates persistence │
  └──────────────────────────────────────────┘
```

---

## 5. Core Design Decisions

### 5.1 Programming Language: Go

Go was chosen for the following reasons:

- **Performance:** Compiled language with low-latency goroutine scheduling, suitable for continuous packet processing.
- **gopacket Ecosystem:** Mature bindings for both libpcap/Npcap (cross-platform) and AF_PACKET (Linux fast path), with a unified API.
- **Single Binary:** Statically linked binaries with no runtime dependencies. Ideal for deployment across heterogeneous endpoints.
- **Concurrency Model:** Goroutines and channels naturally model the parallel concerns of capture, buffer management, API serving, and trigger handling.
- **Cross-Compilation:** Native cross-compilation (GOOS/GOARCH) simplifies multi-platform builds, though cgo is required for libpcap bindings.

Alternatives considered: Python was rejected due to insufficient performance for high-throughput packet capture. Rust was considered but deemed excessive in complexity for this use case without proportional benefit.

### 5.2 Capture Backend

The capture backend is abstracted behind a common interface, with platform-specific implementations:

| Platform | Primary Backend           | Filter Mechanism     | Zero-Copy              |
| -------- | ------------------------- | -------------------- | ---------------------- |
| Linux    | AF_PACKET v3 (MMAP)       | cBPF at socket level | Yes (kernel MMAP ring) |
| Windows  | Npcap (WinPcap successor) | WinPcap BPF filter   | Npcap kernel buffer    |
| macOS    | libpcap (BPF native)      | BPF native           | Yes (BPF device)       |

The default cross-platform backend uses `gopacket/pcap` which wraps libpcap (Linux/macOS) and Npcap (Windows). On Linux, an optional fast path via `gopacket/afpacket` provides zero-copy capture with kernel-managed MMAP ring buffers for higher throughput scenarios.

**Windows Dependency:** Npcap must be installed on Windows hosts. This is the same dependency required by Wireshark and is standard in security tooling environments. For Phase 1, Npcap is treated as a prerequisite. Bundled installation (requiring an OEM license) may be considered for future fleet deployment.

### 5.3 Output Format: pcapng

pcapng (PCAP Next Generation) was chosen over legacy pcap for the following advantages:

- Support for multiple network interfaces in a single file
- Extensible metadata via custom option blocks — ideal for embedding trigger information, hostname, interface details, and dashcap version
- Comment support on packets and sections, enabling annotation of triggered captures
- Native Wireshark support with full feature compatibility
- Section Header Blocks can carry dashcap-specific metadata without breaking compatibility

### 5.4 Ring Buffer Strategy: Pre-Allocated Disk Segments

The ring buffer is implemented as a fixed number of pre-allocated pcapng segment files on disk:

- **Pre-Allocation at Startup:** All segment files are created with their full size using `fallocate` (Linux) or `SetEndOfFile` (Windows) when the instance starts. The total disk footprint is known and constant from the first second of operation.
- **No Runtime Growth:** Since all files are pre-allocated, the instance never increases its disk usage during operation. New data overwrites existing segments in-place.
- **Rotation by Overwrite:** When the ring is full, the oldest segment is truncated, re-allocated, and rewritten. No file creation or deletion during steady-state operation.
- **Startup Safety Check:** Before pre-allocating, the instance checks that sufficient free disk space remains after allocation. If the allocation would leave less than a configurable minimum (default: 1 GB or 5% of partition, whichever is larger), the instance refuses to start with a clear error message.

This approach eliminates the need for runtime disk monitoring, global coordination between instances, or complex cleanup logic. Each instance's footprint is exactly: `segment_count × segment_size`.

### 5.5 Trigger Mechanism

The trigger system supports multiple input channels:

| Method          | Platform     | Use Case                                               |
| --------------- | ------------ | ------------------------------------------------------ |
| REST API (HTTP) | All          | SIEM/SOAR integration, remote trigger, scripting       |
| Unix Socket     | Linux, macOS | Local trigger without network exposure                 |
| Named Pipe      | Windows      | Local trigger, PowerShell scripting                    |
| SIGUSR1 Signal  | Linux, macOS | Simplest possible local trigger                        |
| CLI Command     | All          | Human-initiated via `dashcap trigger --interface eth0` |

When a trigger is received:

1. **Timestamp Recording:** The exact time of the trigger is recorded.
2. **Pre-Trigger Window:** All segments covering the configured pre-trigger duration (e.g., 5 minutes before the trigger) are identified.
3. **Post-Trigger Capture:** Capture continues for the configured post-trigger duration (e.g., 60 seconds after), writing to a separate temporary segment outside the ring.
4. **Persistence:** The identified pre-trigger segments and the post-trigger segment are copied (or hardlinked) to the `saved/` directory.
5. **Metadata:** A JSON metadata file is written alongside the saved segments, containing trigger timestamp, source (API/signal/CLI), pre/post window durations, interface name, and any user-provided context.

---

## 6. Component Design

### 6.1 Capture Engine

The capture engine is defined by a Go interface (`capture.Source`) that abstracts the platform-specific capture mechanism. Each implementation provides methods to start capture, apply BPF filters, read packets, and close the capture handle.

The primary implementation uses `gopacket/pcap` for cross-platform compatibility. A Linux-specific implementation using `gopacket/afpacket` provides a zero-copy fast path selectable via configuration.

BPF exclusion filters are compiled and applied at the capture source level, ensuring excluded traffic never enters userspace. Filters use standard tcpdump/BPF syntax and can be hot-reloaded via the API by compiling a new filter and atomically swapping it on the capture socket.

### 6.2 Segment Writer

The segment writer receives packets from the capture engine and writes them into the current active segment file in pcapng format:

- Each segment file has a maximum size (configurable, default 100 MB).
- When the current segment reaches its size limit, the writer advances to the next segment in the ring, re-initializing the file with a new Section Header Block.
- The Section Header Block contains dashcap metadata: hostname, interface name, dashcap version, and the segment's start timestamp.
- The writer tracks start and end timestamps of each segment for trigger window calculations.

### 6.3 Ring Manager

The ring manager maintains the ring of segment files and handles rotation:

- Pre-allocates `N` segment files at startup (`N = buffer_size / segment_size`).
- Tracks the current write position (active segment index).
- Maintains a segment metadata table: index, file path, start timestamp, end timestamp, packet count, byte count.
- On rotation, resets the oldest segment's metadata and signals the writer to begin overwriting it.
- Persists ring state (current position, segment metadata) to a state file for recovery after restarts.

### 6.4 Trigger Dispatcher

The trigger dispatcher multiplexes trigger signals from all input channels into a unified trigger pipeline:

- Validates the trigger request (rejects duplicates within a debounce window).
- Calculates which ring segments fall within the pre-trigger time window based on segment timestamps.
- Initiates post-trigger capture if configured (writes to a temporary segment outside the ring).
- Hands off the segment list to the persistence layer.
- Returns a trigger ID and status to the caller (for API triggers).

### 6.5 Persistence Layer

- Creates a timestamped directory under `saved/` (e.g., `saved/2026-02-28T14-30-00_api/`).
- Merges relevant ring segments into a single `capture.pcapng` file, sorted chronologically by segment start time. This handles ring buffer wraparound correctly — segments are reordered before merging.
- Writes a `metadata.json` file containing trigger context and the path to the merged capture file.

**Important:** Saved captures are outside the ring buffer's pre-allocated space. Each save operation consumes additional disk space (unless hardlinked). The persistence layer checks available disk space before copying and warns if space is low. Cleanup of old saved captures is the responsibility of external tooling or a configurable retention policy (future phase).

### 6.6 REST API

The REST API provides programmatic control over the dashcap instance, served via HTTP on a configurable TCP port and/or Unix socket / Named Pipe.

| Method | Endpoint           | Description                                                                        |
| ------ | ------------------ | ---------------------------------------------------------------------------------- |
| `GET`  | `/api/v1/status`   | Instance status: running, interface, buffer usage, uptime, active filters          |
| `POST` | `/api/v1/trigger`  | Trigger a save. Accepts optional JSON body with context, pre/post window overrides |
| `GET`  | `/api/v1/triggers` | List recent triggers and their status (pending, completed, failed)                 |
| `GET`  | `/api/v1/ring`     | Ring buffer status: segment count, current position, timestamps per segment        |
| `PUT`  | `/api/v1/filters`  | Update BPF exclusion filters (hot reload)                                          |
| `GET`  | `/api/v1/health`   | Health check endpoint for monitoring integration                                   |

The API uses standard HTTP status codes and returns JSON. All endpoints except `/health` require bearer-token authentication (enabled by default). The token is auto-generated at startup or can be set via `--api-token` flag or `DASHCAP_API_TOKEN` environment variable. TLS is supported via `--tls-cert` / `--tls-key` flags. Authentication can be disabled with `--no-auth`.

---

## 7. Configuration

dashcap is configured via a YAML file, CLI flags, or both. CLI flags override config file values. Each instance reads its own configuration — there is no global shared config.

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
  debounce: 10s          # Min time between triggers

safety:
  min_free_after_alloc: 1GB   # Min free disk after preallocation
  min_free_percent: 5         # Alternative: min % free after prealloc

api:
  socket: /run/dashcap/eth0.sock   # Unix socket (Linux/macOS)
  # pipe: dashcap-eth0             # Named pipe name (Windows)
  tcp_port: 0                      # 0 = disabled, >0 = enable TCP API

capture:
  backend: auto          # auto | pcap | afpacket (Linux only)
  snaplen: 0             # 0 = full packets, >0 = truncate
  promiscuous: true

exclusions:
  - name: backup_traffic
    filter: "host 10.0.0.50 and port 443"
  - name: dns_noise
    filter: "udp port 53 and host 10.0.0.1"

storage:
  data_dir: /var/lib/dashcap/eth0   # Base directory for this instance
  saved_dir: saved                   # Subdir for triggered captures

logging:
  level: info            # debug | info | warn | error
  format: json           # json | text
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
| macOS    | `/var/run/dashcap/dashcap-{interface}.lock`             | `flock()`                               |

The lock file contains the PID of the owning process. If already held, the new instance exits with:

```
Error: interface eth0 already captured by dashcap (PID 4271)
```

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
│   │       ├── capture.pcapng     # Merged capture (all segments)
│   │       └── metadata.json
│   └── dashcap.state              # Ring state for recovery
├── eth1/
│   └── ...
```

### 8.4 API Isolation

Each instance exposes its own API endpoint. The default is a Unix socket / Named Pipe named after the interface (no port conflicts). An optional TCP port can be configured per instance for remote access.

```bash
dashcap trigger --interface eth0          # Talks to eth0's socket/pipe
dashcap status  --interface eth1          # Talks to eth1's socket/pipe
dashcap status  --api-url http://host:9800  # Direct TCP connection
```

### 8.5 Service Integration

**Linux (systemd template unit):**

```ini
# /etc/systemd/system/dashcap@.service
[Unit]
Description=dashcap packet capture on %i
After=network-online.target

[Service]
Type=notify
ExecStart=/usr/local/bin/dashcap --interface %i
RuntimeDirectory=dashcap
StateDirectory=dashcap/%i
AmbientCapabilities=CAP_NET_RAW CAP_NET_ADMIN
ProtectSystem=strict
ReadWritePaths=/var/lib/dashcap/%i

[Install]
WantedBy=multi-user.target
```

```bash
systemctl enable --now dashcap@eth0
systemctl enable --now dashcap@eth1
journalctl -u dashcap@eth0 -f
```

**Windows (service per interface):**

```powershell
dashcap.exe install --interface "Ethernet 2"
# → Registers service: dashcap_ethernet_2

dashcap.exe install --interface "Wi-Fi"
# → Registers service: dashcap_wi-fi
```

Each service is a separate instance of the same binary with different arguments, using Go's `golang.org/x/sys/windows/svc` package.

---

## 9. Platform Abstraction

Platform-specific code is isolated using Go build tags (`//go:build linux`, `//go:build windows`, `//go:build darwin`). Each difference is encapsulated behind a common interface:

| Concern             | Linux                   | Windows                         | macOS                  |
| ------------------- | ----------------------- | ------------------------------- | ---------------------- |
| Capture (fast path) | AF_PACKET v3 MMAP       | N/A                             | N/A                    |
| Capture (default)   | libpcap                 | Npcap                           | libpcap                |
| File locking        | `flock()`               | `LockFileEx()`                  | `flock()`              |
| Pre-allocation      | `fallocate()`           | `SetEndOfFile()`                | `fcntl(F_PREALLOCATE)` |
| Free disk space     | `statfs()`              | `GetDiskFreeSpaceEx()`          | `statfs()`             |
| Local API           | Unix socket             | Named Pipe                      | Unix socket            |
| Signal trigger      | SIGUSR1                 | N/A                             | SIGUSR1                |
| Service             | systemd (template unit) | Windows Service (SCM)           | launchd (plist)        |
| Default data dir    | `/var/lib/dashcap/`     | `C:\ProgramData\dashcap\`       | `/var/lib/dashcap/`    |
| Default lock dir    | `/run/dashcap/`         | `C:\ProgramData\dashcap\locks\` | `/var/run/dashcap/`    |
| Permissions         | `CAP_NET_RAW`           | Administrator / Npcap group     | root / BPF group       |

---

## 10. Disk Safety & Resource Management

Disk safety is achieved through **pre-allocation** rather than runtime monitoring. This is a deliberate design choice favoring simplicity and predictability.

### 10.1 Pre-Allocation Model

- **At Startup:** The ring manager calculates total required space (`segment_count × segment_size`) and pre-allocates all segment files.
- **Safety Check:** Before allocation, available disk space is queried. If the space remaining after allocation would fall below the configured minimum (`min_free_after_alloc` or `min_free_percent`), the instance refuses to start.
- **During Operation:** No additional disk space is consumed by the ring buffer. Segment rotation overwrites existing files in-place.

### 10.2 Saved Captures

Saved captures (triggered persistence) are the only source of disk growth at runtime. Two mitigations:

- **Hardlinks:** Where the filesystem supports it, hardlinks are used. A hardlinked save consumes zero additional space until the ring overwrites the original segment.
- **Pre-Save Check:** Before copying segments (when hardlinks are not possible), the persistence layer checks available disk space and refuses to save if it would breach the minimum free threshold.

### 10.3 Process Resource Limits

- On Linux: the systemd unit can set `MemoryMax` and `OOMScoreAdjust=500` so dashcap is killed before critical system services under memory pressure.
- On Windows: job objects can set memory limits for the service process.

---

## 11. Project Structure

```
dashcap/
├── cmd/
│   └── dashcap/                # Main entry point
│       └── main.go
├── internal/
│   ├── capture/                # Capture engine abstraction
│   │   ├── capture.go          # Interface definition
│   │   ├── pcap.go             # libpcap/Npcap (all platforms)
│   │   └── afpacket_linux.go   # AF_PACKET fast path (Linux only)
│   ├── buffer/                 # Ring buffer and segment management
│   │   ├── ring.go             # Ring manager (rotation, state, prealloc)
│   │   └── writer.go           # pcapng segment writer
│   ├── trigger/                # Trigger handling
│   │   ├── trigger.go          # Interface + dispatcher
│   │   ├── api.go              # REST API trigger source
│   │   ├── signal_unix.go      # SIGUSR1 (Linux/macOS)
│   │   └── pipe_windows.go     # Named Pipe trigger (Windows)
│   ├── persist/                # Save/export logic
│   │   └── persist.go          # Copy/hardlink + metadata
│   ├── storage/                # Disk operations (platform-specific)
│   │   ├── storage.go          # Interface
│   │   ├── disk_unix.go        # statfs, fallocate, flock
│   │   └── disk_windows.go     # Win32 APIs
│   ├── service/                # OS service integration
│   │   ├── service.go          # Interface
│   │   ├── systemd_linux.go    # sd_notify integration
│   │   ├── svc_windows.go      # Windows SCM integration
│   │   └── launchd_darwin.go   # launchd integration
│   ├── filter/                 # BPF filter compilation + management
│   │   └── filter.go
│   ├── api/                    # REST API server
│   │   └── server.go
│   └── config/                 # Configuration loading + validation
│       └── config.go
├── configs/
│   └── dashcap.example.yaml    # Example configuration
├── deployments/
│   ├── dashcap@.service        # systemd template unit
│   ├── install.ps1             # Windows installer script
│   └── com.dashcap.plist       # launchd plist
├── go.mod
├── go.sum
├── Makefile                    # Build targets per platform
└── README.md
```

---

## 12. Development Phases

### Phase 1 — MVP (Core Capture Loop) *(complete)*

*Goal: A working capture-to-disk pipeline with manual trigger on Linux and Windows.*

- `gopacket/pcap` capture backend (cross-platform via libpcap/Npcap)
- pcapng segment writer with fixed-size segments and accurate byte tracking
- Ring buffer with pre-allocated segments and rotation
- REST API with `/trigger`, `/status`, `/health`, `/ring`, `/triggers` endpoints
- Bearer-token API authentication (auto-generated or user-supplied, TLS optional)
- Triggered saves merge segments into a single `capture.pcapng` (chronologically sorted)
- Interface locking via file locks
- CLI: `dashcap --interface eth0 --buffer-size 2GB`
- Startup disk space safety check (absolute + percentage-based thresholds)
- Platform-aware paths (Linux, macOS, Windows)
- Builds for Linux (amd64) and Windows (amd64)

### Phase 2 — Configuration & Filters

*Goal: Production-ready configuration and traffic filtering.*

- YAML configuration file support
- BPF exclusion filters (compile and apply from config, expose active filter in `/status`)
- Pre/post trigger time windows
- Trigger metadata and saved capture management
- Capture metadata extracted from pcapng (protocols, packet counts, IP/MAC addresses)
- CLI client subcommand (`dashcap trigger`, `dashcap status`, etc.) — same binary acts as API client
- Configurable buffer limits (size, segment count, duration)
- Hardlink-based saves where supported

### Phase 3 — Production Hardening

*Goal: Reliable operation as a system service.*

- systemd template unit with `sd_notify` integration
- Windows Service registration and lifecycle management
- Health endpoint for monitoring
- Structured logging (JSON format)
- Ring state persistence and recovery after restart
- Graceful shutdown with capture flush

### Phase 4 — Advanced Features

*Goal: Enhanced capabilities for power users and fleet deployments.*

- AF_PACKET fast path for Linux (optional, configurable)
- Hot-reload of BPF exclusion filters via API
- Unix socket / Named Pipe API endpoints
- Trigger debouncing and API rate limiting (429 response, `Retry-After` header)
- Saved capture retention policies (auto-cleanup by age/count)
- Prometheus metrics endpoint
- ~~Multi-segment merge for saved captures (single output file)~~ *(moved to Phase 1)*
- SIGUSR1 / signal-based trigger (Unix platforms)

### Phase 5 — Ecosystem Integration

*Goal: Deep integration with security tooling.*

- Splunk/Elastic webhook trigger receivers
- Automatic trigger via external anomaly detection
- Pluggable persistence targets (local directory, S3, SMB, SCP)
- macOS / launchd support
- ARM64 builds for embedded/IoT use cases

---

## 13. Open Questions & Future Considerations

### 13.1 Open Questions

- **Npcap Licensing:** For commercial fleet deployment on Windows, an Npcap OEM license may be required. Evaluate whether bundled installation is needed or if Npcap-as-prerequisite is acceptable.
- **pcapng Library:** Evaluate Go pcapng writing libraries (`gopacket/pcapgo`, or custom implementation) for correctness and performance. May need a minimal custom pcapng writer for full control over metadata blocks.
- **Compression:** pcapng supports optional compression (e.g., zstd). Could significantly reduce disk footprint but adds CPU overhead. Evaluate for Phase 3+.
- **Encryption:** Saved captures contain potentially sensitive packet data. Consider optional encryption at rest.
- **API Authentication:** ~~For remote TCP API access, an auth mechanism is needed.~~ Resolved: Bearer-token authentication is implemented with auto-generated tokens, CLI/env override, and TLS support.

### 13.2 Future Directions

- **Central Dashboard:** A lightweight web UI or CLI tool aggregating status from multiple dashcap instances across hosts.
- **Intelligent Triggers:** Integration with ML-based anomaly detection to automatically trigger saves based on traffic patterns.
- **Selective Capture Modes:** Beyond full capture, support modes like "headers only" or "first N bytes per packet" to extend buffer duration.
- **Kernel-Level Capture (eBPF):** For extreme-throughput scenarios (10G+), an eBPF-based capture path could provide maximum performance with flexible in-kernel filtering.