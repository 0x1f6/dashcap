## ADDED Requirements

### Requirement: SHB Application option identifies dashcap
Every pcapng file produced by dashcap SHALL set the Section Header Block `shb_userappl` option to `dashcap <version>`, where `<version>` is the build version string (e.g. `dashcap v1.2.0`).

#### Scenario: Ring segment contains application metadata
- **WHEN** a new ring segment file is created
- **THEN** the pcapng Section Header Block `shb_userappl` option SHALL be set to `dashcap <version>`

#### Scenario: Merged saved capture contains application metadata
- **WHEN** a triggered save produces a merged `capture.pcapng`
- **THEN** the pcapng Section Header Block `shb_userappl` option SHALL be set to `dashcap <version>`

### Requirement: SHB Comment contains hostname and interface
Every pcapng file produced by dashcap SHALL set the Section Header Block `shb_comment` option to a string in the format `host=<hostname> interface=<interface>`, where `<hostname>` is the OS hostname and `<interface>` is the capture interface name.

#### Scenario: Ring segment contains host and interface context
- **WHEN** a new ring segment file is created with hostname `webserver01` and interface `eth0`
- **THEN** the pcapng Section Header Block `shb_comment` option SHALL be `host=webserver01 interface=eth0`

#### Scenario: Merged saved capture contains host and interface context
- **WHEN** a triggered save produces a merged `capture.pcapng` for hostname `webserver01` and interface `eth0`
- **THEN** the pcapng Section Header Block `shb_comment` option SHALL be `host=webserver01 interface=eth0`

### Requirement: SHB metadata passed at construction time
The segment writer SHALL accept version, hostname, and interface name as construction parameters. These values SHALL NOT be resolved internally by the writer.

#### Scenario: Writer receives metadata from caller
- **WHEN** `NewSegmentWriter` is called with version `v1.2.0`, hostname `node42`, and interface `lo`
- **THEN** the resulting pcapng file SHALL contain `shb_userappl` = `dashcap v1.2.0` and `shb_comment` = `host=node42 interface=lo`
