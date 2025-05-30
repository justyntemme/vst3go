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

# Compiler flags
CFLAGS := -fPIC -I./include -O2
LDFLAGS := -shared

# Debug flags
DEBUG_CFLAGS := -fPIC -I./include -g -O0 -DDEBUG_VST3GO
DEBUG_LDFLAGS := -shared -g

# Default target
all: gain

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

# Build SimpleGain plugin as a single shared library
$(BUILD_DIR)/SimpleGain.$(SO_EXT): examples/gain/main.go $(BRIDGE_DIR)/bridge.c $(BRIDGE_DIR)/component.c
	@mkdir -p $(BUILD_DIR)
	@echo "Building SimpleGain VST3 plugin as single library"
	CGO_CFLAGS="$(CFLAGS)" CGO_LDFLAGS="$(LDFLAGS)" go build -buildmode=c-shared \
		-o $@ \
		-ldflags="-s -w" \
		./examples/gain

# Build SimpleDelay plugin as a single shared library
$(BUILD_DIR)/SimpleDelay.$(SO_EXT): examples/delay/main.go $(BRIDGE_DIR)/bridge.c $(BRIDGE_DIR)/component.c
	@mkdir -p $(BUILD_DIR)
	@echo "Building SimpleDelay VST3 plugin as single library"
	CGO_CFLAGS="$(CFLAGS)" CGO_LDFLAGS="$(LDFLAGS)" go build -buildmode=c-shared \
		-o $@ \
		-ldflags="-s -w" \
		./examples/delay

# Create VST3 bundle
bundle: $(BUILD_DIR)/$(PLUGIN_NAME).$(SO_EXT)
	@echo "Creating VST3 bundle for $(PLUGIN_NAME)"
	@rm -rf $(BUILD_DIR)/$(PLUGIN_NAME).vst3
	@mkdir -p $(BUILD_DIR)/$(PLUGIN_NAME).vst3/Contents/$(VST3_ARCH)
	@cp $(BUILD_DIR)/$(PLUGIN_NAME).$(SO_EXT) $(BUILD_DIR)/$(PLUGIN_NAME).vst3/Contents/$(VST3_ARCH)/
	@chmod +x $(BUILD_DIR)/$(PLUGIN_NAME).vst3/Contents/$(VST3_ARCH)/$(PLUGIN_NAME).$(SO_EXT)
	@echo "VST3 bundle created: $(BUILD_DIR)/$(PLUGIN_NAME).vst3"

# Install VST3 plugin to user's VST3 directory
install: bundle
	@echo "Installing $(PLUGIN_NAME).vst3 to ~/.vst3"
	@mkdir -p ~/.vst3
	@rm -rf ~/.vst3/$(PLUGIN_NAME).vst3
	@cp -r $(BUILD_DIR)/$(PLUGIN_NAME).vst3 ~/.vst3/
	@echo "Installation complete: ~/.vst3/$(PLUGIN_NAME).vst3"

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)

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
test: test-go test-validate

# Run automated validator test suite
test-auto:
	@./scripts/test_validator.sh $(PLUGIN_NAME)

# Run all validation tests
test-all: test-go test-validate test-extensive test-bundle

# Help
help:
	@echo "VST3Go Makefile targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  make gain         - Build the SimpleGain example plugin"
	@echo "  make delay        - Build the SimpleDelay example plugin"
	@echo "  make bundle       - Create VST3 bundle for current plugin"
	@echo "  make install      - Install plugin to ~/.vst3 directory"
	@echo "  make clean        - Remove all build artifacts"
	@echo ""
	@echo "Test targets:"
	@echo "  make test         - Run Go tests and basic VST3 validation"
	@echo "  make test-go      - Run only Go unit tests"
	@echo "  make test-validate - Run VST3 validator on the plugin"
	@echo "  make test-quick   - Run quick validation (errors only)"
	@echo "  make test-extensive - Run extensive validation tests"
	@echo "  make test-local   - Run validation with local instance per test"
	@echo "  make test-bundle  - Run validation on the VST3 bundle"
	@echo "  make test-list    - List all installed VST3 plugins"
	@echo "  make test-selftest - Run validator selftest"
	@echo "  make test-all     - Run all tests (Go + all validations)"
	@echo ""
	@echo "  make help         - Show this help message"

.PHONY: all gain bundle install clean help \
	test test-go test-validate test-quick test-extensive \
	test-local test-bundle test-list test-selftest test-all