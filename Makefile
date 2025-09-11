# ──────────────────────────────────────────────────────────────────────────────
# ERC-2470 Address Miner — Makefile
# ──────────────────────────────────────────────────────────────────────────────

.DEFAULT_GOAL := help
.SHELLFLAGS   := -c
.PHONY: build build-all clean test install deps help

# ── Project
BINARY_NAME := erc2470-miner
PKG        := ./cmd/erc2470-miner

# VERSION via git; falls back to "dev"
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

# ── Platform specifics
BIN_DIR     := bin
MKDIR       := mkdir -p $(BIN_DIR)
RMDIR       := rm -rf $(BIN_DIR)
BUILD_TIME  := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Linker flags (version + build time)
LDFLAGS := -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -s -w

# Optional: allow `make CGO=0 build` to disable CGO
ifdef CGO
  CGO_ENV := CGO_ENABLED=$(CGO)
else
  CGO_ENV :=
endif

# ── Targets ───────────────────────────────────────────────────────────────────

## build: Build native binary for the host OS/arch
build:
	@echo Building $(BINARY_NAME) v$(VERSION)...
	@$(MKDIR)
	$(CGO_ENV) go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) $(PKG)
	@echo ✔ Output: $(BIN_DIR)/$(BINARY_NAME)

## build-all: Cross-compile for Linux (amd64), macOS (amd64/arm64), and Windows (amd64)
build-all:
	@echo Cross-building $(BINARY_NAME) v$(VERSION)...
	@$(MKDIR)
	@echo • linux/amd64
	GOOS=linux   GOARCH=amd64 $(CGO_ENV) go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 $(PKG)
	@echo • darwin/amd64
	GOOS=darwin  GOARCH=amd64 $(CGO_ENV) go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-darwin-amd64 $(PKG)
	@echo • darwin/arm64
	GOOS=darwin  GOARCH=arm64 $(CGO_ENV) go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-darwin-arm64 $(PKG)
	@echo • windows/amd64
	GOOS=windows GOARCH=amd64 $(CGO_ENV) go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-windows-amd64.exe $(PKG)
	@echo ✔ Cross-build complete in $(BIN_DIR)/

## deps: Sync and download Go module dependencies
deps:
	go mod tidy
	go mod download

## test: Run all tests (verbose)
test:
	go test -v ./...

## test-race: Run all tests with race detection
test-race:
	go test -race -v ./...

## clean: Remove build artifacts
clean:
	@echo Cleaning $(BIN_DIR)...
	@$(RMDIR)
	@echo ✔ Clean

## install: Build and install to GOPATH/bin
install: build
	go install

## help: Show available targets
help:
	@echo "Targets:"
	@echo "  build       Build native binary"
	@echo "  build-all   Cross-compile (linux/darwin/windows)"
	@echo "  deps        Sync & download modules"
	@echo "  test        Run tests"
	@echo "  test-race   Run tests with race detection"
	@echo "  clean       Remove build artifacts"
	@echo "  install     Build and install to GOPATH/bin"
	@echo ""
	@echo "Info:"
	@echo "  Version     : $(VERSION)"
	@echo "  Build Time  : $(BUILD_TIME)"
	@echo "  Bin Dir     : $(BIN_DIR)"
	@echo ""
	@echo "Options:"
	@echo "  CGO=0       Disable CGO for deterministic cross-builds (e.g., 'make CGO=0 build-all')"
