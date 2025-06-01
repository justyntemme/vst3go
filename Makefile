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
PLUGIN_NAME ?= SimpleGain

# Compiler flags - Default to debug mode
CFLAGS := -fPIC -I./include -g -O0 -DDEBUG_VST3GO
LDFLAGS := -shared -g

# Debug flags
DEBUG_CFLAGS := -fPIC -I./include -g -O0 -DDEBUG_VST3GO
DEBUG_LDFLAGS := -shared -g

# Default target
all: gain

# Build all example plugins
all-examples: gain delay filter compressor gate

# Build gain example
gain: PLUGIN_NAME := SimpleGain
gain: $(BUILD_DIR)/SimpleGain.$(SO_EXT)

# Build gain example with debug
gain-debug: PLUGIN_NAME := SimpleGain
gain-debug: CFLAGS := $(DEBUG_CFLAGS)
gain-debug: LDFLAGS := $(DEBUG_LDFLAGS)
gain-debug: $(BUILD_DIR)/SimpleGain.$(SO_EXT)

# Build delay example
delay: PLUGIN_NAME := SimpleDelay
delay: $(BUILD_DIR)/SimpleDelay.$(SO_EXT)

# Build filter example
filter: PLUGIN_NAME := MultiModeFilter
filter: $(BUILD_DIR)/MultiModeFilter.$(SO_EXT)

# Build SimpleGain plugin as a single shared library
$(BUILD_DIR)/SimpleGain.$(SO_EXT): examples/gain/main.go $(BRIDGE_DIR)/bridge.c $(BRIDGE_DIR)/component.c
	@mkdir -p $(BUILD_DIR)
	@echo "Building SimpleGain VST3 plugin as single library"
	CGO_CFLAGS="$(CFLAGS)" CGO_LDFLAGS="$(LDFLAGS)" go build -buildmode=c-shared \
		-o $@ \
		./examples/gain

# Build SimpleDelay plugin as a single shared library
$(BUILD_DIR)/SimpleDelay.$(SO_EXT): examples/delay/main.go $(BRIDGE_DIR)/bridge.c $(BRIDGE_DIR)/component.c
	@mkdir -p $(BUILD_DIR)
	@echo "Building SimpleDelay VST3 plugin as single library"
	CGO_CFLAGS="$(CFLAGS)" CGO_LDFLAGS="$(LDFLAGS)" go build -buildmode=c-shared \
		-o $@ \
		./examples/delay

# Build MultiModeFilter plugin as a single shared library
$(BUILD_DIR)/MultiModeFilter.$(SO_EXT): examples/filter/main.go $(BRIDGE_DIR)/bridge.c $(BRIDGE_DIR)/component.c
	@mkdir -p $(BUILD_DIR)
	@echo "Building MultiModeFilter VST3 plugin as single library"
	CGO_CFLAGS="$(CFLAGS)" CGO_LDFLAGS="$(LDFLAGS)" go build -buildmode=c-shared \
		-o $@ \
		./examples/filter

# Build compressor example
compressor: PLUGIN_NAME := MasterCompressor
compressor: $(BUILD_DIR)/MasterCompressor.$(SO_EXT)

# Build MasterCompressor plugin as a single shared library
$(BUILD_DIR)/MasterCompressor.$(SO_EXT): examples/compressor/main.go $(BRIDGE_DIR)/bridge.c $(BRIDGE_DIR)/component.c
	@mkdir -p $(BUILD_DIR)
	@echo "Building MasterCompressor VST3 plugin as single library"
	CGO_CFLAGS="$(CFLAGS)" CGO_LDFLAGS="$(LDFLAGS)" go build -buildmode=c-shared \
		-o $@ \
		./examples/compressor

# Build gate example
gate: PLUGIN_NAME := StudioGate
gate: $(BUILD_DIR)/StudioGate.$(SO_EXT)

# Build StudioGate plugin as a single shared library
$(BUILD_DIR)/StudioGate.$(SO_EXT): examples/gate/main.go $(BRIDGE_DIR)/bridge.c $(BRIDGE_DIR)/component.c
	@mkdir -p $(BUILD_DIR)
	@echo "Building StudioGate VST3 plugin as single library"
	CGO_CFLAGS="$(CFLAGS)" CGO_LDFLAGS="$(LDFLAGS)" go build -buildmode=c-shared \
		-o $@ \
		./examples/gate

# Create VST3 bundle
bundle: $(BUILD_DIR)/$(PLUGIN_NAME).$(SO_EXT)
	@echo "Creating VST3 bundle for $(PLUGIN_NAME)"
	@rm -rf $(BUILD_DIR)/$(PLUGIN_NAME).vst3
	@mkdir -p $(BUILD_DIR)/$(PLUGIN_NAME).vst3/Contents/$(VST3_ARCH)
	@cp $(BUILD_DIR)/$(PLUGIN_NAME).$(SO_EXT) $(BUILD_DIR)/$(PLUGIN_NAME).vst3/Contents/$(VST3_ARCH)/
	@chmod +x $(BUILD_DIR)/$(PLUGIN_NAME).vst3/Contents/$(VST3_ARCH)/$(PLUGIN_NAME).$(SO_EXT)
	@echo "VST3 bundle created: $(BUILD_DIR)/$(PLUGIN_NAME).vst3"

# Install all example VST3 plugins to user's VST3 directory
install: all-examples
	@echo "Installing all example VST3 plugins to ~/.vst3"
	@mkdir -p ~/.vst3
	@echo "Creating and installing SimpleGain.vst3 bundle"
	@rm -rf $(BUILD_DIR)/SimpleGain.vst3
	@mkdir -p $(BUILD_DIR)/SimpleGain.vst3/Contents/$(VST3_ARCH)
	@cp $(BUILD_DIR)/SimpleGain.$(SO_EXT) $(BUILD_DIR)/SimpleGain.vst3/Contents/$(VST3_ARCH)/
	@chmod +x $(BUILD_DIR)/SimpleGain.vst3/Contents/$(VST3_ARCH)/SimpleGain.$(SO_EXT)
	@rm -rf ~/.vst3/SimpleGain.vst3
	@cp -r $(BUILD_DIR)/SimpleGain.vst3 ~/.vst3/
	@echo "Installed: ~/.vst3/SimpleGain.vst3"
	@echo "Creating and installing SimpleDelay.vst3 bundle"
	@rm -rf $(BUILD_DIR)/SimpleDelay.vst3
	@mkdir -p $(BUILD_DIR)/SimpleDelay.vst3/Contents/$(VST3_ARCH)
	@cp $(BUILD_DIR)/SimpleDelay.$(SO_EXT) $(BUILD_DIR)/SimpleDelay.vst3/Contents/$(VST3_ARCH)/
	@chmod +x $(BUILD_DIR)/SimpleDelay.vst3/Contents/$(VST3_ARCH)/SimpleDelay.$(SO_EXT)
	@rm -rf ~/.vst3/SimpleDelay.vst3
	@cp -r $(BUILD_DIR)/SimpleDelay.vst3 ~/.vst3/
	@echo "Installed: ~/.vst3/SimpleDelay.vst3"
	@echo "Creating and installing MultiModeFilter.vst3 bundle"
	@rm -rf $(BUILD_DIR)/MultiModeFilter.vst3
	@mkdir -p $(BUILD_DIR)/MultiModeFilter.vst3/Contents/$(VST3_ARCH)
	@cp $(BUILD_DIR)/MultiModeFilter.$(SO_EXT) $(BUILD_DIR)/MultiModeFilter.vst3/Contents/$(VST3_ARCH)/
	@chmod +x $(BUILD_DIR)/MultiModeFilter.vst3/Contents/$(VST3_ARCH)/MultiModeFilter.$(SO_EXT)
	@rm -rf ~/.vst3/MultiModeFilter.vst3
	@cp -r $(BUILD_DIR)/MultiModeFilter.vst3 ~/.vst3/
	@echo "Installed: ~/.vst3/MultiModeFilter.vst3"
	@echo "Creating and installing MasterCompressor.vst3 bundle"
	@rm -rf $(BUILD_DIR)/MasterCompressor.vst3
	@mkdir -p $(BUILD_DIR)/MasterCompressor.vst3/Contents/$(VST3_ARCH)
	@cp $(BUILD_DIR)/MasterCompressor.$(SO_EXT) $(BUILD_DIR)/MasterCompressor.vst3/Contents/$(VST3_ARCH)/
	@chmod +x $(BUILD_DIR)/MasterCompressor.vst3/Contents/$(VST3_ARCH)/MasterCompressor.$(SO_EXT)
	@rm -rf ~/.vst3/MasterCompressor.vst3
	@cp -r $(BUILD_DIR)/MasterCompressor.vst3 ~/.vst3/
	@echo "Installed: ~/.vst3/MasterCompressor.vst3"
	@echo "Creating and installing StudioGate.vst3 bundle"
	@rm -rf $(BUILD_DIR)/StudioGate.vst3
	@mkdir -p $(BUILD_DIR)/StudioGate.vst3/Contents/$(VST3_ARCH)
	@cp $(BUILD_DIR)/StudioGate.$(SO_EXT) $(BUILD_DIR)/StudioGate.vst3/Contents/$(VST3_ARCH)/
	@chmod +x $(BUILD_DIR)/StudioGate.vst3/Contents/$(VST3_ARCH)/StudioGate.$(SO_EXT)
	@rm -rf ~/.vst3/StudioGate.vst3
	@cp -r $(BUILD_DIR)/StudioGate.vst3 ~/.vst3/
	@echo "Installed: ~/.vst3/StudioGate.vst3"
	@echo "All example plugins installed successfully"

# Alias for install
install-all: install

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

# Run VST3 validator on the built plugin
test-validate: bundle
	@echo "Running VST3 validator on $(PLUGIN_NAME).vst3"
	validator $(BUILD_DIR)/$(PLUGIN_NAME).vst3

# Run quick validation (errors only)
test-quick: bundle
	@echo "Running quick validation (errors only)"
	validator -q $(BUILD_DIR)/$(PLUGIN_NAME).vst3

# Run extensive validation tests
test-extensive: bundle
	@echo "Running extensive validation tests (may take a long time)"
	validator -e $(BUILD_DIR)/$(PLUGIN_NAME).vst3

# Run validation with local instance per test
test-local: bundle
	@echo "Running validation with local instance per test"
	validator -l $(BUILD_DIR)/$(PLUGIN_NAME).vst3

# Run validation on the bundle version
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
test-auto:
	@./scripts/test_validator.sh $(PLUGIN_NAME)

# Run all validation tests
test-all: fmt-check lint test-go test-validate test-extensive test-bundle

# Help
help:
	@echo "VST3Go Makefile targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  make gain         - Build the SimpleGain example plugin"
	@echo "  make delay        - Build the SimpleDelay example plugin"
	@echo "  make filter       - Build the MultiModeFilter example plugin"
	@echo "  make compressor   - Build the MasterCompressor example plugin"
	@echo "  make gate         - Build the StudioGate example plugin"
	@echo "  make all-examples - Build all example plugins"
	@echo "  make bundle       - Create VST3 bundle for current plugin"
	@echo "  make install      - Build and install all example plugins to ~/.vst3"
	@echo "  make install-all  - Alias for 'make install'"
	@echo "  make clean        - Remove all build artifacts"
	@echo ""
	@echo "Code Quality targets:"
	@echo "  make lint         - Run Go linters"
	@echo "  make fmt          - Format Go code"
	@echo "  make fmt-check    - Check Go formatting"
	@echo ""
	@echo "Test targets:"
	@echo "  make test         - Run formatting check, linting, Go tests and basic VST3 validation"
	@echo "  make test-go      - Run only Go unit tests"
	@echo "  make test-validate - Run VST3 validator on the plugin"
	@echo "  make test-quick   - Run quick validation (errors only)"
	@echo "  make test-extensive - Run extensive validation tests"
	@echo "  make test-local   - Run validation with local instance per test"
	@echo "  make test-bundle  - Run validation on the VST3 bundle"
	@echo "  make test-list    - List all installed VST3 plugins"
	@echo "  make test-selftest - Run validator selftest"
	@echo "  make test-all     - Run all tests (formatting, linting, Go + all validations)"
	@echo ""
	@echo "  make help         - Show this help message"

.PHONY: all all-examples gain delay filter compressor gate bundle install install-all clean help \
	lint fmt fmt-check test test-go test-validate test-quick test-extensive \
	test-local test-bundle test-list test-selftest test-all