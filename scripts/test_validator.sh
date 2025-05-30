#!/bin/bash
# VST3Go Validator Test Harness

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
PLUGIN_NAME="${1:-SimpleGain}"

echo "VST3Go Validator Test Harness"
echo "============================="
echo "Testing plugin: $PLUGIN_NAME"
echo ""

# Build the plugin
echo "Building plugin..."
cd "$PROJECT_DIR"
make clean > /dev/null 2>&1
make gain > /dev/null 2>&1
make install > /dev/null 2>&1

# Run different validator tests
echo ""
echo "Running validator tests..."
echo ""

# Quick test (errors only)
echo "1. Quick validation (errors only):"
if validator -q ~/.vst3/${PLUGIN_NAME}.vst3 2>&1 | grep -E "(ERROR|FAIL)" > /dev/null; then
    echo "   ❌ FAILED - Errors detected"
    validator -q ~/.vst3/${PLUGIN_NAME}.vst3 2>&1 | grep -E "(ERROR|FAIL)"
else
    echo "   ✅ PASSED - No errors"
fi

# Basic test
echo ""
echo "2. Basic validation:"
if validator ~/.vst3/${PLUGIN_NAME}.vst3 > /tmp/validator_output.txt 2>&1; then
    echo "   ✅ PASSED"
else
    echo "   ❌ FAILED"
    tail -20 /tmp/validator_output.txt
fi

# Count tests passed
echo ""
echo "3. Test summary:"
if [ -f /tmp/validator_output.txt ]; then
    TOTAL_TESTS=$(grep -c "Test" /tmp/validator_output.txt || echo "0")
    PASSED_TESTS=$(grep -c "succeeded" /tmp/validator_output.txt || echo "0")
    echo "   Total tests: $TOTAL_TESTS"
    echo "   Passed: $PASSED_TESTS"
fi

echo ""
echo "Test complete!"