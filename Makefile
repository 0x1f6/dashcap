# Project variables
BINARY_NAME := $(shell basename $(CURDIR))
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)"

# Go parameters
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

.PHONY: all build test lint fmt clean help

## all: Build and test
all: lint test build

## build: Build the binary
build:
	CGO_ENABLED=1 go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/$(BINARY_NAME)

## test: Run tests
test:
	go test -v -race -count=1 ./...

## cover: Run tests with coverage
cover:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## lint: Run golangci-lint
lint:
	golangci-lint run

## fmt: Format code
fmt:
	gofmt -s -w .
	goimports -w .

## tidy: Tidy and verify dependencies
tidy:
	go mod tidy
	go mod verify

## clean: Remove build artifacts
clean:
	rm -rf bin/ dist/ coverage.out coverage.html

## cross: Cross-compile for Linux and Windows (requires platform-specific libpcap/Npcap at build time)
cross:
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 ./cmd/$(BINARY_NAME)
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe ./cmd/$(BINARY_NAME)
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 ./cmd/$(BINARY_NAME)

## help: Show this help
help:
	@echo "Available targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
