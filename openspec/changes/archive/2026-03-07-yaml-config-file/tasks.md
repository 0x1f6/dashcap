## 1. Dependencies & Shared Utilities

- [x] 1.1 Add `gopkg.in/yaml.v3` dependency (`go get gopkg.in/yaml.v3`)
- [x] 1.2 Move `parseSize` from `cmd/dashcap/main.go` to `internal/config/size.go` so both CLI and YAML parsing share it; update `main.go` to call `config.ParseSize`

## 2. YAML Loading

- [x] 2.1 Create `internal/config/load.go` with intermediate `fileConfig` struct (YAML tags matching the nested structure from `configs/dashcap.example.yaml`), `LoadFile(path string) (*Config, error)` function that unmarshals YAML, uses `yaml.v3` strict mode (disallow unknown keys), parses human-readable sizes and durations, and maps to `Config`
- [x] 2.2 Add platform-specific default config path function `DefaultConfigPath() string` in `internal/config/load.go` (Linux/macOS: `/etc/dashcap/dashcap.yaml`, Windows: `C:\ProgramData\dashcap\dashcap.yaml`)
- [x] 2.3 Add `ResolveConfigFile(explicit string) (string, error)` that returns the explicit path (error if not found), or the default path if it exists, or empty string if no file found

## 3. CLI Integration

- [x] 3.1 Add `--config` flag to the root command in `cmd/dashcap/main.go`
- [x] 3.2 In the `RunE` function: call `ResolveConfigFile`, if a file is found call `LoadFile` to get base config, then apply only explicitly-changed CLI flags (using `cmd.Flags().Changed()`) on top; log the config file path at info level when loaded

## 4. Config Example Update

- [x] 4.1 Update `configs/dashcap.example.yaml` to include `api.token`, `api.no_auth`, `api.tls_cert`, `api.tls_key` fields with comments, ensuring all supported config fields are represented

## 5. Tests

- [x] 5.1 Unit tests for `LoadFile`: valid YAML, unknown keys (strict mode error), invalid YAML syntax, size parsing (`50MB`, `2GB`, plain bytes), duration parsing (`5m`, `30s`)
- [x] 5.2 Unit tests for `ResolveConfigFile`: explicit path exists, explicit path missing (error), no explicit path with no default (returns empty)
- [x] 5.3 Integration test for CLI flag precedence: config file sets a value, CLI flag overrides it; config file sets a value, no CLI flag leaves it; CLI flag at Cobra default does not override config file value
- [x] 5.4 Unit test for `ParseSize` (moved function) — ensure existing size parsing tests still pass or are ported

## 6. Documentation

- [x] 6.1 Update README.md: add `--config` flag to the Server Flags table, add a brief section on config file support under Configuration, note CLI flag precedence
