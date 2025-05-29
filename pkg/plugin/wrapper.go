package plugin

// #cgo CFLAGS: -I../../include
// #include "../../include/vst3/vst3_c_api.h"
// #include "../../bridge/component.h"
// #include <stdlib.h>
// #include <string.h>
import "C"
import (
	"sync"
	"unsafe"
	
	_ "github.com/justyntemme/vst3go/pkg/vst3" // Used in component creation
)

// componentWrapper wraps a Go component for C callbacks
type componentWrapper struct {
	component Component
	handle    unsafe.Pointer
	id        uintptr
}

var (
	// Global map of component wrappers indexed by ID
	components   = make(map[uintptr]*componentWrapper)
	componentsMu sync.RWMutex
	nextID       uintptr = 1
)

// registerComponent registers a component wrapper and returns its ID
func registerComponent(wrapper *componentWrapper) uintptr {
	componentsMu.Lock()
	defer componentsMu.Unlock()
	id := nextID
	nextID++
	wrapper.id = id
	components[id] = wrapper
	return id
}

// unregisterComponent removes a component wrapper by ID
func unregisterComponent(id uintptr) {
	componentsMu.Lock()
	defer componentsMu.Unlock()
	delete(components, id)
}

// getComponent retrieves a component wrapper by ID
func getComponent(id uintptr) *componentWrapper {
	componentsMu.RLock()
	defer componentsMu.RUnlock()
	return components[id]
}

//export GoCreateInstance
func GoCreateInstance(cid, iid *C.char) unsafe.Pointer {
	if globalPlugin == nil {
		return nil
	}
	
	// Check if the class ID matches our plugin
	var requestedCID [16]byte
	C.memcpy(unsafe.Pointer(&requestedCID[0]), unsafe.Pointer(cid), 16)
	
	pluginUID := globalPlugin.GetUID()
	if requestedCID != pluginUID {
		return nil
	}
	
	// Check requested interface - for now accept any interface
	// The component's QueryInterface will handle specific interface requests
	// var requestedIID [16]byte
	// C.memcpy(unsafe.Pointer(&requestedIID[0]), unsafe.Pointer(iid), 16)
	
	// Create component instance
	component := globalPlugin.CreateComponent()
	if component == nil {
		return nil
	}
	
	// Create wrapper
	wrapper := &componentWrapper{
		component: component,
	}
	
	// Register and get ID
	id := registerComponent(wrapper)
	
	// Create C component with ID instead of Go pointer
	cComponent := C.createComponent(unsafe.Pointer(id))
	if cComponent == nil {
		unregisterComponent(id)
		return nil
	}
	
	wrapper.handle = cComponent
	
	return cComponent
}

//export GoReleaseComponent
func GoReleaseComponent(componentPtr unsafe.Pointer) {
	id := uintptr(componentPtr)
	unregisterComponent(id)
}

// IComponent callbacks
//export GoComponentInitialize
func GoComponentInitialize(componentPtr unsafe.Pointer, context unsafe.Pointer) C.Steinberg_tresult {
	id := uintptr(componentPtr)
	wrapper := getComponent(id)
	if wrapper == nil {
		return C.Steinberg_tresult(2)
	}
	
	err := wrapper.component.Initialize(context)
	if err != nil {
		return C.Steinberg_tresult(1)
	}
	return C.Steinberg_tresult(0)
}

//export GoComponentTerminate
func GoComponentTerminate(componentPtr unsafe.Pointer) C.Steinberg_tresult {
	id := uintptr(componentPtr)
	wrapper := getComponent(id)
	if wrapper == nil {
		return C.Steinberg_tresult(2)
	}
	
	err := wrapper.component.Terminate()
	if err != nil {
		return C.Steinberg_tresult(1)
	}
	return C.Steinberg_tresult(0)
}

//export GoComponentGetControllerClassId
func GoComponentGetControllerClassId(componentPtr unsafe.Pointer, classId *C.char) {
	id := uintptr(componentPtr)
	wrapper := getComponent(id)
	if wrapper == nil {
		return
	}
	
	uid := wrapper.component.GetControllerClassID()
	C.memcpy(unsafe.Pointer(classId), unsafe.Pointer(&uid[0]), 16)
}

//export GoComponentSetIoMode
func GoComponentSetIoMode(componentPtr unsafe.Pointer, mode C.int32_t) C.Steinberg_tresult {
	id := uintptr(componentPtr)
	wrapper := getComponent(id)
	if wrapper == nil {
		return C.Steinberg_tresult(2)
	}
	
	err := wrapper.component.SetIOMode(int32(mode))
	if err != nil {
		return C.Steinberg_tresult(1)
	}
	return C.Steinberg_tresult(0)
}

//export GoComponentGetBusCount
func GoComponentGetBusCount(componentPtr unsafe.Pointer, mediaType, dir C.int32_t) C.int32_t {
	id := uintptr(componentPtr)
	wrapper := getComponent(id)
	if wrapper == nil {
		return 0
	}
	
	return C.int32_t(wrapper.component.GetBusCount(int32(mediaType), int32(dir)))
}

//export GoComponentGetBusInfo
func GoComponentGetBusInfo(componentPtr unsafe.Pointer, mediaType, dir, index C.int32_t, bus unsafe.Pointer) C.Steinberg_tresult {
	id := uintptr(componentPtr)
	wrapper := getComponent(id)
	if wrapper == nil {
		return C.Steinberg_tresult(2)
	}
	
	info, err := wrapper.component.GetBusInfo(int32(mediaType), int32(dir), int32(index))
	if err != nil || info == nil {
		return C.Steinberg_tresult(1)
	}
	
	// Copy bus info to C struct
	cBus := (*C.struct_Steinberg_Vst_BusInfo)(bus)
	cBus.mediaType = C.Steinberg_Vst_MediaType(info.MediaType)
	cBus.direction = C.Steinberg_Vst_BusDirection(info.Direction)
	cBus.channelCount = C.Steinberg_int32(info.ChannelCount)
	
	// Copy name
	nameBytes := []byte(info.Name)
	if len(nameBytes) > 127 {
		nameBytes = nameBytes[:127]
	}
	for i, b := range nameBytes {
		cBus.name[i] = C.Steinberg_char16(b)
	}
	cBus.name[len(nameBytes)] = 0
	
	cBus.busType = C.Steinberg_Vst_BusType(info.BusType)
	cBus.flags = C.Steinberg_uint32(info.Flags)
	
	return C.Steinberg_tresult(0)
}

//export GoComponentActivateBus
func GoComponentActivateBus(componentPtr unsafe.Pointer, mediaType, dir, index, state C.int32_t) C.Steinberg_tresult {
	id := uintptr(componentPtr)
	wrapper := getComponent(id)
	if wrapper == nil {
		return C.Steinberg_tresult(2)
	}
	
	err := wrapper.component.ActivateBus(int32(mediaType), int32(dir), int32(index), state != 0)
	if err != nil {
		return C.Steinberg_tresult(1)
	}
	return C.Steinberg_tresult(0)
}

//export GoComponentSetActive
func GoComponentSetActive(componentPtr unsafe.Pointer, state C.int32_t) C.Steinberg_tresult {
	id := uintptr(componentPtr)
	wrapper := getComponent(id)
	if wrapper == nil {
		return C.Steinberg_tresult(2)
	}
	
	err := wrapper.component.SetActive(state != 0)
	if err != nil {
		return C.Steinberg_tresult(1)
	}
	return C.Steinberg_tresult(0)
}