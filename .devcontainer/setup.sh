#!/bin/bash
set -e

echo "🚀 Starting Go devcontainer setup..."

# Setup template files if they don't exist
echo "📄 Checking for template files..."

if [ ! -f ".gitignore" ] && [ -f ".devcontainer/template.gitignore" ]; then
    echo "  ✅ Copying .gitignore template"
    cp .devcontainer/template.gitignore .gitignore
fi

if [ ! -f ".golangci.yml" ] && [ -f ".devcontainer/template.golangci.yml" ]; then
    echo "  ✅ Copying .golangci.yml template"
    cp .devcontainer/template.golangci.yml .golangci.yml
fi

if [ ! -f ".forgejo/workflows/ci.yml" ] && [ -f ".devcontainer/template.ci.yaml" ]; then
    echo "  ✅ Copying .forgejo/workflows/ci.yml template"
    mkdir -p .forgejo/workflows
    cp .devcontainer/template.ci.yaml .forgejo/workflows/ci.yml
fi

if [ ! -f "Makefile" ] && [ -f ".devcontainer/template.Makefile" ]; then
    echo "  ✅ Copying Makefile template"
    cp .devcontainer/template.Makefile Makefile
fi

echo ""

# Install Go tools not included in the base image
echo "🔧 Installing Go tools..."
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest 2>/dev/null && echo "  ✅ golangci-lint" || echo "  ⚠️  golangci-lint failed"
go install honnef.co/go/tools/cmd/staticcheck@latest 2>/dev/null && echo "  ✅ staticcheck" || echo "  ⚠️  staticcheck failed"

# Install AI tools
echo ""
echo "🤖 Installing AI tools..."

if command -v npm &>/dev/null; then
    npm i -g opencode-ai 2>/dev/null && echo "  ✅ opencode" || echo "  ⚠️  opencode failed"
    npm install -g @fission-ai/openspec@latest 2>/dev/null && echo "  ✅ openspec" || echo "  ⚠️  openspec failed"
else
    echo "  ⚠️  npm not available, skipping openspec"
fi

echo ""

# Initialize Go module if go.mod doesn't exist
if [ ! -f "go.mod" ]; then
    MODULE_NAME=""
    if git remote get-url origin &>/dev/null; then
        REMOTE_URL=$(git remote get-url origin)
        MODULE_NAME=$(echo "$REMOTE_URL" | sed -E 's|^https?://||;s|^git@([^:]+):|\\1/|;s|\.git$||')
    fi
    if [ -z "$MODULE_NAME" ]; then
        MODULE_NAME=$(basename "$(pwd)")
    fi
    echo "📦 Initializing Go module: ${MODULE_NAME}"
    go mod init "${MODULE_NAME}"
fi

# Download dependencies
if [ -f "go.sum" ]; then
    echo "📚 Downloading dependencies..."
    go mod download
fi

# Verify installations
echo ""
echo "✅ Verifying installations..."
go version
golangci-lint --version 2>/dev/null || echo "⚠️  golangci-lint not found"
staticcheck --version 2>/dev/null || echo "⚠️  staticcheck not found"
dlv version 2>/dev/null || echo "⚠️  delve not found"
opencode --version 2>/dev/null || echo "⚠️  opencode not found"
openspec --version 2>/dev/null || echo "⚠️  openspec not found"

echo ""
echo "✨ Go devcontainer setup complete!"
echo ""
echo "Quick commands:"
echo "  go build ./...           - Build project"
echo "  go test ./...            - Run tests"
echo "  go test -v -cover ./...  - Run tests with coverage"
echo "  golangci-lint run        - Lint code"
echo "  make help                - Show Makefile targets"
