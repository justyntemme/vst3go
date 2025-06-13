#!/bin/bash

# Test script for all modified plugins
# This script builds and validates each plugin to ensure functionality

set -e  # Exit on error

echo "=== DSP Cleanup Plugin Validation Test ==="
echo "Testing all modified plugins..."
echo

# List of plugins to test
PLUGINS=(
    "gain"
    "drumbus"
    "vocalstrip"
    "chain_fx"
    "debug_example"
)

# Track results
PASSED=0
FAILED=0
FAILED_PLUGINS=()

# Test each plugin
for plugin in "${PLUGINS[@]}"; do
    echo "----------------------------------------"
    echo "Testing plugin: $plugin"
    echo "----------------------------------------"
    
    # Build the plugin
    echo "Building $plugin..."
    if make install PLUGIN_NAME=$plugin > /tmp/${plugin}_build.log 2>&1; then
        echo "✓ Build successful"
        
        # Validate the plugin
        echo "Validating $plugin..."
        if validator ~/.vst3/${plugin}.vst3 > /tmp/${plugin}_validate.log 2>&1; then
            # Check if all tests passed
            if grep -q "Result: 47 tests passed, 0 tests failed" /tmp/${plugin}_validate.log; then
                echo "✓ Validation successful (47/47 tests passed)"
                PASSED=$((PASSED + 1))
            else
                echo "✗ Validation failed - not all tests passed"
                FAILED=$((FAILED + 1))
                FAILED_PLUGINS+=("$plugin")
                tail -10 /tmp/${plugin}_validate.log
            fi
        else
            echo "✗ Validation failed"
            FAILED=$((FAILED + 1))
            FAILED_PLUGINS+=("$plugin")
            tail -10 /tmp/${plugin}_validate.log
        fi
    else
        echo "✗ Build failed"
        FAILED=$((FAILED + 1))
        FAILED_PLUGINS+=("$plugin")
        tail -10 /tmp/${plugin}_build.log
    fi
    
    echo
done

# Summary
echo "========================================"
echo "SUMMARY"
echo "========================================"
echo "Total plugins tested: ${#PLUGINS[@]}"
echo "Passed: $PASSED"
echo "Failed: $FAILED"

if [ $FAILED -gt 0 ]; then
    echo
    echo "Failed plugins:"
    for plugin in "${FAILED_PLUGINS[@]}"; do
        echo "  - $plugin"
    done
    exit 1
else
    echo
    echo "✓ All plugins passed validation!"
fi

# Clean up log files
rm -f /tmp/*_build.log /tmp/*_validate.log