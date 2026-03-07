## 1. CI Workflow

- [x] 1.1 Create `.github/workflows/ci.yml` with push/PR triggers on `main`
- [x] 1.2 Add step to install `libpcap-dev` via apt
- [x] 1.3 Add Go setup with `actions/setup-go` using `go-version-file: go.mod`
- [x] 1.4 Add lint step using `golangci/golangci-lint-action`
- [x] 1.5 Add test step running `make cover` (race + coverage)
- [x] 1.6 Add build verification step running `make build`

## 2. Release Workflow

- [x] 2.1 Create `.github/workflows/release.yml` with `v*` tag trigger
- [x] 2.2 Add Linux amd64 build step with LDFLAGS (version, commit, build time)
- [x] 2.3 Add Windows amd64 cross-compilation step with MinGW and WinPcap/Npcap headers
- [x] 2.4 Add SHA256 checksum generation step
- [x] 2.5 Add GitHub Release creation using `softprops/action-gh-release` with all artifacts

## 3. Dependabot

- [x] 3.1 Create `.github/dependabot.yml` for Go modules and GitHub Actions version updates

## 4. Verification

- [x] 4.1 Push to a branch and verify CI workflow runs successfully
- [x] 4.2 Review workflow files for correctness and security (no secrets leaks, pinned action versions)
