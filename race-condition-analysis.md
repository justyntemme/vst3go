# VST3 Plugin Initialization Race Conditions Analysis

## Potential Race Conditions Identified

### 1. Global Plugin Not Initialized When GoCreateInstance Called

**Location**: `pkg/plugin/wrapper.go`

The `globalPlugin` variable is set by `Register()` which is called in the `init()` function of the example plugins. However, there's a potential race condition:

```go
// In wrapper.go:
var globalPlugin Plugin  // Not initialized

//export GoCreateInstance
func GoCreateInstance(cid *C.char, iid *C.char) unsafe.Pointer {
    if globalPlugin == nil {  // This check exists but might not be sufficient
        return nil
    }
    // ...
}
```

**Issue**: The host may call `GetPluginFactory()` and subsequently `GoCreateInstance()` before the Go runtime has fully initialized and executed all `init()` functions.

### 2. C Library Entry Points vs Go Runtime Initialization

**Location**: `bridge/bridge.c`

The VST3 host calls these entry points in order:
1. `ModuleEntry()` - Module initialization
2. `GetPluginFactory()` - Get the plugin factory
3. `factory_createInstance()` -> `GoCreateInstance()` - Create plugin instance

**Issue**: The Go runtime initialization (including `init()` functions) happens when the shared library is loaded, but there's no explicit synchronization to ensure it completes before `GoCreateInstance` is called.

### 3. Component Registration Map Access

**Location**: `pkg/plugin/wrapper.go`

```go
var (
    components   = make(map[uintptr]*componentWrapper)
    componentsMu sync.RWMutex
    nextID       uintptr = 1
)
```

While the map access is protected by mutex, the map is initialized at package level. If `GoCreateInstance` is called before package initialization completes, this could cause issues.

### 4. BufferedProcessor Initialization

**Location**: `pkg/plugin/wrapper.go`, lines 299-306

```go
if globalConfig.EnableBuffering {
    channels := globalConfig.BufferChannels
    if channels <= 0 {
        channels = 2 // Default to stereo
    }
    processor = NewBufferedProcessor(processor, channels)
}
```

The `globalConfig` is initialized at package level but could theoretically be accessed before initialization.

## Recommended Fixes

### 1. Add Initialization Guard

Add an initialization flag and check:

```go
var (
    initialized     bool
    initMu          sync.Mutex
    globalPlugin    Plugin
)

//export GoCreateInstance
func GoCreateInstance(cid *C.char, iid *C.char) unsafe.Pointer {
    initMu.Lock()
    if !initialized {
        initMu.Unlock()
        return nil
    }
    initMu.Unlock()
    
    if globalPlugin == nil {
        return nil
    }
    // ... rest of function
}

func Register(p Plugin) {
    initMu.Lock()
    defer initMu.Unlock()
    globalPlugin = p
    initialized = true
}
```

### 2. Add Explicit Initialization in ModuleEntry

Modify `bridge.c` to ensure Go runtime is ready:

```c
// Add Go function declaration
extern int GoIsInitialized();

__attribute__((visibility("default")))
int ModuleEntry(void* sharedLibraryHandle) {
    if (moduleInitialized) {
        return 1;
    }
    
    // Wait for Go runtime initialization
    int retries = 100; // 1 second total
    while (retries > 0 && !GoIsInitialized()) {
        usleep(10000); // 10ms
        retries--;
    }
    
    if (!GoIsInitialized()) {
        return 0; // Initialization failed
    }
    
    moduleInitialized = 1;
    return 1;
}
```

### 3. Use sync.Once for Global Initialization

Replace direct initialization with sync.Once:

```go
var (
    globalPlugin Plugin
    globalOnce   sync.Once
    globalErr    error
)

func ensureInitialized() error {
    globalOnce.Do(func() {
        // Any initialization that needs to happen once
        if globalPlugin == nil {
            globalErr = errors.New("plugin not registered")
        }
    })
    return globalErr
}

//export GoCreateInstance
func GoCreateInstance(cid *C.char, iid *C.char) unsafe.Pointer {
    if err := ensureInitialized(); err != nil {
        return nil
    }
    // ... rest of function
}
```

### 4. Add Component Wrapper Validation

Add nil checks and validation in component wrapper operations:

```go
func getComponent(id uintptr) *componentWrapper {
    componentsMu.RLock()
    defer componentsMu.RUnlock()
    
    if id == 0 {
        return nil
    }
    
    // Ensure map is initialized
    if components == nil {
        return nil
    }
    
    wrapper, exists := components[id]
    if !exists {
        return nil
    }
    
    return wrapper
}
```

### 5. Factory Method Defensive Checks

Add defensive checks in all exported Go functions:

```go
//export GoCountClasses
func GoCountClasses() C.int32_t {
    if globalPlugin == nil {
        DBG_LOG("GoCountClasses: globalPlugin is nil")
        return 0
    }
    return 1
}

//export GoGetClassInfo
func GoGetClassInfo(index C.int32_t, cid *C.char, cardinality *C.int32_t, category, name *C.char) {
    if globalPlugin == nil || index != 0 {
        DBG_LOG("GoGetClassInfo: globalPlugin is nil or invalid index")
        return
    }
    // ... rest of function
}
```

## Testing Recommendations

1. **Stress Test**: Create a test that rapidly loads/unloads the plugin
2. **Early Call Test**: Modify the host to call factory methods immediately after dlopen
3. **Concurrent Load Test**: Load multiple instances of the plugin simultaneously
4. **Memory Barrier Test**: Use tools like ThreadSanitizer to detect race conditions

## Conclusion

The main race condition risk is that the VST3 host may call plugin factory methods before the Go runtime has fully initialized, particularly before `init()` functions have run. The recommended solution is to add explicit initialization checks and synchronization to ensure the plugin is fully ready before accepting any calls from the host.