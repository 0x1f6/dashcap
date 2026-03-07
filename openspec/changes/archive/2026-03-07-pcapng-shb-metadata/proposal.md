## Why

Captured pcapng files currently contain no dashcap-specific context. When a saved capture is shared or opened in Wireshark, there is no way to tell which dashcap version produced it, which host or interface it came from. Embedding this metadata in the pcapng Section Header Block (SHB) makes every capture self-describing — visible in Wireshark's "Capture File Properties" without any external metadata file.

## What Changes

- Write `shb_userappl` option in every SHB with the value `dashcap <version>` (e.g. `dashcap v1.2.0`)
- Write `shb_comment` option in every SHB with a structured comment containing hostname and interface name (e.g. `host=webserver01 interface=eth0`)
- Both ring segment files and merged saved captures carry this metadata
- The version string, hostname, and interface name are passed into the segment writer at construction time

## Capabilities

### New Capabilities
- `shb-metadata`: Embed dashcap version, hostname, and interface name in pcapng Section Header Block options

### Modified Capabilities

## Impact

- `internal/buffer/writer.go`: `NewSegmentWriter` gains parameters for version, hostname, and interface name; passes SHB options to `pcapgo.NewNgWriter`
- `internal/buffer/ring.go`: Passes the new parameters when creating segment writers
- `internal/persist/persist.go`: Merged captures inherit SHB metadata from the writer
- `cmd/dashcap/main.go`: Supplies version/hostname/interface to the ring/writer
