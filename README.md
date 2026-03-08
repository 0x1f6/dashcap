# dashcap

A network packet dashcam — continuous full-packet capture with on-demand persistence.

> **Work in progress.** dashcap is under active development and not yet stable or suitable for production use. APIs, CLI flags, and on-disk formats may change without notice.

dashcap continuously captures all network traffic into a pre-allocated ring buffer of pcapng segments. When triggered (via the REST API), it saves the relevant capture window — including traffic from *before* the trigger — for later analysis. Think of it as a dashcam for your network: always recording, only saving when something happens.

## How It Works

```
Network Interface → Capture Engine → Segment Writer → Ring Buffer (fixed size, overwrites oldest)
                                                            ↓ on trigger
                                                      Saved Captures (pcapng.zst + metadata.json)
```

- Packets are captured via libpcap (Linux/macOS) or Npcap (Windows) and written into fixed-size pcapng segment files
- Segments rotate in a ring — when the buffer is full, the oldest segment is overwritten
- All disk space is pre-allocated at startup; the footprint is constant at `segment_count * segment_size`
- A trigger merges the relevant time window into a single zstd-compressed `capture.pcapng.zst` in a `saved/` directory with metadata and capture statistics

## Status

**Phase 1 (MVP) is complete.** The core capture-to-disk pipeline with REST API triggers works on Linux. See [DESIGN.md](DESIGN.md) for the full roadmap.

What's implemented:
- `gopacket/pcap` capture backend (cross-platform)
- pcapng segment writer with accurate byte tracking
- Ring buffer with pre-allocated segments and rotation
- REST API with `/trigger`, `/status`, `/health`, `/ring`, `/triggers` endpoints
- Bearer-token API authentication (enabled by default, auto-generated token)
- TLS support for the API server (`--tls-cert` / `--tls-key`)
- Built-in CLI client (`dashcap client`) with human-readable and JSON output modes
- Triggered saves merge segments into a single zstd-compressed `capture.pcapng.zst` file
- Capture statistics (protocol distribution, top IPs/MACs, time span) in metadata.json
- Interface locking (one instance per interface)
- Disk safety checks (absolute + percentage-based free space thresholds)
- Platform-aware paths (Linux, macOS, Windows)

## Prerequisites

| Platform | Requirement |
|----------|-------------|
| Linux    | `libpcap-dev` (`apt install libpcap-dev` or `dnf install libpcap-devel`) |
| Windows  | [Npcap](https://npcap.com/) installed (same as Wireshark) |
| macOS    | libpcap (ships with Xcode Command Line Tools) |
| All      | Go 1.25+ with CGO enabled |

## Building

```bash
make build          # → bin/dashcap
```

The binary requires `CGO_ENABLED=1` because gopacket links against libpcap. This is handled automatically by the Makefile.

For cross-compilation (requires platform-specific libpcap/Npcap SDK):

```bash
make cross          # → dist/dashcap-linux-amd64, dist/dashcap-windows-amd64.exe, dist/dashcap-darwin-arm64
```

## Quick Start

This example uses the loopback interface (`lo`) so you can try it without any special hardware or permissions beyond `CAP_NET_RAW`.

### 1. Build and start dashcap

```bash
make build

# Start with a small ring buffer (10 MB total, 1 MB segments = 10 segments)
sudo bin/dashcap -i lo --buffer-size 10MB --segment-size 1MB --api-port 9800
```

You should see:

```
time=... level=INFO msg="API token generated" token=<generated-token>
time=... level=INFO msg="dashcap starting" version=dev interface=lo
time=... level=INFO msg="ring buffer configured" segments=10 segment_mb=1 total_mb=10
time=... level=INFO msg="ring pre-allocated" path=/var/lib/dashcap/lo
time=... level=WARN msg="API auth enabled without TLS — tokens sent in cleartext"
time=... level=INFO msg="REST API listening" port=9800 proto=HTTP
```

Copy the API token from the output — you'll need it for all API requests (except `/health`). To use a predictable token, pass `--api-token <value>` or set `DASHCAP_API_TOKEN=<value>`.

### 2. Generate some traffic

In a second terminal:

```bash
ping -c 100 127.0.0.1
# or
curl http://127.0.0.1:9800/api/v1/health
```

### 3. Check status

Using the built-in client (recommended):

```bash
dashcap client status --token <token>
```

Or with curl:

```bash
curl -s -H "Authorization: Bearer <token>" http://127.0.0.1:9800/api/v1/status | python3 -m json.tool
```

```json
{
    "interface": "lo",
    "uptime": "42s",
    "segment_count": 10,
    "total_packets": 200,
    "total_bytes": 19600
}
```

### 4. Trigger a save

```bash
dashcap client trigger --token <token>
```

Or with curl:

```bash
curl -s -X POST -H "Authorization: Bearer <token>" http://127.0.0.1:9800/api/v1/trigger | python3 -m json.tool
```

```json
{
    "id": "1740960600000000000-1",
    "timestamp": "2026-03-02T22:30:00Z",
    "source": "api",
    "status": "pending"
}
```

### 5. Inspect the saved capture

```bash
ls /var/lib/dashcap/lo/saved/
# → 2026-03-02T22-30-00_api/

cat /var/lib/dashcap/lo/saved/2026-03-02T22-30-00_api/metadata.json
# → trigger metadata with capture path and statistics

# Open in Wireshark ≥ 3.6 (handles .pcapng.zst natively):
wireshark /var/lib/dashcap/lo/saved/2026-03-02T22-30-00_api/capture.pcapng.zst

# Or decompress first for older Wireshark versions:
# zstd -d capture.pcapng.zst
```

### 6. View ring buffer state

```bash
dashcap client ring --token <token>
```

Or with curl:

```bash
curl -s -H "Authorization: Bearer <token>" http://127.0.0.1:9800/api/v1/ring | python3 -m json.tool
```

### 7. Stop dashcap

Press `Ctrl+C` or send `SIGTERM` — dashcap flushes the active segment and exits cleanly.

## CLI Reference

### Server Flags

These flags configure the capture daemon (the root `dashcap` command):

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | *(see below)* | Path to YAML config file |
| `-i`, `--interface` | *(required)* | Network interface to capture on |
| `--buffer-size` | `2GB` | Total ring buffer size (e.g. `2GB`, `500MB`) |
| `--segment-size` | `100MB` | Size of each ring segment (e.g. `100MB`, `1MB`) |
| `--data-dir` | `/var/lib/dashcap/<interface>` | Data directory for ring and saved captures |
| `--api-port` | `9800` | TCP port for REST API (`0` = disabled) |
| `--api-token` | *(auto-generated)* | Bearer token for API authentication |
| `--no-auth` | `false` | Disable API authentication entirely |
| `--tls-cert` | | Path to TLS certificate file (requires `--tls-key`) |
| `--tls-key` | | Path to TLS private key file (requires `--tls-cert`) |
| `--default-duration` | `5m` | Default time window to save on trigger |
| `--promiscuous` | `true` | Enable promiscuous mode on the interface |
| `--snaplen` | `0` | Snapshot length (`0` = full packets) |
| `--exclude` | | BPF exclusion filter expression, tcpdump syntax (e.g. `"host 10.0.0.50"`) |
| `--debug` | `false` | Enable debug-level logging (ring rotations, packet details) |

### Client Flags

These flags configure the `dashcap client` subcommand group for interacting with a running dashcap instance:

| Flag | Default | Description |
|------|---------|-------------|
| `--host` | `localhost` | API server host |
| `--port` | `9800` | API server port |
| `--token` | | Bearer token (falls back to `$DASHCAP_API_TOKEN`) |
| `--tls` | `false` | Use HTTPS |
| `--tls-skip-verify` | `false` | Skip TLS certificate verification |
| `--pretty` | *(auto)* | Force human-readable output |
| `--json` | *(auto)* | Force JSON output |

Output mode is auto-detected: pretty when stdout is a TTY, JSON when piped. Use `--pretty` or `--json` to override.

### Environment Variables

| Variable | Description |
|----------|-------------|
| `DASHCAP_API_TOKEN` | API token — used by both server (overridden by `--api-token`) and client (overridden by `--token`) |

### Subcommands

```bash
dashcap version                # Print version, commit, and build time
dashcap client health          # Check server health
dashcap client status          # Show capture status (interface, uptime, packets, bytes)
dashcap client trigger         # Trigger a capture save
dashcap client trigger --duration 30s   # Trigger with custom duration
dashcap client trigger --since 2024-01-01T00:00:00Z  # Trigger from specific time
dashcap client triggers        # List trigger history
dashcap client ring            # Show ring buffer segment metadata
```

## REST API

All endpoints return JSON. The API listens on the port specified by `--api-port`.

**Authentication:** All endpoints except `/health` require a bearer token. Include the header `Authorization: Bearer <token>` with every request. The token is printed to stderr at startup. Disable auth with `--no-auth`.

**TLS:** Pass `--tls-cert` and `--tls-key` to enable HTTPS. Without TLS, tokens are sent in cleartext (a warning is logged).

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/api/v1/health` | No | Liveness check — returns `{"status": "ok"}` |
| `GET` | `/api/v1/status` | Yes | Instance status: interface, uptime, packet/byte counts |
| `POST` | `/api/v1/trigger` | Yes | Trigger a save of the pre-trigger window |
| `GET` | `/api/v1/triggers` | Yes | List all trigger records (newest first) |
| `GET` | `/api/v1/ring` | Yes | Per-segment metadata: index, path, timestamps, packet/byte counts |

## Configuration

dashcap supports both CLI flags and a YAML configuration file. CLI flags always take precedence over config file values.

**Config file discovery:**
- `--config <path>` — use the specified file (error if not found)
- No `--config` — automatically loads from the platform default path if the file exists:
  - Linux/macOS: `/etc/dashcap/dashcap.yaml`
  - Windows: `C:\ProgramData\dashcap\dashcap.yaml`
- If no config file is found, dashcap runs with CLI flags and built-in defaults only

An example configuration file is provided at [`configs/dashcap.example.yaml`](configs/dashcap.example.yaml).

## Project Structure

```
dashcap/
├── cmd/dashcap/           # CLI entry point (Cobra)
├── internal/
│   ├── api/               # REST API server (net/http)
│   ├── buffer/            # Ring manager + pcapng segment writer
│   ├── capture/           # Packet capture abstraction (gopacket/pcap)
│   ├── client/            # HTTP client for the REST API (used by `dashcap client`)
│   ├── config/            # Runtime configuration + validation
│   ├── persist/           # Save triggered captures to disk
│   ├── storage/           # Platform-specific disk ops (prealloc, flock, free space)
│   └── trigger/           # Trigger dispatcher (API-triggered saves)
├── configs/               # Example configuration
├── DESIGN.md              # Full architecture and design document
├── Makefile               # Build, test, lint, cross-compile targets
└── go.mod
```

## CI/CD

GitHub Actions runs on every push and PR to `main`:

- **CI** (`.github/workflows/ci.yml`): lint (`golangci-lint`), test (race detector + coverage), build verification
- **Release** (`.github/workflows/release.yml`): triggered on `v*` tags or manually. Builds Linux amd64 and Windows amd64 binaries, creates a GitHub Release with SHA256 checksums.
- **Dependabot** (`.github/dependabot.yml`): weekly updates for Go modules and GitHub Actions versions.

## Development

```bash
make test           # Run tests with race detector
make lint           # Run golangci-lint
make fmt            # Format code (gofmt + goimports)
make cover          # Generate coverage report (coverage.html)
```
