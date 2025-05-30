# C Bridge Layer

## Overview

The C Bridge is the thin translation layer between the VST3 C API and Go. It follows the principle of "Just a Bridge" - no business logic, no state management, just direct function routing.

## Design Philosophy

### What the C Bridge DOES:
- Implements VST3 C API entry points
- Routes function calls to Go
- Manages handle-based object references
- Provides vtable structures for VST3 interfaces

### What the C Bridge DOES NOT:
- Store plugin state
- Implement business logic
- Manage parameters
- Handle audio processing
- Make architectural decisions

## Components

### bridge.c/h - Factory and Entry Point

#### GetPluginFactory
The single entry point for VST3 hosts:
```c
SMTG_EXPORT IPluginFactory* PLUGIN_API GetPluginFactory() {
    return &pluginFactory;
}
```

#### Factory Implementation
```c
static struct FactoryVtbl {
    // IUnknown methods
    queryInterface_func queryInterface;
    addRef_func addRef;
    release_func release;
    
    // IPluginFactory methods
    getFactoryInfo_func getFactoryInfo;
    countClasses_func countClasses;
    getClassInfo_func getClassInfo;
    createInstance_func createInstance;
} factoryVtbl = {
    // Direct routing to Go functions
    factory_queryInterface,
    factory_addRef,
    factory_release,
    factory_getFactoryInfo,
    factory_countClasses,
    factory_getClassInfo,
    factory_createInstance
};
```

### component.c/h - Component Interface Routing

#### Component Structure
```c
typedef struct ComponentWrapper {
    void* component;        // IComponent vtable
    void* audioProcessor;   // IAudioProcessor vtable  
    void* editController;   // IEditController vtable
    uintptr_t goHandle;     // Handle to Go object
} ComponentWrapper;
```

#### Interface Routing
Each VST3 interface method is directly routed to Go:
```c
tresult component_initialize(void* self, FUnknown* context) {
    ComponentWrapper* wrapper = (ComponentWrapper*)self;
    return GoComponent_Initialize(wrapper->goHandle, context);
}
```

## Handle Management

### Why Handles?
CGO has restrictions on passing Go pointers that contain pointers. Using integer handles avoids these restrictions:

```c
// C side uses handles
uintptr_t goHandle;

// Go side maintains registry
var componentRegistry = sync.Map{}

func RegisterComponent(c *Component) uintptr {
    handle := atomic.AddUintptr(&nextHandle, 1)
    componentRegistry.Store(handle, c)
    return handle
}
```

### Lifecycle
1. Go creates component, gets handle
2. C stores handle in wrapper
3. All C→Go calls use handle
4. Go looks up component from handle
5. Component destruction removes from registry

## Vtable Organization

### Correct Interface Order
VST3 requires specific vtable ordering:
```c
struct ComponentVtbl {
    // IUnknown (must be first)
    queryInterface_func queryInterface;
    addRef_func addRef;
    release_func release;
    
    // IPluginBase (must be second)
    initialize_func initialize;
    terminate_func terminate;
    
    // IComponent (specific to this interface)
    getControllerClassId_func getControllerClassId;
    setIoMode_func setIoMode;
    getBusCount_func getBusCount;
    // ... etc
};
```

## Function Calling Convention

### C to Go Calls
```c
// C function receives VST3 call
tresult component_process(void* self, ProcessData* data) {
    ComponentWrapper* wrapper = (ComponentWrapper*)self;
    // Direct call to Go with handle
    return GoComponent_Process(wrapper->goHandle, data);
}
```

### Go Export Functions
```go
//export GoComponent_Process
func GoComponent_Process(handle C.uintptr_t, data *C.ProcessData) C.tresult {
    // Look up component from handle
    if comp, ok := componentRegistry.Load(uintptr(handle)); ok {
        return comp.(*Component).Process(data)
    }
    return C.kResultFalse
}
```

## Memory Safety

### String Handling
C strings must be properly converted:
```c
// Factory info example
void factory_getFactoryInfo(void* self, PFactoryInfo* info) {
    // Go returns static strings, no allocation
    GoGetFactoryInfo((GoFactoryInfo*)info);
}
```

### Pointer Restrictions
- No Go pointers in C structs
- Use handles for object references
- Copy data instead of sharing pointers
- Static strings for factory info

## Platform Considerations

### Windows
```c
#ifdef _WIN32
    #define PLUGIN_API __stdcall
    // Windows-specific adjustments
#endif
```

### macOS
```c
#ifdef __APPLE__
    // macOS bundle handling
    // Objective-C bridging if needed
#endif
```

### Linux
```c
#ifdef __linux__
    // Linux-specific code
    // .so library handling
#endif
```

## Error Handling

The C bridge uses simple error passing:
```c
tresult result = GoComponent_Initialize(handle, context);
if (result != kResultOk) {
    // Return error to host
    return result;
}
```

No error recovery or business logic in C layer.

## Debugging Support

### Debug Builds
```c
#ifdef DEBUG_VST3GO
    #define DEBUG_LOG(msg) fprintf(stderr, "[C Bridge] %s\n", msg)
#else
    #define DEBUG_LOG(msg)
#endif
```

### Function Tracing
```c
tresult component_initialize(void* self, FUnknown* context) {
    DEBUG_LOG("component_initialize called");
    // ... implementation
}
```

## Best Practices

1. **Keep It Simple**: No complex logic in C
2. **Direct Mapping**: One C function → One Go function
3. **No State**: All state lives in Go
4. **Safe Defaults**: Return safe values on error
5. **Platform Agnostic**: Use preprocessor for platform code

## Common Pitfalls to Avoid

1. **Don't Store Go Pointers**: Use handles instead
2. **Don't Allocate in C**: Let Go manage memory
3. **Don't Add Business Logic**: Keep it in Go
4. **Don't Optimize Here**: C bridge is not the bottleneck
5. **Don't Break ABI**: Maintain vtable compatibility

## Future Considerations

The C bridge is designed to remain stable even as the Go framework evolves:
- New interfaces can be added without changing existing code
- Platform-specific code is isolated
- Handle system scales to any number of objects
- Debug features can be toggled at compile time