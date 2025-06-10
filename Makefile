# VST3Go Makefile

# Platform detection
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Linux)
    PLATFORM := linux
    SO_EXT := so
    VST3_ARCH_64 := x86_64-linux
    VST3_ARCH_32 := i386-linux
    RPATH_FLAG := -Wl,-rpath,'$$ORIGIN'
endif
ifeq ($(UNAME_S),Darwin)
    PLATFORM := darwin
    SO_EXT := dylib
    VST3_ARCH_64 := MacOS
    VST3_ARCH_32 := MacOS
    RPATH_FLAG := -Wl,-rpath,@loader_path
endif

# Build variables
BUILD_DIR := build
BUILD_DIR_64 := $(BUILD_DIR)/64
BUILD_DIR_32 := $(BUILD_DIR)/32
BRIDGE_DIR := bridge
EXAMPLES_DIR := examples

# Compiler flags - Default to debug mode
CFLAGS_BASE := -fPIC -I./include -g -O0 -DDEBUG_VST3GO
LDFLAGS_BASE := -shared -g

# Architecture-specific flags
CFLAGS_64 := $(CFLAGS_BASE)
LDFLAGS_64 := $(LDFLAGS_BASE)
CFLAGS_32 := $(CFLAGS_BASE) -m32
LDFLAGS_32 := $(LDFLAGS_BASE) -m32

# Debug flags
DEBUG_CFLAGS := -fPIC -I./include -g -O0 -DDEBUG_VST3GO
DEBUG_LDFLAGS := -shared -g

# Default target
all: build

# Build all example plugins (both 32-bit and 64-bit)
build: build-64 build-32

# Build 64-bit plugins
build-64:
	@mkdir -p $(BUILD_DIR_64)
	@for dir in $(EXAMPLES_DIR)/*; do \
		if [ -d "$$dir" ] && [ -f "$$dir/main.go" ]; then \
			example=$$(basename $$dir); \
			echo "Building $$example plugin (64-bit)"; \
			GOARCH=amd64 CGO_CFLAGS="$(CFLAGS_64)" CGO_LDFLAGS="$(LDFLAGS_64)" go build -buildvcs=false -buildmode=c-shared \
				-o $(BUILD_DIR_64)/$$example.$(SO_EXT) \
				./$$dir || exit 1; \
		fi; \
	done
	@echo "All 64-bit plugins built successfully"

# Build 32-bit plugins (Linux only)
build-32:
ifeq ($(PLATFORM),linux)
	@mkdir -p $(BUILD_DIR_32)
	@for dir in $(EXAMPLES_DIR)/*; do \
		if [ -d "$$dir" ] && [ -f "$$dir/main.go" ]; then \
			example=$$(basename $$dir); \
			echo "Building $$example plugin (32-bit)"; \
			GOARCH=386 CGO_CFLAGS="$(CFLAGS_32)" CGO_LDFLAGS="$(LDFLAGS_32)" go build -buildvcs=false -buildmode=c-shared \
				-o $(BUILD_DIR_32)/$$example.$(SO_EXT) \
				./$$dir || exit 1; \
		fi; \
	done
	@echo "All 32-bit plugins built successfully"
else
	@echo "32-bit build not supported on $(PLATFORM)"
endif

# Install all example VST3 plugins to user's VST3 directory
install: build
	@echo "Installing all example VST3 plugins to ~/.vst3"
	@mkdir -p ~/.vst3
	@for dir in $(EXAMPLES_DIR)/*; do \
		if [ -d "$$dir" ] && [ -f "$$dir/main.go" ]; then \
			example=$$(basename $$dir); \
			echo "Creating and installing $$example.vst3 bundle"; \
			rm -rf $(BUILD_DIR)/$$example.vst3; \
			if [ -f "$(BUILD_DIR_64)/$$example.$(SO_EXT)" ]; then \
				mkdir -p $(BUILD_DIR)/$$example.vst3/Contents/$(VST3_ARCH_64); \
				cp $(BUILD_DIR_64)/$$example.$(SO_EXT) $(BUILD_DIR)/$$example.vst3/Contents/$(VST3_ARCH_64)/; \
				chmod +x $(BUILD_DIR)/$$example.vst3/Contents/$(VST3_ARCH_64)/$$example.$(SO_EXT); \
			fi; \
			if [ -f "$(BUILD_DIR_32)/$$example.$(SO_EXT)" ]; then \
				mkdir -p $(BUILD_DIR)/$$example.vst3/Contents/$(VST3_ARCH_32); \
				cp $(BUILD_DIR_32)/$$example.$(SO_EXT) $(BUILD_DIR)/$$example.vst3/Contents/$(VST3_ARCH_32)/; \
				chmod +x $(BUILD_DIR)/$$example.vst3/Contents/$(VST3_ARCH_32)/$$example.$(SO_EXT); \
			fi; \
			rm -rf ~/.vst3/$$example.vst3; \
			cp -r $(BUILD_DIR)/$$example.vst3 ~/.vst3/; \
			echo "Installed: ~/.vst3/$$example.vst3"; \
		fi; \
	done
	@echo "All example plugins installed successfully"

# Create VST3 bundle for a specific plugin
bundle: PLUGIN_NAME ?= gain
bundle:
	@echo "Creating VST3 bundle for $(PLUGIN_NAME)"
	@rm -rf $(BUILD_DIR)/$(PLUGIN_NAME).vst3
	@if [ -f "$(BUILD_DIR_64)/$(PLUGIN_NAME).$(SO_EXT)" ]; then \
		mkdir -p $(BUILD_DIR)/$(PLUGIN_NAME).vst3/Contents/$(VST3_ARCH_64); \
		cp $(BUILD_DIR_64)/$(PLUGIN_NAME).$(SO_EXT) $(BUILD_DIR)/$(PLUGIN_NAME).vst3/Contents/$(VST3_ARCH_64)/; \
		chmod +x $(BUILD_DIR)/$(PLUGIN_NAME).vst3/Contents/$(VST3_ARCH_64)/$(PLUGIN_NAME).$(SO_EXT); \
		echo "Added 64-bit binary to bundle"; \
	fi; \
	if [ -f "$(BUILD_DIR_32)/$(PLUGIN_NAME).$(SO_EXT)" ]; then \
		mkdir -p $(BUILD_DIR)/$(PLUGIN_NAME).vst3/Contents/$(VST3_ARCH_32); \
		cp $(BUILD_DIR_32)/$(PLUGIN_NAME).$(SO_EXT) $(BUILD_DIR)/$(PLUGIN_NAME).vst3/Contents/$(VST3_ARCH_32)/; \
		chmod +x $(BUILD_DIR)/$(PLUGIN_NAME).vst3/Contents/$(VST3_ARCH_32)/$(PLUGIN_NAME).$(SO_EXT); \
		echo "Added 32-bit binary to bundle"; \
	fi; \
	if [ -d "$(BUILD_DIR)/$(PLUGIN_NAME).vst3" ]; then \
		echo "VST3 bundle created: $(BUILD_DIR)/$(PLUGIN_NAME).vst3"; \
	else \
		echo "Error: No binaries found. Run 'make build' first."; \
		exit 1; \
	fi

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)

# Run Go linting
lint:
	@echo "Running Go linters"
	@export PATH="$$HOME/go/bin:$$PATH" && command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; exit 1; }
	@export PATH="$$HOME/go/bin:$$PATH" && golangci-lint run ./pkg/... ./examples/...

# Run Go formatting check
fmt-check:
	@echo "Checking Go formatting"
	@unformatted=$$(gofmt -l pkg/ examples/); \
	if [ -n "$$unformatted" ]; then \
		echo "The following files are not formatted:"; \
		echo "$$unformatted"; \
		echo "Run 'make fmt' to fix formatting"; \
		exit 1; \
	fi

# Format Go code
fmt:
	@echo "Formatting Go code"
	gofmt -w pkg/ examples/

# Run Go tests
test-go:
	@echo "Running Go unit tests (non-CGO packages only)"
	go test ./pkg/vst3/...

# Run VST3 validator on both architectures
test-validate: PLUGIN_NAME ?= gain
test-validate: test-validate-64 test-validate-32

# Run VST3 validator on 64-bit plugin
test-validate-64: PLUGIN_NAME ?= gain
test-validate-64: bundle
	@echo "Running VST3 validator on $(PLUGIN_NAME).vst3 (64-bit)"
	validator $(BUILD_DIR)/$(PLUGIN_NAME).vst3

# Run VST3 validator on 32-bit plugin (if available)
test-validate-32: PLUGIN_NAME ?= gain
test-validate-32:
ifeq ($(PLATFORM),linux)
	@if [ -f "$(BUILD_DIR_32)/$(PLUGIN_NAME).$(SO_EXT)" ]; then \
		echo "Running VST3 validator on $(PLUGIN_NAME).vst3 (32-bit)"; \
		rm -rf $(BUILD_DIR)/$(PLUGIN_NAME)-32.vst3; \
		mkdir -p $(BUILD_DIR)/$(PLUGIN_NAME)-32.vst3/Contents/$(VST3_ARCH_32); \
		cp $(BUILD_DIR_32)/$(PLUGIN_NAME).$(SO_EXT) $(BUILD_DIR)/$(PLUGIN_NAME)-32.vst3/Contents/$(VST3_ARCH_32)/; \
		chmod +x $(BUILD_DIR)/$(PLUGIN_NAME)-32.vst3/Contents/$(VST3_ARCH_32)/$(PLUGIN_NAME).$(SO_EXT); \
		validator $(BUILD_DIR)/$(PLUGIN_NAME)-32.vst3 || echo "32-bit validation failed"; \
	else \
		echo "No 32-bit binary found for $(PLUGIN_NAME)"; \
	fi
else
	@echo "32-bit validation not supported on $(PLATFORM)"
endif

# Run quick validation (errors only)
test-quick: PLUGIN_NAME ?= gain
test-quick: bundle
	@echo "Running quick validation (errors only)"
	validator -q $(BUILD_DIR)/$(PLUGIN_NAME).vst3

# Run extensive validation tests
test-extensive: PLUGIN_NAME ?= gain
test-extensive: bundle
	@echo "Running extensive validation tests (may take a long time)"
	validator -e $(BUILD_DIR)/$(PLUGIN_NAME).vst3

# Run validation with local instance per test
test-local: PLUGIN_NAME ?= gain
test-local: bundle
	@echo "Running validation with local instance per test"
	validator -l $(BUILD_DIR)/$(PLUGIN_NAME).vst3

# Run validation on the bundle version
test-bundle: PLUGIN_NAME ?= gain
test-bundle: bundle
	@echo "Running validation on VST3 bundle"
	validator $(BUILD_DIR)/$(PLUGIN_NAME).vst3

# List all VST3 plugins found by validator
test-list:
	@echo "Listing all installed VST3 plugins"
	validator -list

# Run validator selftest
test-selftest:
	@echo "Running validator selftest"
	validator -selftest

# Run all tests
test: fmt-check lint test-go test-validate

# Run automated validator test suite
test-auto: PLUGIN_NAME ?= gain
test-auto:
	@./scripts/test_validator.sh $(PLUGIN_NAME)

# Run all validation tests
test-all: fmt-check lint test-go test-validate test-extensive test-bundle

# List discovered examples
list-examples:
	@echo "Found example plugins:"
	@for dir in $(EXAMPLES_DIR)/*; do \
		if [ -d "$$dir" ] && [ -f "$$dir/main.go" ]; then \
			example=$$(basename $$dir); \
			echo "  $$example"; \
		fi; \
	done

# Help
help:
	@echo "VST3Go Makefile targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  make build        - Build all example plugins (both 32-bit and 64-bit)"
	@echo "  make build-64     - Build only 64-bit plugins"
	@echo "  make build-32     - Build only 32-bit plugins (Linux only)"
	@echo "  make install      - Build and install all example plugins to ~/.vst3"
	@echo "  make bundle       - Create VST3 bundle for a plugin (use PLUGIN_NAME=...)"
	@echo "  make clean        - Remove all build artifacts"
	@echo "  make list-examples - List all discovered example plugins"
	@echo ""
	@echo "Code Quality targets:"
	@echo "  make lint         - Run Go linters"
	@echo "  make fmt          - Format Go code"
	@echo "  make fmt-check    - Check Go formatting"
	@echo ""
	@echo "Test targets:"
	@echo "  make test         - Run formatting check, linting, Go tests and basic VST3 validation"
	@echo "  make test-go      - Run only Go unit tests"
	@echo "  make test-validate - Run VST3 validator on both 32-bit and 64-bit plugins"
	@echo "  make test-validate-64 - Run VST3 validator on 64-bit plugin only"
	@echo "  make test-validate-32 - Run VST3 validator on 32-bit plugin only (Linux)"
	@echo "  make test-quick   - Run quick validation (errors only)"
	@echo "  make test-extensive - Run extensive validation tests"
	@echo "  make test-local   - Run validation with local instance per test"
	@echo "  make test-bundle  - Run validation on the VST3 bundle"
	@echo "  make test-list    - List all installed VST3 plugins"
	@echo "  make test-selftest - Run validator selftest"
	@echo "  make test-all     - Run all tests (formatting, linting, Go + all validations)"
	@echo ""
	@echo "Examples:"
	@echo "  make                         # Build all plugins"
	@echo "  make install                 # Build and install all plugins"
	@echo "  make test-validate PLUGIN_NAME=delay  # Test specific plugin"
	@echo ""
	@echo "  make help         - Show this help message"

.PHONY: all build build-64 build-32 install bundle clean help list-examples \
	lint fmt fmt-check test test-go test-validate test-validate-64 test-validate-32 \
	test-quick test-extensive test-local test-bundle test-list test-selftest test-all