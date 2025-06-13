# Windows Support for VST3Go

## Overview

This document outlines the requirements and implementation plan for adding Windows support to VST3Go. Currently, the framework only supports Linux and macOS. Windows support requires changes primarily in the build system and C bridge layer, with minimal modifications to the Go code.

## Current Status

- **Supported Platforms**: Linux, macOS (Darwin)
- **Build System**: GNU Make with platform detection
- **Library Format**: `.so` (Linux), `.dylib` (macOS)
- **Entry Points**: Unix-style `ModuleEntry`/`ModuleExit`

## Windows VST3 Requirements

### 1. Library Format
- Windows VST3 plugins are DLLs with a `.vst3` extension
- Must export `GetPluginFactory` function
- Requires proper DLL entry point (`DllMain`)

### 2. Bundle Structure
Windows VST3 bundles differ from Unix platforms:
```
MyPlugin.vst3/
├── Contents/
│   ├── x86_64-win/
│   │   └── MyPlugin.vst3  (actually a DLL)
│   └── Resources/
│       └── plugin.uidesc
```

### 3. Installation Locations
- **System-wide**: `C:\Program Files\Common Files\VST3\`
- **User-specific**: `%LOCALAPPDATA%\Programs\Common\VST3\`

## Implementation Plan

### Phase 1: Build System Updates

#### 1.1 Makefile Platform Detection
```makefile
# Detect Windows
ifeq ($(OS),Windows_NT)
    PLATFORM := windows
    SO_EXT := dll
    VST3_ARCH := x86_64-win
    
    # Windows-specific flags
    CFLAGS_BASE := -I./include -g -O0 -DDEBUG_VST3GO -D_WIN32
    LDFLAGS_BASE := -shared -Wl,--export-all-symbols
    
    # Installation paths
    VST3_SYSTEM_DIR := C:/Program Files/Common Files/VST3
    VST3_USER_DIR := $(APPDATA)/VST3
endif
```

#### 1.2 Bundle Creation
Update the bundle creation logic to handle Windows structure:
```makefile
bundle-windows:
	@mkdir -p $(BUNDLE_DIR)/Contents/x86_64-win
	@cp $(TARGET) $(BUNDLE_DIR)/Contents/x86_64-win/$(NAME).vst3
	@echo "Created Windows VST3 bundle: $(BUNDLE_DIR)"
```

### Phase 2: C Bridge Modifications

#### 2.1 Windows Entry Points
Add to `bridge/bridge.c`:
```c
#ifdef _WIN32
#include <windows.h>

// Windows DLL entry point
BOOL WINAPI DllMain(HINSTANCE hinstDLL, DWORD fdwReason, LPVOID lpvReserved) {
    switch (fdwReason) {
        case DLL_PROCESS_ATTACH:
            // Initialize module
            if (!moduleInitialized) {
                initModule();
                moduleInitialized = 1;
            }
            break;
            
        case DLL_PROCESS_DETACH:
            // Cleanup module
            if (moduleInitialized) {
                deinitModule();
                moduleInitialized = 0;
            }
            break;
            
        case DLL_THREAD_ATTACH:
        case DLL_THREAD_DETACH:
            // Thread-specific initialization/cleanup if needed
            break;
    }
    return TRUE;
}

// Export the factory function
__declspec(dllexport) struct Steinberg_IPluginFactory* GetPluginFactory(void) {
    static struct Steinberg_IPluginFactory* factory = NULL;
    if (!factory) {
        factory = createPluginFactory();
    }
    return factory;
}

// Windows-specific module initialization
static void initModule(void) {
    // Any Windows-specific initialization
}

static void deinitModule(void) {
    // Any Windows-specific cleanup
}

#else
// Existing Unix entry points
bool ModuleEntry(void* sharedLibraryHandle) { ... }
bool ModuleExit(void) { ... }
#endif
```

#### 2.2 Export Definitions
Create `bridge/exports.def` for explicit symbol exports:
```
EXPORTS
    GetPluginFactory
    DllMain
```

### Phase 3: CGO Configuration

#### 3.1 Update CGO Directives
In `pkg/vst3/bridge.go` and other relevant files:
```go
// #cgo windows CFLAGS: -I../../include -D_WIN32 -DWINDOWS
// #cgo windows LDFLAGS: -shared -Wl,--export-all-symbols -static-libgcc -static-libstdc++
```

#### 3.2 Platform-Specific Code
Add platform build tags where needed:
```go
// +build windows

package vst3

// Windows-specific implementation if needed
```

### Phase 4: Path Handling

#### 4.1 Cross-Platform Paths
Update any hardcoded paths to use `filepath` package:
```go
import "path/filepath"

// Instead of:
path := "some/unix/path"

// Use:
path := filepath.Join("some", "unix", "path")
```

#### 4.2 Installation Scripts
Create Windows batch scripts or PowerShell equivalents:
```batch
@echo off
REM install.bat - Windows installation script

set VST3_DIR=%PROGRAMFILES%\Common Files\VST3

if not exist "%VST3_DIR%" mkdir "%VST3_DIR%"
xcopy /E /I "%1" "%VST3_DIR%\%~n1.vst3"

echo Installed %~n1.vst3 to %VST3_DIR%
```

### Phase 5: Testing and Validation

#### 5.1 Validator Support
Create `scripts/test_validator.bat`:
```batch
@echo off
REM Windows VST3 validator script

set VALIDATOR="C:\Program Files\Steinberg\VST3 SDK\bin\validator.exe"

if not exist %VALIDATOR% (
    echo VST3 validator not found at %VALIDATOR%
    exit /b 1
)

%VALIDATOR% "%~1"
```

#### 5.2 Cross-Compilation Testing
Document cross-compilation from Linux:
```bash
# Install MinGW-w64
sudo apt-get install mingw-w64

# Cross-compile for Windows
make windows CC=x86_64-w64-mingw32-gcc
```

### Phase 6: Compiler Support

#### 6.1 MinGW-w64
Recommended for initial Windows support:
- Free and open source
- Good CGO compatibility
- Can cross-compile from Linux

#### 6.2 MSVC (Future)
For better Windows integration:
- Better debugging support
- Native Windows toolchain
- Requires more complex build configuration

## Known Challenges

### 1. Go Shared Libraries on Windows
- Go's `-buildmode=c-shared` has limitations on Windows
- May need to statically link Go runtime

### 2. Real-time Performance
- Windows thread scheduling differs from Linux
- May need to adjust thread priorities
- MMCSS (Multimedia Class Scheduler Service) integration

### 3. File Locking
- Windows locks files more aggressively
- May affect plugin reloading during development

### 4. Path Length Limitations
- Windows has 260-character path limit (unless long paths enabled)
- VST3 bundle paths can get long

## Testing Strategy

### 1. Build Verification
- [ ] Successful compilation with MinGW-w64
- [ ] Successful compilation with MSVC (future)
- [ ] Proper DLL exports verified with `dumpbin /exports`

### 2. VST3 Compliance
- [ ] Passes Steinberg VST3 validator
- [ ] Proper bundle structure
- [ ] Correct installation locations

### 3. DAW Testing
Priority DAWs for Windows testing:
1. **Cubase** (Steinberg reference implementation)
2. **Reaper** (Popular and thorough VST3 implementation)
3. **FL Studio** (Different threading model)
4. **Ableton Live** (Widely used)

### 4. Performance Testing
- [ ] No audio dropouts at low latencies
- [ ] CPU usage comparable to Linux/macOS
- [ ] Memory usage stable

## Migration Guide for Plugin Developers

### 1. Code Changes
Most plugins require no code changes. Ensure:
- No hardcoded paths
- No Unix-specific system calls
- Proper path separator usage

### 2. Build Process
```bash
# Linux/macOS
make

# Windows (native)
make windows

# Windows (cross-compile from Linux)
make windows CC=x86_64-w64-mingw32-gcc
```

### 3. Installation
```bash
# Linux/macOS
make install

# Windows
make install-windows
# or manually copy to %PROGRAMFILES%\Common Files\VST3\
```

## Timeline

### Phase 1: Basic Support (2 weeks)
- Makefile updates
- C bridge modifications
- Basic build support

### Phase 2: Full Integration (1 week)
- Path handling fixes
- Installation scripts
- Documentation updates

### Phase 3: Testing and Polish (1 week)
- Validator testing
- DAW compatibility testing
- Performance optimization

### Total: ~1 month for production-ready Windows support

## Conclusion

Adding Windows support to VST3Go is achievable with minimal architectural changes. The main work involves:
1. Updating the build system
2. Adding Windows-specific entry points
3. Handling platform differences in paths and installation

The existing codebase's clean architecture makes this a straightforward platform port rather than a major rewrite. Once implemented, VST3Go will support all major desktop platforms, making it a truly comprehensive framework for VST3 development in Go.