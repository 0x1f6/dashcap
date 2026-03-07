## 1. Config Model

- [x] 1.1 Add `Exclusions []Exclusion` field to `config.Config` and define `Exclusion` struct with `Name` and `Filter` string fields
- [x] 1.2 Add `exclusions` list to YAML `fileConfig` struct in `internal/config/load.go` and map it to `Config.Exclusions` in `LoadFile`
- [x] 1.3 Add validation in `Config.Validate()`: each exclusion must have non-empty `Name` and `Filter`

## 2. BPF Filter Compilation

- [x] 2.1 Add a `BuildBPFFilter` function in `internal/config/` that takes `[]Exclusion` and returns the combined `not (<expr>) and not (<expr>)` BPF string (empty string if no exclusions)
- [x] 2.2 Add a `ValidateExclusions` function that validates each exclusion's BPF syntax individually using `pcap.CompileBPFFilter`, returning an error that identifies the invalid exclusion by name

## 3. CLI Flag

- [x] 3.1 Add `--exclude` string flag to the root command in `main.go`
- [x] 3.2 In `run()`, if `--exclude` is set, append an `Exclusion{Name: "cli", Filter: value}` to `cfg.Exclusions`

## 4. Filter Application at Startup

- [x] 4.1 In `run()`, after `capture.OpenLive`, call `ValidateExclusions` and then `BuildBPFFilter`; apply the result via `src.SetBPFFilter` if non-empty
- [x] 4.2 Log each configured exclusion (name + filter) at INFO level, and log the final combined BPF expression
- [x] 4.3 Store the active BPF filter string in `config.Config` (new `ActiveBPFFilter` field) for the API to read

## 5. Status API

- [x] 5.1 Add `bpf_filter` field to the `/api/v1/status` response in `handleStatus`, reading from `cfg.ActiveBPFFilter`

## 6. Example Config & Documentation

- [x] 6.1 Add an `exclusions` section to `configs/dashcap.example.yaml` with commented examples matching DESIGN.md section 7.1
- [x] 6.2 Add `--exclude` flag to the CLI reference table in `README.md`

## 7. Tests

- [x] 7.1 Unit test `BuildBPFFilter`: zero exclusions, one exclusion, multiple exclusions
- [x] 7.2 Unit test `ValidateExclusions`: valid filters pass, invalid filter returns error with exclusion name
- [x] 7.3 Unit test config validation: empty name rejected, empty filter rejected
- [x] 7.4 Unit test YAML loading: exclusions parsed correctly from config file
- [x] 7.5 Integration test: `/api/v1/status` response includes `bpf_filter` field
