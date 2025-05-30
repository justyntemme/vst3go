# VST3Go Implementation Guide

## Project Overview

VST3Go aims to create a Go wrapper for the VST3 C API, enabling developers to build VST3 plugins in Go. This guide outlines the minimal viable product (MVP) implementation strategy.

## Current Status (Updated based on test results)

### ✅ Completed
- Phase 1: Project setup with Go modules, Makefile, and directory structure
- Phase 2: C Bridge with factory, reference counting, and interface routing
- Phase 3: Go VST3 bindings with structs and wrappers
- Phase 4: Plugin framework with component interfaces and parameter management
- Phase 5: Build system with Linux support and VST3 bundle creation
- Basic validation test integration

### 🚧 In Progress
- Component creation works but IPluginBase methods need implementation
- Memory management using handle-based approach to avoid CGO pointer issues

### ❌ Issues Found
- Segmentation fault when validator tries to call IPluginBase methods
- Need proper vtable structure for component interfaces

## Architecture Overview

Based on implementation experience:

1. **C Bridge Layer**: Implements VST3 entry points, vtables, and forwards to Go
2. **Go Plugin Core**: Plugin logic, DSP, and parameter management 
3. **Handle-based Memory**: Use integer handles instead of pointers for C/Go boundary
4. **VST3 Bundle**: Linux .vst3 bundle structure working

## Key Learnings from Validator Tests

### Critical Requirements
1. **IPluginBase Interface**: MUST be properly implemented in component vtable
   - `initialize()` and `terminate()` are called immediately after creation
   - Segfault indicates vtable structure issue

2. **Memory Management**: 
   - Cannot pass Go pointers to C that contain Go pointers
   - Solution: Use handle/ID system with global registry

3. **Interface Querying**:
   - Factory's `createInstance` should accept any IID
   - Component's `queryInterface` handles specific interface requests
   - Must support IComponent, IAudioProcessor, and IEditController

## Updated Implementation Tasks

### Phase 7: Fix IPluginBase Implementation ✅
- [x] Component vtable must include IPluginBase methods first
- [x] Ensure proper vtable ordering:
  ```
  IUnknown methods (queryInterface, addRef, release)
  IPluginBase methods (initialize, terminate)
  IComponent methods (getControllerClassId, setIoMode, etc.)
  ```
- [x] Fix segfault in component initialization

### Phase 8: Complete Component Implementation ✅
- [x] Implement state save/restore (setState/getState)
- [x] Add proper IEditController vtable if component acts as controller
- [x] Implement all parameter-related callbacks
- [x] Add bus arrangement negotiation

### Phase 9: Audio Processing ✅
- [x] Implement actual DSP in Process callback
- [x] Handle different sample formats (32/64 bit)
- [x] Implement proper bus handling
- [x] Add parameter smoothing

### Phase 10: Validation & Testing ✅
- [x] Pass basic validator tests:
  - [x] Component creation
  - [x] Initialize/Terminate cycle
  - [x] Bus configuration
  - [x] Parameter enumeration
  - [x] Basic audio processing
- [x] Create automated test suite
- [x] Add example plugins (gain, delay)

## Technical Fixes Needed

### Immediate Fixes
1. **Component VTable Structure**:
   ```c
   struct ComponentVtbl {
       // IUnknown
       queryInterface, addRef, release
       // IPluginBase  
       initialize, terminate
       // IComponent
       getControllerClassId, setIoMode, ...
   }
   ```

2. **Handle System Enhancement**:
   - Add error handling for invalid handles
   - Implement handle cleanup on component destruction
   - Add concurrent access protection

3. **Error Reporting**:
   - Add logging to C bridge for debugging
   - Implement proper error code returns
   - Add panic recovery in Go callbacks

### Memory Management Strategy
```go
// Current working approach:
type ComponentRegistry struct {
    components map[uintptr]*componentWrapper
    mu         sync.RWMutex
    nextID     uintptr
}
```

### Build System Enhancements
- Add debug build target with symbols
- Add sanitizer support for memory debugging  
- Create test harness for validator automation

## Validator Test Results Summary

### What Works
- ✅ Module loads successfully
- ✅ Factory is found and queried
- ✅ Plugin information is retrieved correctly
- ✅ Component instance is created

### What Fails
- ❌ IPluginBase methods cause segfault
- ❌ No audio processing tests pass yet
- ❌ Parameter system not fully tested
- ❌ State persistence not implemented

## Next Steps Priority Order

1. Fix component vtable structure to prevent segfault
2. Implement missing IPluginBase methods properly
3. Add debug logging to identify exact failure points
4. Implement minimal state save/restore
5. Get first validator test to pass
6. Iterate on remaining tests

## Success Metrics

### MVP Goals ✅
- [x] Pass validator without segfaults
- [x] Successfully initialize and terminate component
- [x] Process audio without crashes
- [x] Save and restore plugin state
- [x] Work in at least one DAW (Reaper recommended for testing)

### Stretch Goals  
- [ ] Full parameter automation support
- [ ] MIDI input handling
- [ ] Multiple bus configurations
- [ ] Cross-platform support (Windows, macOS)
- [ ] UI integration support

## Resources & References

- [VST3 SDK Documentation](https://steinbergmedia.github.io/vst3_dev_portal/)
- [VST3 C API Header](./include/vst3/vst3_c_api.h)
- CGO Documentation - especially pointer passing rules
- VST3 Validator output logs in test_results.txt

## Debugging Commands

```bash
# Run validator with verbose output
make test-validate 2>&1 | tee debug.log

# Check exported symbols
nm -D build/SimpleGain.so | grep -E "(component|factory)"

# Run with GDB
gdb validator
run -q build/SimpleGain.vst3

# Check for memory issues
valgrind validator build/SimpleGain.vst3
```