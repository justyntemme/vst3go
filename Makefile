# VST3Go Makefile

# Platform detection
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Linux)
    PLATFORM := linux
    SO_EXT := so
    VST3_ARCH := x86_64-linux
    RPATH_FLAG := -Wl,-rpath,'$$ORIGIN'
endif
ifeq ($(UNAME_S),Darwin)
    PLATFORM := darwin
    SO_EXT := dylib
    VST3_ARCH := MacOS
    RPATH_FLAG := -Wl,-rpath,@loader_path
endif

# Build variables
BUILD_DIR := build
BRIDGE_DIR := bridge
EXAMPLES_DIR := examples

# Compiler flags - Default to debug mode
CFLAGS := -fPIC -I./include -g -O0 -DDEBUG_VST3GO
LDFLAGS := -shared -g

# Debug flags
DEBUG_CFLAGS := -fPIC -I./include -g -O0 -DDEBUG_VST3GO
DEBUG_LDFLAGS := -shared -g

# Default target
all: build

# Build all example plugins
build:
	@mkdir -p $(BUILD_DIR)
	@for dir in $(EXAMPLES_DIR)/*; do \
		if [ -d "$$dir" ] && [ -f "$$dir/main.go" ]; then \
			example=$$(basename $$dir); \
			echo "Building $$example plugin"; \
			CGO_CFLAGS="$(CFLAGS)" CGO_LDFLAGS="$(LDFLAGS)" go build -buildmode=c-shared \
				-o $(BUILD_DIR)/$$example.$(SO_EXT) \
				./$$dir || exit 1; \
		fi; \
	done
	@echo "All plugins built successfully"

# Install all example VST3 plugins to user's VST3 directory
install: build
	@echo "Installing all example VST3 plugins to ~/.vst3"
	@mkdir -p ~/.vst3
	@for dir in $(EXAMPLES_DIR)/*; do \
		if [ -d "$$dir" ] && [ -f "$$dir/main.go" ]; then \
			example=$$(basename $$dir); \
			plugin_file=$(BUILD_DIR)/$$example.$(SO_EXT); \
			if [ -f "$$plugin_file" ]; then \
				echo "Creating and installing $$example.vst3 bundle"; \
				rm -rf $(BUILD_DIR)/$$example.vst3; \
				mkdir -p $(BUILD_DIR)/$$example.vst3/Contents/$(VST3_ARCH); \
				cp $$plugin_file $(BUILD_DIR)/$$example.vst3/Contents/$(VST3_ARCH)/; \
				chmod +x $(BUILD_DIR)/$$example.vst3/Contents/$(VST3_ARCH)/$$example.$(SO_EXT); \
				rm -rf ~/.vst3/$$example.vst3; \
				cp -r $(BUILD_DIR)/$$example.vst3 ~/.vst3/; \
				echo "Installed: ~/.vst3/$$example.vst3"; \
			fi; \
		fi; \
	done
	@echo "All example plugins installed successfully"

# Create VST3 bundle for a specific plugin
bundle: PLUGIN_NAME ?= gain
bundle:
	@echo "Creating VST3 bundle for $(PLUGIN_NAME)"
	@if [ -f "$(BUILD_DIR)/$(PLUGIN_NAME).$(SO_EXT)" ]; then \
		rm -rf $(BUILD_DIR)/$(PLUGIN_NAME).vst3; \
		mkdir -p $(BUILD_DIR)/$(PLUGIN_NAME).vst3/Contents/$(VST3_ARCH); \
		cp $(BUILD_DIR)/$(PLUGIN_NAME).$(SO_EXT) $(BUILD_DIR)/$(PLUGIN_NAME).vst3/Contents/$(VST3_ARCH)/; \
		chmod +x $(BUILD_DIR)/$(PLUGIN_NAME).vst3/Contents/$(VST3_ARCH)/$(PLUGIN_NAME).$(SO_EXT); \
		echo "VST3 bundle created: $(BUILD_DIR)/$(PLUGIN_NAME).vst3"; \
	else \
		echo "Error: $(BUILD_DIR)/$(PLUGIN_NAME).$(SO_EXT) not found. Run 'make build' first."; \
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

# Run VST3 validator on a specific plugin
test-validate: PLUGIN_NAME ?= gain
test-validate: bundle
	@echo "Running VST3 validator on $(PLUGIN_NAME).vst3"
	validator $(BUILD_DIR)/$(PLUGIN_NAME).vst3

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
	@echo "  make build        - Build all example plugins"
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
	@echo "  make test-validate - Run VST3 validator on a plugin (use PLUGIN_NAME=...)"
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

.PHONY: all build install bundle clean help list-examples \
	lint fmt fmt-check test test-go test-validate test-quick test-extensive \
	test-local test-bundle test-list test-selftest test-all