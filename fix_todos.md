# VST3Go TODO Implementation Guide

This document provides detailed instructions for implementing all TODO comments found in the VST3Go codebase. Each TODO represents missing functionality that is critical for a production-ready VST3 plugin framework.

## Overview

The codebase contains **6 critical implementation TODOs** that block core VST3 functionality:

- 🔴 **Critical**: Parameter automation & state management
- 🟡 **Important**: UUID generation & component handler storage
- 🟢 **Low**: Documentation examples

---

## 1. Parameter Change Processing 🔴 CRITICAL

**File:** `pkg/plugin/component.go:211`  
**Priority:** Highest - blocks automation functionality

### Current Code
```go
// Process parameter changes
if processData.inputParameterChanges != nil {
    // TODO: Implement parameter change processing
}
```

### Implementation Plan

#### Step 1: Parse Parameter Changes
```go
// Process parameter changes
if processData.inputParameterChanges != nil {
    paramChanges := (*C.struct_Steinberg_Vst_IParameterChanges)(processData.inputParameterChanges)
    
    paramCount := C.getParameterChangeCount(paramChanges)
    for i := C.int32_t(0); i < paramCount; i++ {
        paramQueue := C.getParameterData(paramChanges, i)
        if paramQueue != nil {
            paramID := C.getParameterId(paramQueue)
            pointCount := C.getPointCount(paramQueue)
            
            // Process all automation points for this parameter
            for j := C.int32_t(0); j < pointCount; j++ {
                var sampleOffset C.int32_t
                var value C.double
                
                if C.getPoint(paramQueue, j, &sampleOffset, &value) == C.kResultOk {
                    // Apply parameter change at specific sample offset
                    c.processCtx.SetParameterAtOffset(uint32(paramID), float64(value), int(sampleOffset))
                }
            }
        }
    }
}
```

#### Step 2: Add Required C Helper Functions
Add to `bridge/bridge.h`:
```c
// Parameter change processing helpers
int32_t getParameterChangeCount(struct Steinberg_Vst_IParameterChanges* changes);
struct Steinberg_Vst_IParamValueQueue* getParameterData(struct Steinberg_Vst_IParameterChanges* changes, int32_t index);
Steinberg_Vst_ParamID getParameterId(struct Steinberg_Vst_IParamValueQueue* queue);
int32_t getPointCount(struct Steinberg_Vst_IParamValueQueue* queue);
Steinberg_tresult getPoint(struct Steinberg_Vst_IParamValueQueue* queue, int32_t index, int32_t* sampleOffset, double* value);
```

#### Step 3: Extend ProcessContext
Add to `pkg/framework/process/context.go`:
```go
// SetParameterAtOffset sets a parameter value at a specific sample offset within the current block
func (c *Context) SetParameterAtOffset(paramID uint32, value float64, sampleOffset int) {
    if param := c.params.Get(paramID); param != nil {
        // For now, apply immediately (sample-accurate automation would require more complex implementation)
        param.SetValue(value)
    }
}
```

### Testing
- Test with host automation (REAPER, Logic Pro, etc.)
- Verify parameter values change during playback
- Check sample-accurate timing for critical parameters

---

## 2. State Management System 🔴 CRITICAL

**Files:** Multiple wrapper functions  
**Priority:** Highest - blocks preset save/load

### Current Status
Five state management functions return `ResultOK` without implementation:
- `GoEditControllerSetComponentState` (wrapper_controller.go:21)
- `GoEditControllerSetState` (wrapper_controller.go:32) 
- `GoEditControllerGetState` (wrapper_controller.go:43)
- `GoComponentSetState` (wrapper.go:337)
- `GoComponentGetState` (wrapper.go:348)

### Implementation Plan

#### Step 1: Implement State Reading
```go
//export GoComponentGetState
func GoComponentGetState(componentPtr unsafe.Pointer, state unsafe.Pointer) C.Steinberg_tresult {
    wrapper := getComponent(uintptr(componentPtr))
    if wrapper == nil {
        return C.Steinberg_tresult(vst3.ResultFalse)
    }

    // Get state from component
    stateData, err := wrapper.component.GetState()
    if err != nil {
        return C.Steinberg_tresult(vst3.ResultFalse)
    }

    // Write to VST3 stream
    streamWrapper := vst3.NewStreamWrapper(state)
    if err := streamWrapper.Write(stateData); err != nil {
        return C.Steinberg_tresult(vst3.ResultFalse)
    }

    return C.Steinberg_tresult(vst3.ResultOK)
}
```

#### Step 2: Implement State Writing
```go
//export GoComponentSetState
func GoComponentSetState(componentPtr unsafe.Pointer, state unsafe.Pointer) C.Steinberg_tresult {
    wrapper := getComponent(uintptr(componentPtr))
    if wrapper == nil {
        return C.Steinberg_tresult(vst3.ResultFalse)
    }

    // Read from VST3 stream
    streamWrapper := vst3.NewStreamWrapper(state)
    stateData, err := streamWrapper.ReadAll()
    if err != nil {
        return C.Steinberg_tresult(vst3.ResultFalse)
    }

    // Apply state to component
    if err := wrapper.component.SetState(stateData); err != nil {
        return C.Steinberg_tresult(vst3.ResultFalse)
    }

    return C.Steinberg_tresult(vst3.ResultOK)
}
```

#### Step 3: Connect to Framework State Manager
Update `pkg/plugin/component.go`:
```go
func (c *componentImpl) GetState() ([]byte, error) {
    if c.processor == nil {
        return nil, fmt.Errorf("no processor available")
    }
    
    // Use the existing state manager
    params := c.processor.GetParameters()
    if params == nil {
        return nil, fmt.Errorf("no parameters available")
    }
    
    stateManager := state.NewManager(params)
    
    var buf bytes.Buffer
    if err := stateManager.Save(&buf); err != nil {
        return nil, err
    }
    
    return buf.Bytes(), nil
}

func (c *componentImpl) SetState(data []byte) error {
    if c.processor == nil {
        return fmt.Errorf("no processor available")
    }
    
    params := c.processor.GetParameters()
    if params == nil {
        return fmt.Errorf("no parameters available")
    }
    
    stateManager := state.NewManager(params)
    
    buf := bytes.NewReader(data)
    return stateManager.Load(buf)
}
```

#### Step 4: Implement Edit Controller State Functions
```go
//export GoEditControllerSetState
func GoEditControllerSetState(componentPtr unsafe.Pointer, state unsafe.Pointer) C.Steinberg_tresult {
    // Edit controller state is typically the same as component state
    return GoComponentSetState(componentPtr, state)
}

//export GoEditControllerGetState  
func GoEditControllerGetState(componentPtr unsafe.Pointer, state unsafe.Pointer) C.Steinberg_tresult {
    // Edit controller state is typically the same as component state
    return GoComponentGetState(componentPtr, state)
}

//export GoEditControllerSetComponentState
func GoEditControllerSetComponentState(componentPtr unsafe.Pointer, state unsafe.Pointer) C.Steinberg_tresult {
    // Component state received from processor - apply to edit controller
    return GoComponentSetState(componentPtr, state)
}
```

### Testing
- Test preset save/load in DAW
- Verify project save/restore preserves plugin settings
- Test with invalid state data (should return error gracefully)

---

## 3. UUID Generation System 🟡 IMPORTANT

**File:** `pkg/framework/plugin/info.go:15`  
**Priority:** Medium - needed for multi-plugin development

### Current Code
```go
// TODO: Implement proper UUID generation from string ID
```

### Implementation Plan

#### Step 1: Deterministic UUID Generation
```go
import (
    "crypto/md5"
    "encoding/binary"
)

// UID converts the string ID to a 16-byte array for VST3
func (i *Info) UID() [16]byte {
    // Generate deterministic UUID from plugin ID string
    // This ensures the same plugin ID always generates the same UUID
    
    hash := md5.Sum([]byte(i.ID))
    
    // Ensure it's a valid UUID v4 format
    // Set version (4) and variant bits
    hash[6] = (hash[6] & 0x0f) | 0x40 // Version 4
    hash[8] = (hash[8] & 0x3f) | 0x80 // Variant 10
    
    return hash
}
```

#### Step 2: Add Validation
```go
// ValidateUID checks if the generated UID is unique
func (i *Info) ValidateUID() error {
    uid := i.UID()
    
    // Check against known UIDs (could be expanded to registry)
    knownUIDs := map[string][16]byte{
        "com.vst3go.examples.gain":   {0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88},
        "com.vst3go.examples.filter": {0x87, 0x65, 0x43, 0x21, 0xFE, 0xDC, 0xBA, 0x98, 0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11},
        "com.vst3go.examples.delay":  {0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11, 0x99, 0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22},
    }
    
    for id, knownUID := range knownUIDs {
        if id != i.ID && uid == knownUID {
            return fmt.Errorf("UID collision detected with plugin %s", id)
        }
    }
    
    return nil
}
```

#### Step 3: Backward Compatibility
Keep hardcoded UIDs for existing examples to maintain compatibility:
```go
func (i *Info) UID() [16]byte {
    // Maintain backward compatibility for existing examples
    switch i.ID {
    case "com.vst3go.examples.gain":
        return [16]byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}
    case "com.vst3go.examples.filter":
        return [16]byte{0x87, 0x65, 0x43, 0x21, 0xFE, 0xDC, 0xBA, 0x98, 0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11}
    case "com.vst3go.examples.delay":
        return [16]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11, 0x99, 0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22}
    default:
        // Generate deterministic UUID for new plugins
        return i.generateDeterministicUID()
    }
}

func (i *Info) generateDeterministicUID() [16]byte {
    hash := md5.Sum([]byte(i.ID))
    hash[6] = (hash[6] & 0x0f) | 0x40 // Version 4
    hash[8] = (hash[8] & 0x3f) | 0x80 // Variant 10
    return hash
}
```

### Testing
- Create multiple test plugins with different IDs
- Verify each generates unique UIDs
- Test UID consistency across restarts
- Validate UUID v4 format compliance

---

## 4. Component Handler Storage 🟡 IMPORTANT

**File:** `pkg/plugin/wrapper_controller.go:199`  
**Priority:** Medium - enables host communication

### Current Code
```go
// TODO: Store component handler for parameter change notifications
```

### Implementation Plan

#### Step 1: Add Handler Storage to Component
Update `pkg/plugin/component.go`:
```go
type componentImpl struct {
    processor     Processor
    processCtx    *process.Context
    sampleRate    float64
    maxBlockSize  int32
    active        bool
    processing    bool
    mu            sync.RWMutex
    
    // Add component handler storage
    componentHandler unsafe.Pointer
    handlerMutex     sync.RWMutex
}
```

#### Step 2: Implement Handler Storage
```go
func (c *componentImpl) SetComponentHandler(handler interface{}) error {
    c.handlerMutex.Lock()
    defer c.handlerMutex.Unlock()
    
    if handler == nil {
        c.componentHandler = nil
        return nil
    }
    
    // Store the handler pointer
    if ptr, ok := handler.(unsafe.Pointer); ok {
        c.componentHandler = ptr
        return nil
    }
    
    return fmt.Errorf("invalid component handler type")
}

func (c *componentImpl) GetComponentHandler() unsafe.Pointer {
    c.handlerMutex.RLock()
    defer c.handlerMutex.RUnlock()
    return c.componentHandler
}
```

#### Step 3: Update Wrapper Function
```go
//export GoEditControllerSetComponentHandler
func GoEditControllerSetComponentHandler(componentPtr unsafe.Pointer, handler unsafe.Pointer) C.Steinberg_tresult {
    wrapper := getComponent(uintptr(componentPtr))
    if wrapper == nil {
        return C.Steinberg_tresult(vst3.ResultFalse)
    }

    // Store component handler for parameter change notifications
    if err := wrapper.component.SetComponentHandler(handler); err != nil {
        return C.Steinberg_tresult(vst3.ResultFalse)
    }
    
    return C.Steinberg_tresult(vst3.ResultOK)
}
```

#### Step 4: Add Parameter Change Notification
```go
// NotifyParameterChanged sends parameter change notification to host
func (c *componentImpl) NotifyParameterChanged(paramID uint32, value float64) {
    c.handlerMutex.RLock()
    handler := c.componentHandler
    c.handlerMutex.RUnlock()
    
    if handler != nil {
        // Call host notification function
        C.notifyParameterChanged(handler, C.uint32_t(paramID), C.double(value))
    }
}
```

#### Step 5: Add C Helper Function
Add to `bridge/bridge.h`:
```c
// Notify host of parameter changes
void notifyParameterChanged(void* componentHandler, uint32_t paramID, double value);
```

Add to `bridge/bridge.c`:
```c
void notifyParameterChanged(void* componentHandler, uint32_t paramID, double value) {
    if (componentHandler) {
        struct Steinberg_Vst_IComponentHandler* handler = 
            (struct Steinberg_Vst_IComponentHandler*)componentHandler;
        
        if (handler && handler->lpVtbl && handler->lpVtbl->performEdit) {
            handler->lpVtbl->performEdit(handler, paramID, value);
        }
    }
}
```

### Testing
- Test parameter changes trigger host updates
- Verify host automation recording works
- Test with different hosts (REAPER, Logic, etc.)

---

## 5. Implementation Priority & Roadmap

### Phase 1: Core Functionality (Immediate)
1. **Parameter Change Processing** - Essential for automation
2. **State Management** - Essential for presets

### Phase 2: Framework Polish (Next Sprint)  
3. **UUID Generation** - Needed for multiple plugins
4. **Component Handler** - Improves host integration

### Phase 3: Quality Assurance
- Comprehensive testing with multiple DAWs
- Performance optimization
- Error handling improvements

---

## 6. Testing Strategy

### Unit Tests
```go
// Add to pkg/plugin/component_test.go
func TestParameterAutomation(t *testing.T) {
    // Test parameter change processing
}

func TestStateManagement(t *testing.T) {
    // Test save/load functionality  
}

func TestUUIDGeneration(t *testing.T) {
    // Test UUID uniqueness and consistency
}
```

### Integration Tests
- Test with VST3 validator
- Test with multiple DAW hosts
- Test automation recording/playback
- Test preset save/load workflows

### Performance Tests
- Measure parameter automation overhead
- Benchmark state save/load times
- Profile memory usage during state operations

---

## 7. Documentation Updates

After implementation, update:
- `README.md` - Remove limitations section about missing features
- `docs/plugin-wrapper.md` - Add state management examples
- `docs/framework-core.md` - Document parameter automation
- Add new documentation for UUID system

---

## 8. Breaking Changes

### UUID Generation
- Existing hardcoded UIDs will be preserved for backward compatibility
- New plugins will use deterministic generation

### State Management  
- No breaking changes - currently returns empty state
- New implementation will return actual plugin state

### Parameter Processing
- No breaking changes - currently ignores automation
- New implementation will apply host automation

---

## Notes

- All TODO implementations should include comprehensive error handling
- Consider thread safety for parameter changes during audio processing
- State management should handle version compatibility
- UUID generation must be deterministic and collision-resistant
- Component handler storage must be thread-safe

---

*This document should be updated as TODOs are implemented and new ones are discovered.*