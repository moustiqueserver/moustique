# Moustique Makefile
# Build server and CLI for multiple platforms

# Version info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Directories
BUILD_DIR := build
DIST_DIR := dist

# Targets
.PHONY: all clean server cli install help \
	server-linux server-darwin server-windows \
	cli-linux cli-darwin cli-windows cli-arm \
	dist-all test

# Default target
all: server cli

help:
	@echo "Moustique Build System"
	@echo ""
	@echo "Targets:"
	@echo "  make all           - Build server and CLI for current platform"
	@echo "  make server        - Build server for current platform"
	@echo "  make cli           - Build CLI for current platform"
	@echo "  make install       - Install server and CLI to /usr/local/bin"
	@echo "  make dist-all      - Build all binaries for all platforms"
	@echo "  make clean         - Remove build artifacts"
	@echo "  make test          - Run Go unit tests"
	@echo "  make test-clients  - Run client integration tests (requires running server)"
	@echo ""
	@echo "Platform-specific builds:"
	@echo "  make server-linux  - Build server for Linux (amd64, arm64)"
	@echo "  make server-darwin - Build server for macOS (amd64, arm64)"
	@echo "  make cli-linux     - Build CLI for Linux (amd64, arm64, arm)"
	@echo "  make cli-darwin    - Build CLI for macOS (amd64, arm64)"
	@echo "  make cli-windows   - Build CLI for Windows (amd64)"

# Build server for current platform
server:
	@echo "Building Moustique server..."
	go build $(LDFLAGS) -o moustique .

# Build CLI for current platform
cli:
	@echo "Building Moustique CLI..."
	go build $(LDFLAGS) -o moustique-cli ./cmd/moustique-cli

# Install to /usr/local/bin
install: server cli
	@echo "Installing to /usr/local/bin..."
	sudo cp moustique /usr/local/bin/
	sudo cp moustique-cli /usr/local/bin/
	@echo "Installed successfully!"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f moustique moustique-cli
	rm -rf $(BUILD_DIR) $(DIST_DIR)
	@echo "Clean complete!"

# Run Go unit tests
test:
	@echo "Running Go unit tests..."
	go test -v ./...

# Run client integration tests
test-clients:
	@echo "Running client integration tests..."
	@./tests/test_all_clients.sh

# ============================================================================
# Cross-compilation targets
# ============================================================================

# Create build directories
$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

$(DIST_DIR):
	mkdir -p $(DIST_DIR)

# Server builds
server-linux: $(BUILD_DIR)
	@echo "Building server for Linux AMD64..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/moustique-linux-amd64 .
	@echo "Building server for Linux ARM64..."
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/moustique-linux-arm64 .

server-darwin: $(BUILD_DIR)
	@echo "Building server for macOS AMD64..."
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/moustique-darwin-amd64 .
	@echo "Building server for macOS ARM64 (M1/M2)..."
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/moustique-darwin-arm64 .

server-windows: $(BUILD_DIR)
	@echo "Building server for Windows AMD64..."
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/moustique-windows-amd64.exe .

# CLI builds
cli-linux: $(BUILD_DIR)
	@echo "Building CLI for Linux AMD64..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/moustique-cli-linux-amd64 ./cmd/moustique-cli
	@echo "Building CLI for Linux ARM64..."
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/moustique-cli-linux-arm64 ./cmd/moustique-cli
	@echo "Building CLI for Linux ARM (32-bit)..."
	GOOS=linux GOARCH=arm GOARM=7 go build $(LDFLAGS) -o $(BUILD_DIR)/moustique-cli-linux-arm ./cmd/moustique-cli

cli-darwin: $(BUILD_DIR)
	@echo "Building CLI for macOS AMD64..."
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/moustique-cli-darwin-amd64 ./cmd/moustique-cli
	@echo "Building CLI for macOS ARM64 (M1/M2)..."
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/moustique-cli-darwin-arm64 ./cmd/moustique-cli

cli-windows: $(BUILD_DIR)
	@echo "Building CLI for Windows AMD64..."
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/moustique-cli-windows-amd64.exe ./cmd/moustique-cli

cli-arm: $(BUILD_DIR)
	@echo "Building CLI for Raspberry Pi (ARM)..."
	GOOS=linux GOARCH=arm GOARM=7 go build $(LDFLAGS) -o $(BUILD_DIR)/moustique-cli-rpi ./cmd/moustique-cli

# Build everything and create distribution archives
dist-all: $(DIST_DIR) server-linux server-darwin server-windows cli-linux cli-darwin cli-windows
	@echo "Creating distribution archives..."
	cd $(BUILD_DIR) && tar -czf ../$(DIST_DIR)/moustique-server-linux-amd64.tar.gz moustique-linux-amd64
	cd $(BUILD_DIR) && tar -czf ../$(DIST_DIR)/moustique-server-linux-arm64.tar.gz moustique-linux-arm64
	cd $(BUILD_DIR) && tar -czf ../$(DIST_DIR)/moustique-server-darwin-amd64.tar.gz moustique-darwin-amd64
	cd $(BUILD_DIR) && tar -czf ../$(DIST_DIR)/moustique-server-darwin-arm64.tar.gz moustique-darwin-arm64
	cd $(BUILD_DIR) && zip ../$(DIST_DIR)/moustique-server-windows-amd64.zip moustique-windows-amd64.exe
	cd $(BUILD_DIR) && tar -czf ../$(DIST_DIR)/moustique-cli-linux-amd64.tar.gz moustique-cli-linux-amd64
	cd $(BUILD_DIR) && tar -czf ../$(DIST_DIR)/moustique-cli-linux-arm64.tar.gz moustique-cli-linux-arm64
	cd $(BUILD_DIR) && tar -czf ../$(DIST_DIR)/moustique-cli-linux-arm.tar.gz moustique-cli-linux-arm
	cd $(BUILD_DIR) && tar -czf ../$(DIST_DIR)/moustique-cli-darwin-amd64.tar.gz moustique-cli-darwin-amd64
	cd $(BUILD_DIR) && tar -czf ../$(DIST_DIR)/moustique-cli-darwin-arm64.tar.gz moustique-cli-darwin-arm64
	cd $(BUILD_DIR) && zip ../$(DIST_DIR)/moustique-cli-windows-amd64.zip moustique-cli-windows-amd64.exe
	@echo "Distribution archives created in $(DIST_DIR)/"
	@echo ""
	@echo "Built binaries:"
	@ls -lh $(BUILD_DIR)/
	@echo ""
	@echo "Distribution archives:"
	@ls -lh $(DIST_DIR)/
