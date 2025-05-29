# VST3Go Implementation Guide

## Project Overview

VST3Go aims to create a Go wrapper for the VST3 C API, enabling developers to build VST3 plugins in Go. This guide outlines the minimal viable product (MVP) implementation strategy.

## Architecture Overview

Based on the VST3 C API analysis and common practices for C/Go audio plugin development, we'll use a hybrid approach:

1. **C Bridge Layer**: Minimal C code that implements VST3 entry points and forwards to Go
2. **Go Plugin Core**: Main plugin logic, DSP, and parameter management in Go
3. **Shared Library**: Compile Go as a shared library, link with C bridge
4. **VST3 Bundle**: Package as proper .vst3 format (platform-specific)

## Key VST3 Interfaces to Implement

### 1. Core Interfaces (Required)
- `Steinberg_IPluginFactory` - Plugin factory for creating instances
- `Steinberg_Vst_IComponent` - Main plugin component
- `Steinberg_Vst_IAudioProcessor` - Audio processing interface
- `Steinberg_FUnknown` - COM-style base interface

### 2. Additional Interfaces (MVP)
- `Steinberg_Vst_IEditController` - Parameter control (can be same object as IComponent)
- `Steinberg_IPluginBase` - Plugin initialization

## Implementation Tasks

### Phase 1: Project Setup
- [ ] Set up Go module structure
- [ ] Create build system (Makefile/CMake)
- [ ] Set up CGO configuration
- [ ] Create directory structure:
  ```
  /bridge/        # C bridge code
  /pkg/vst3/      # Go VST3 bindings
  /pkg/plugin/    # Plugin interface
  /examples/      # Example plugins
  /build/         # Build output
  ```

### Phase 2: C Bridge Implementation
- [ ] Create `bridge.c` with VST3 entry points:
  - [ ] `GetPluginFactory` export function
  - [ ] Factory implementation that calls into Go
  - [ ] COM-style vtable setup for interfaces
- [ ] Create `bridge.h` with function declarations
- [ ] Implement reference counting in C
- [ ] Set up interface query routing to Go

### Phase 3: Go VST3 Bindings
- [ ] Create Go structs matching VST3 C structures:
  - [ ] `ProcessData` wrapper
  - [ ] `AudioBusBuffers` wrapper
  - [ ] `ParameterInfo` wrapper
  - [ ] Interface vtables
- [ ] Implement Go functions for each VST3 interface method
- [ ] Create safe wrappers for unsafe pointers
- [ ] Handle memory management across C/Go boundary

### Phase 4: Plugin Framework
- [ ] Define Go plugin interface:
  ```go
  type Plugin interface {
      GetInfo() PluginInfo
      Process(input, output [][]float32) error
      GetParameterCount() int
      GetParameterInfo(index int) ParameterInfo
      // etc.
  }
  ```
- [ ] Implement base plugin struct with VST3 integration
- [ ] Create parameter management system
- [ ] Add audio buffer handling utilities

### Phase 5: Build System
- [ ] Create Makefile with targets:
  - [ ] `build-go`: Compile Go to shared library
  - [ ] `build-bridge`: Compile C bridge
  - [ ] `link`: Link everything into .vst3
  - [ ] `bundle`: Create VST3 bundle structure
- [ ] Handle platform differences (Linux/macOS/Windows)
- [ ] Add cross-compilation support

### Phase 6: Example Plugin
- [ ] Create simple gain plugin as example
- [ ] Implement all required interfaces
- [ ] Add basic parameter (gain control)
- [ ] Test audio processing

## Technical Considerations

### Memory Management
- Use C.malloc/C.free for memory crossing boundaries
- Implement proper reference counting
- Be careful with Go garbage collector and C pointers
- Pin Go memory when passing to C

### Threading
- VST3 has specific threading requirements
- Audio processing happens on real-time thread
- Use atomic operations for parameter access
- Avoid allocations in process callback

### Build Process
```bash
# 1. Build Go shared library
go build -buildmode=c-shared -o libvst3go.so ./cmd/plugin

# 2. Compile C bridge
gcc -c -fPIC -o bridge.o bridge/bridge.c

# 3. Link final plugin
gcc -shared -o plugin.vst3 bridge.o -L. -lvst3go -Wl,-rpath,'$ORIGIN'

# 4. Create bundle structure
mkdir -p plugin.vst3/Contents/x86_64-linux
mv plugin.vst3 plugin.vst3/Contents/x86_64-linux/
```

### Platform Specifics
- **Linux**: .so files, specific rpath handling
- **macOS**: .dylib files, bundle structure, code signing
- **Windows**: .dll files, different calling conventions

## MVP Feature Set

### Must Have
- [ ] Basic audio processing (stereo in/out)
- [ ] Factory and component creation
- [ ] Initialize/terminate lifecycle
- [ ] Process callback implementation
- [ ] Minimal parameter support (at least one parameter)
- [ ] Proper reference counting
- [ ] State save/load

### Nice to Have
- [ ] MIDI event processing
- [ ] Multiple bus support
- [ ] Advanced parameter features
- [ ] UI support (via IPlugView)
- [ ] Preset management

## Testing Strategy

1. **Unit Tests**: Test Go components in isolation
2. **Integration Tests**: Test C/Go boundary
3. **Plugin Validation**: Use VST3 validator tool
4. **Host Testing**: Test in common DAWs (Reaper, Bitwig, etc.)

## Known Challenges

1. **COM-style Interfaces**: VST3 uses COM-style vtables which need careful mapping
2. **Real-time Constraints**: Audio callback must be real-time safe
3. **Memory Management**: Complex due to C/Go boundary
4. **Platform Differences**: Each OS has different requirements
5. **Bundle Format**: VST3 uses specific directory structure

## Next Steps

1. Set up basic project structure
2. Implement minimal C bridge with factory export
3. Create Go bindings for core interfaces
4. Build simple gain plugin as proof of concept
5. Test in VST3 host
6. Iterate and expand functionality

## Resources

- [VST3 SDK Documentation](https://steinbergmedia.github.io/vst3_dev_portal/)
- [VST3 C API Header](./include/vst3/vst3_c_api.h)
- CGO Documentation
- VST3 Validator Tool

## Open Questions

1. Should we embed Go runtime or link dynamically?
2. How to handle platform-specific UI integration?
3. Best approach for preset/state serialization?
4. Performance implications of C/Go boundary in audio callback?

## Success Criteria

- [ ] Can build a working .vst3 plugin
- [ ] Plugin loads in major DAWs
- [ ] Audio processes without glitches
- [ ] Parameters work correctly
- [ ] State persistence works
- [ ] Cross-platform build works