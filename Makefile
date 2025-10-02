# Makefile for RN-LB

# --- Build variables ---
VERSION     ?= $(shell git describe --tags --always --dirty)
COMMIT      ?= $(shell git rev-parse HEAD)
BUILD_TIME  ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS     := -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)

GOFLAGS     := -trimpath -ldflags "$(LDFLAGS)"
ISOLATION_TAG := isolation

# --- Directories ---
BIN_DIR     := bin
DIST_DIR    := dist
DAEMON_PKG  := ./cmd/daemon
PROXY_PKG   := ./cmd/proxy

# --- Binaries ---
MONOLITHIC_BIN := $(BIN_DIR)/rn-lb
MAIN_BIN       := $(BIN_DIR)/rn-lb-main
PROXY_BIN      := $(BIN_DIR)/rn-lb-proxy

# --- Cross compilation platforms ---
PLATFORMS := \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64

# --- Targets ---
.PHONY: all help build-monolithic build-split build-main build-proxy \
        test test-unit test-race test-integration \
        clean install install-systemd release generate-secret \
        run-monolithic run-split lint fmt deps vet

## Default build (monolithic + split)
all: build-monolithic build-split

## Show available targets
help:
	@echo "Available make targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

## Build monolithic daemon
build-monolithic: 
	@echo "Building monolithic binary..."
	@mkdir -p $(BIN_DIR)
	go build $(GOFLAGS) -o $(MONOLITHIC_BIN) $(DAEMON_PKG)

## Build split binaries
build-split: build-main build-proxy

## Build main daemon with isolation layer
build-main:
	@echo "Building main daemon..."
	@mkdir -p $(BIN_DIR)
	go build $(GOFLAGS) -tags $(ISOLATION_TAG) -o $(MAIN_BIN) $(DAEMON_PKG)

## Build proxy
build-proxy:
	@echo "Building proxy..."
	@mkdir -p $(BIN_DIR)
	go build $(GOFLAGS) -o $(PROXY_BIN) $(PROXY_PKG)

## Run all tests
test: test-unit test-race

## Unit tests
test-unit:
	go test ./...

## Race detector tests
test-race:
	go test -race ./...

## Integration tests
test-integration:
	go test -tags=integration ./tests/integration/...

## Clean build artifacts
clean:
	rm -rf $(BIN_DIR) $(DIST_DIR)

## Install binaries to system
install: build-monolithic build-split
	install -m 755 $(MONOLITHIC_BIN) /usr/local/bin/
	install -m 755 $(MAIN_BIN) /usr/local/bin/
	install -m 755 $(PROXY_BIN) /usr/local/bin/
	install -d /etc/rn-lb /var/lib/rn-lb /var/log/rn-lb

## Install systemd units
install-systemd:
	install -m 644 systemd/rn-lb.service /etc/systemd/system/
	install -m 644 systemd/rn-lb-proxy.service /etc/systemd/system/
	systemctl daemon-reload

## Build release binaries for multiple platforms
release: clean
	@echo "Building release packages..."
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/}; \
		echo "  > $$GOOS/$$GOARCH"; \
		go build $(GOFLAGS) -o $(DIST_DIR)/rn-lb-$$GOOS-$$GOARCH $(DAEMON_PKG); \
		go build $(GOFLAGS) -tags $(ISOLATION_TAG) -o $(DIST_DIR)/rn-lb-main-$$GOOS-$$GOARCH $(DAEMON_PKG); \
		go build $(GOFLAGS) -o $(DIST_DIR)/rn-lb-proxy-$$GOOS-$$GOARCH $(PROXY_PKG); \
	done
	@echo "Release packages are in $(DIST_DIR)/"

## Generate JWT secret
generate-secret:
	@openssl rand -base64 32 > jwt-secret.key
	@echo "Secret saved to jwt-secret.key"

## Run monolithic daemon
run-monolithic: build-monolithic
	./$(MONOLITHIC_BIN) -config config.yaml

## Run split daemon and proxy
run-split: build-split
	./$(PROXY_BIN) -config proxy-config.yaml &
	@sleep 2
	./$(MAIN_BIN) -config config.yaml

## Lint code
lint:
	golangci-lint run ./...

## Format code
fmt:
	go fmt ./...
	goimports -w .

## Vet code (extra checks)
vet:
	go vet ./...

## Update dependencies
deps:
	go mod tidy
	go mod verify
