## Context

dashcap currently uses CLI flags exclusively for configuration, parsed via Cobra in `cmd/dashcap/main.go` and stored in `internal/config/config.go`. The `Config` struct holds all runtime settings with defaults provided by `Defaults()`. An example YAML file exists at `configs/dashcap.example.yaml` but is not consumed by the application.

For production deployments (systemd units, Windows services), operators need a persistent config file rather than embedding all options in command-line arguments. The YAML structure is already defined in the example and the DESIGN.md.

## Goals / Non-Goals

**Goals:**
- Load configuration from a YAML file and merge it into the existing `Config` struct
- CLI flags always override config file values
- Auto-discover platform-specific default config paths when `--config` is not specified
- Keep the config file optional — dashcap works identically without one
- Validate the merged config the same way as today (`Config.Validate()`)

**Non-Goals:**
- Hot-reloading of config file changes at runtime
- Config file generation or migration tooling
- Environment variable mapping beyond the existing `DASHCAP_API_TOKEN`
- TOML, JSON, or other config file formats

## Decisions

### 1. YAML parser: `gopkg.in/yaml.v3`

Use `gopkg.in/yaml.v3` directly rather than Viper or similar frameworks.

**Rationale:** dashcap has a single, flat config struct. Viper adds transitive dependencies and complexity (env binding, remote config, watchers) that we don't need. `yaml.v3` is a single dependency with no transitives, well-tested, and gives us full control over the merge logic.

**Alternative considered:** `github.com/spf13/viper` — rejected due to heavy dependency tree and magic behavior around key casing and precedence that can be hard to debug.

### 2. Merge strategy: YAML provides base, CLI flags override

The loading order is:
1. `config.Defaults()` — hardcoded defaults
2. YAML file values override defaults (if a file is found)
3. CLI flags override everything (only flags explicitly set by the user)

To detect which CLI flags were explicitly set, we use Cobra's `cmd.Flags().Changed("flag-name")`. Only changed flags override the config file.

**Rationale:** This is the standard convention (kubectl, docker, etc.) and matches what users expect. The tricky part is distinguishing "user passed `--api-port 9800`" from "flag has default value 9800" — `Changed()` solves this.

### 3. Config file structure: flat YAML with nested groups

The YAML structure mirrors the example in `configs/dashcap.example.yaml`:

```yaml
interface: eth0
buffer:
  size: 2GB
  segment_size: 100MB
trigger:
  default_duration: 5m
safety:
  min_free_after_alloc: 1GB
  min_free_percent: 5
api:
  tcp_port: 9800
  token: ""
  no_auth: false
  tls_cert: ""
  tls_key: ""
capture:
  snaplen: 0
  promiscuous: true
storage:
  data_dir: /var/lib/dashcap/eth0
logging:
  level: info
```

An intermediate `fileConfig` struct with YAML tags handles parsing. After unmarshalling, values are mapped onto the existing `Config` struct. Human-readable sizes (`2GB`, `100MB`) are parsed using the existing `parseSize` logic.

**Rationale:** Keeps the YAML user-friendly (human-readable sizes, grouped keys) while the runtime `Config` struct stays flat and simple.

### 4. Config file discovery: explicit flag > platform defaults > none

- `--config <path>`: use exactly this file; error if it does not exist
- No `--config`: check platform default paths in order, use the first that exists, silently skip if none found
  - Linux/macOS: `/etc/dashcap/dashcap.yaml`
  - Windows: `C:\ProgramData\dashcap\dashcap.yaml`

**Rationale:** Mirrors conventions from tools like Docker, systemd, and sshd. Silent skip on missing default avoids breaking existing CLI-only workflows.

### 5. Implementation location

- New file `internal/config/load.go`: contains `LoadFile(path string) (*Config, error)` and the intermediate YAML struct
- `cmd/dashcap/main.go`: adds `--config` flag, calls `LoadFile` before applying flag overrides
- The `parseSize` function is moved from `main.go` to `internal/config/` so both CLI and YAML parsing share the same logic

## Risks / Trade-offs

- **[Risk] Size parsing inconsistency** — YAML and CLI could parse sizes differently → Mitigation: share the same `parseSize` function for both paths; add unit tests covering both.
- **[Risk] Precedence confusion** — User may not understand why a config file value is overridden → Mitigation: log the config source at startup (e.g., `config loaded from /etc/dashcap/dashcap.yaml`); log when CLI flags override file values at debug level.
- **[Trade-off] No hot-reload** — Config changes require a restart → Acceptable for Phase 2; hot-reload can be added later if needed.
- **[Trade-off] New dependency** — `gopkg.in/yaml.v3` adds a dependency → Minimal risk; it's a well-maintained, zero-transitive-dependency package.
