# Go DevContainer Template

Production-ready DevContainer setup for Go projects with modern tools.

## Quick Start

### 1. Use Template

**In Forgejo:**
- "New Repository" > "From Template" > Select this template
- Clone the repository

**Or clone directly:**
```bash
git clone <template-url> my-project
cd my-project
```

### 2. Open in VSCode

```bash
code .
```

VSCode shows: "Reopen in Container" > Click!

### 3. Done!

The container starts and automatically sets up:
- Go 1.25 with CGO support (libpcap)
- Dev Tools: golangci-lint, staticcheck, delve, gopls
- AI Tools: opencode, openspec, Claude Code
- `.gitignore`, `.golangci.yml`, `Makefile`

## Features

### Development Tools
- **golangci-lint**: Comprehensive linter (configurable via `.golangci.yml`)
- **staticcheck**: Additional static analysis
- **delve**: Go debugger with VSCode integration
- **gopls**: Language server
- **opencode** & **openspec**: AI-assisted development

### VSCode Integration
- Go extension with format on save
- Auto-import organization
- golangci-lint integration
- Debugger configuration
- Makefile tools

### Template Files
Automatically copied on first start (if not already present):
- `.gitignore` — Go-specific ignore rules
- `.golangci.yml` — Linter configuration
- `Makefile` — Build, test, lint, cross-compile targets
- `.forgejo/workflows/ci.yml` — Forgejo Actions CI pipeline

## Changing the Go Version

Edit the base image tag in `.devcontainer/Dockerfile`:
```dockerfile
FROM mcr.microsoft.com/devcontainers/go:1.25-trixie
```
Then rebuild the container: "Dev Containers: Rebuild Container"

## Makefile Targets

```bash
make help    # Show all available targets
make build   # Build binary
make test    # Run tests
make cover   # Run tests with coverage
make lint    # Run golangci-lint
make fmt     # Format code
make tidy    # go mod tidy + verify
make cross   # Cross-compile (Linux, Windows, macOS)
make clean   # Clean build artifacts
```

## Customization

### Installing System Packages
```dockerfile
# .devcontainer/Dockerfile
RUN apt-get update && apt-get install -y \
    libpcap-dev \
    protobuf-compiler
```

### CGO
CGO is enabled by default (`CGO_ENABLED=1`). For pure Go builds:
```json
// devcontainer.json > remoteEnv
"CGO_ENABLED": "0"
```

### Private Go Modules
```json
// devcontainer.json > remoteEnv
"GOPRIVATE": "git.example.com/*"
```

## Structure

```
.
└── .devcontainer/
    ├── devcontainer.json               # DevContainer configuration
    ├── Dockerfile                      # Custom Go dev image
    ├── setup.sh                        # Auto-setup script
    ├── README.md                       # This file
    ├── template.gitignore              # > .gitignore
    ├── template.golangci.yml           # > .golangci.yml
    ├── template.Makefile               # > Makefile
    └── template.ci.yaml               # > .forgejo/workflows/ci.yml
```

## License

Freely usable as a template.
