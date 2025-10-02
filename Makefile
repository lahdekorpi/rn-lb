# Makefile for RN-LB

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT ?= $(shell git rev-parse HEAD)
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)

# Go build flags
GOFLAGS := -trimpath -ldflags "$(LDFLAGS)"
ISOLATION_TAG := isolation

# Output directories
BIN_DIR := bin
DIST_DIR := dist

# Binaries
MONOLITHIC_BIN := $(BIN_DIR)/rn-lb
MAIN_BIN := $(BIN_DIR)/rn-lb-main
PROXY_BIN := $(BIN_DIR)/rn-lb-proxy

.PHONY: all build-monolithic build-split build-main build-proxy clean test install

# Default target
all: build-monolithic build-split

# Build monolithic binary (no isolation layer)
build-monolithic:
	@echo "Building monolithic binary (no isolation)..."
	@mkdir -p $(BIN_DIR)
	go build $(GOFLAGS) -o $(MONOLITHIC_BIN) ./cmd/daemon

# Build split binaries (with isolation layer)
build-split: build-main build-proxy

# Build main daemon with isolation layer support
build-main:
	@echo "Building main daemon (with isolation layer)..."
	@mkdir -p $(BIN_DIR)
	go build $(GOFLAGS) -tags $(ISOLATION_TAG) -o $(MAIN_BIN) ./cmd/daemon

# Build API proxy
build-proxy:
	@echo "Building API proxy..."
	@mkdir -p $(BIN_DIR)
	go build $(GOFLAGS) -o $(PROXY_BIN) ./cmd/proxy

# Run tests
test:
	go test -v -race -cover ./...

# Run integration tests
test-integration:
	go test -v -tags=integration ./tests/integration/...

# Clean build artifacts
clean:
	rm -rf $(BIN_DIR) $(DIST_DIR)

# Install binaries to system (requires root)
install: build-monolithic build-split
	install -m 755 $(MONOLITHIC_BIN) /usr/local/bin/
	install -m 755 $(MAIN_BIN) /usr/local/bin/
	install -m 755 $(PROXY_BIN) /usr/local/bin/
	install -d /etc/rn-lb
	install -d /var/lib/rn-lb
	install -d /var/log/rn-lb

# Install systemd service files
install-systemd:
	install -m 644 systemd/rn-lb.service /etc/systemd/system/
	install -m 644 systemd/rn-lb-proxy.service /etc/systemd/system/
	systemctl daemon-reload

# Build release packages for multiple platforms
release: clean
	@echo "Building release packages..."
	@mkdir -p $(DIST_DIR)
	# Linux AMD64
	GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -o $(DIST_DIR)/rn-lb-linux-amd64 ./cmd/daemon
	GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -tags $(ISOLATION_TAG) -o $(DIST_DIR)/rn-lb-main-linux-amd64 ./cmd/daemon
	GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -o $(DIST_DIR)/rn-lb-proxy-linux-amd64 ./cmd/proxy
	# Linux ARM64
	GOOS=linux GOARCH=arm64 go build $(GOFLAGS) -o $(DIST_DIR)/rn-lb-linux-arm64 ./cmd/daemon
	GOOS=linux GOARCH=arm64 go build $(GOFLAGS) -tags $(ISOLATION_TAG) -o $(DIST_DIR)/rn-lb-main-linux-arm64 ./cmd/daemon
	GOOS=linux GOARCH=arm64 go build $(GOFLAGS) -o $(DIST_DIR)/rn-lb-proxy-linux-arm64 ./cmd/proxy
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 go build $(GOFLAGS) -o $(DIST_DIR)/rn-lb-darwin-amd64 ./cmd/daemon
	GOOS=darwin GOARCH=amd64 go build $(GOFLAGS) -tags $(ISOLATION_TAG) -o $(DIST_DIR)/rn-lb-main-darwin-amd64 ./cmd/daemon
	GOOS=darwin GOARCH=amd64 go build $(GOFLAGS) -o $(DIST_DIR)/rn-lb-proxy-darwin-amd64 ./cmd/proxy
	# macOS ARM64
	GOOS=darwin GOARCH=arm64 go build $(GOFLAGS) -o $(DIST_DIR)/rn-lb-darwin-arm64 ./cmd/daemon
	GOOS=darwin GOARCH=arm64 go build $(GOFLAGS) -tags $(ISOLATION_TAG) -o $(DIST_DIR)/rn-lb-main-darwin-arm64 ./cmd/daemon
	GOOS=darwin GOARCH=arm64 go build $(GOFLAGS) -o $(DIST_DIR)/rn-lb-proxy-darwin-arm64 ./cmd/proxy
	@echo "Release packages built in $(DIST_DIR)/"

# Generate JWT secret for isolation layer
generate-secret:
	@echo "Generating JWT secret..."
	@openssl rand -base64 32 > jwt-secret.key
	@echo "Secret saved to jwt-secret.key"

# Development: run monolithic daemon
run-monolithic: build-monolithic
	./$(MONOLITHIC_BIN) -config config.yaml

# Development: run split daemon and proxy
run-split:
	@echo "Starting proxy in background..."
	./$(PROXY_BIN) -config proxy-config.yaml &
	@sleep 2
	@echo "Starting main daemon..."
	./$(MAIN_BIN) -config config.yaml

# Lint code
lint:
	golangci-lint run ./...

# Format code
fmt:
	go fmt ./...
	goimports -w .

# Update dependencies
deps:
	go mod tidy
	go mod verify