# VST3Go

A Go wrapper for building VST3 plugins using the VST3 C API.

## Project Structure

```
vst3go/
├── bridge/         # C bridge code for VST3 entry points
├── pkg/
│   ├── vst3/      # Go bindings for VST3 C API types
│   └── plugin/    # Plugin framework
├── examples/      # Example plugins
├── include/       # VST3 C API headers
└── build/         # Build output
```

## Building

```bash
# Build example plugin
make

# Create VST3 bundle
make bundle

# Clean build artifacts
make clean
```

## Status

This is a minimal MVP implementation. Currently in Phase 1 of development.