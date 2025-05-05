# Builds for Mac, Linux, and Windows (both amd64 and arm64)

APP_NAME=go-check-ssl-certs
VERSION=0.1.0
BUILD_DIR=./build
MAIN_FILE=./main.go
LDFLAGS=-ldflags "-s -w -X main.toolVersion=${VERSION}"

# List of OSes and architectures to build for
OS_LIST=darwin linux windows
ARCH_LIST=amd64 arm64

# Go build command
GO=go

.PHONY: all clean build-all build-darwin build-linux build-windows help

# Default target
all: clean build-all

# Help
help:
	@echo "Usage:"
	@echo "  make              - Clean and build for all platforms"
	@echo "  make build-all    - Build for all platforms"
	@echo "  make build-darwin - Build for macOS (darwin) only"
	@echo "  make build-linux  - Build for Linux only"
	@echo "  make build-windows - Build for Windows only"
	@echo "  make clean        - Remove build directory"
	@echo "  make help         - Show this help message"

# Clean build directory
clean:
	@echo "Cleaning build directory..."
	@rm -rf $(BUILD_DIR)

# Create build directory
prepare:
	@mkdir -p $(BUILD_DIR)

# Build for all platforms
build-all: prepare
	@echo "Building for all platforms..."
	@$(MAKE) build-darwin
	@$(MAKE) build-linux
	@$(MAKE) build-windows
	@echo "All builds completed successfully in $(BUILD_DIR)/"

# Build for macOS (darwin)
build-darwin: prepare
	@echo "Building for macOS (darwin)..."
	@for arch in $(ARCH_LIST); do \
		echo "Building for darwin/$$arch..."; \
		mkdir -p $(BUILD_DIR)/darwin-$$arch; \
		GOOS=darwin GOARCH=$$arch $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/darwin-$$arch/$(APP_NAME) $(MAIN_FILE); \
		zip -j $(BUILD_DIR)/$(APP_NAME)-darwin-$$arch.zip $(BUILD_DIR)/darwin-$$arch/$(APP_NAME); \
	done

# Build for Linux
build-linux: prepare
	@echo "Building for Linux..."
	@for arch in $(ARCH_LIST); do \
		echo "Building for linux/$$arch..."; \
		mkdir -p $(BUILD_DIR)/linux-$$arch; \
		GOOS=linux GOARCH=$$arch $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/linux-$$arch/$(APP_NAME) $(MAIN_FILE); \
		tar -czf $(BUILD_DIR)/$(APP_NAME)-linux-$$arch.tar.gz -C $(BUILD_DIR)/linux-$$arch $(APP_NAME); \
	done

# Build for Windows
build-windows: prepare
	@echo "Building for Windows..."
	@for arch in $(ARCH_LIST); do \
		echo "Building for windows/$$arch..."; \
		mkdir -p $(BUILD_DIR)/windows-$$arch; \
		GOOS=windows GOARCH=$$arch $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/windows-$$arch/$(APP_NAME).exe $(MAIN_FILE); \
		zip -j $(BUILD_DIR)/$(APP_NAME)-windows-$$arch.zip $(BUILD_DIR)/windows-$$arch/$(APP_NAME).exe; \
	done

# Release target - creates builds and checksums
release: build-all
	@echo "Creating release checksums..."
	@cd $(BUILD_DIR) && shasum -a 256 *.zip *.tar.gz > SHA256SUMS.txt
	@echo "Release artifacts ready in $(BUILD_DIR)/"