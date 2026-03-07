## ADDED Requirements

### Requirement: CI workflow triggers on push and PR
The CI workflow SHALL trigger on pushes to `main` and on pull requests targeting `main`.

#### Scenario: Push to main triggers CI
- **WHEN** a commit is pushed to the `main` branch
- **THEN** the CI workflow SHALL execute

#### Scenario: PR to main triggers CI
- **WHEN** a pull request targeting `main` is opened or updated
- **THEN** the CI workflow SHALL execute

#### Scenario: Push to feature branch without PR does not trigger CI
- **WHEN** a commit is pushed to a non-main branch without an open PR
- **THEN** the CI workflow SHALL NOT execute

### Requirement: CI runs golangci-lint
The CI workflow SHALL run `golangci-lint` using the `golangci/golangci-lint-action`.

#### Scenario: Lint passes
- **WHEN** all code passes golangci-lint checks
- **THEN** the lint step SHALL succeed

#### Scenario: Lint fails
- **WHEN** code contains lint violations
- **THEN** the lint step SHALL fail and the workflow SHALL report failure

### Requirement: CI runs tests with race detector and coverage
The CI workflow SHALL run `make cover` which executes tests with `-race` and `-coverprofile`.

#### Scenario: Tests pass with coverage
- **WHEN** all tests pass
- **THEN** the test step SHALL succeed and produce a coverage report

#### Scenario: Tests fail
- **WHEN** any test fails
- **THEN** the workflow SHALL report failure

### Requirement: CI verifies the build
The CI workflow SHALL run `make build` to verify the binary compiles successfully.

#### Scenario: Build succeeds
- **WHEN** the code compiles without errors
- **THEN** the build step SHALL succeed

### Requirement: CI installs libpcap-dev
The CI workflow SHALL install `libpcap-dev` before building, since CGO_ENABLED=1 requires libpcap headers.

#### Scenario: libpcap-dev is available for build
- **WHEN** the CI environment is set up
- **THEN** `libpcap-dev` SHALL be installed via apt before any Go build or test steps
