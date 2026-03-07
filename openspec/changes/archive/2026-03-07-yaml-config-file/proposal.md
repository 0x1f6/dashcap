## Why

dashcap is currently configured exclusively via CLI flags. This works for simple invocations but becomes unwieldy for production deployments with many options (interface, buffer sizes, safety thresholds, API settings, TLS, capture settings). A YAML configuration file allows operators to define a persistent, version-controllable configuration that is easier to manage, audit, and deploy across hosts — especially when running dashcap as a systemd service or Windows service. This is the first item in the Phase 2 roadmap.

## What Changes

- Add a `--config` flag that accepts a path to a YAML configuration file
- Parse the YAML file and map it to the existing `config.Config` struct
- CLI flags take precedence over config file values (config file provides base, flags override)
- The config file follows the structure defined in `configs/dashcap.example.yaml`
- Platform-specific default config paths are searched automatically if `--config` is not specified:
  - Linux/macOS: `/etc/dashcap/dashcap.yaml`
  - Windows: `C:\ProgramData\dashcap\dashcap.yaml`
- Config file is optional — dashcap continues to work with CLI flags only (no file required)
- Add `gopkg.in/yaml.v3` dependency for YAML parsing
- Update the example config file to reflect the supported fields accurately

## Capabilities

### New Capabilities
- `yaml-config`: YAML configuration file loading with platform-specific default paths, CLI flag precedence, and validation

### Modified Capabilities

(none — this adds a new config source but does not change any existing API or capture behavior)

## Impact

- **Code**: `internal/config/` gets a new `load.go` (or similar) for YAML parsing and merge logic; `cmd/dashcap/main.go` gains the `--config` flag and calls the loader before flag override
- **Dependencies**: New dependency `gopkg.in/yaml.v3`
- **Config example**: `configs/dashcap.example.yaml` updated to match supported fields
- **Docs**: README.md and DESIGN.md references to "CLI flags only" should note config file support
