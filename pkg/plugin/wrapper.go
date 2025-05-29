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
	
	"github.com/justyntemme/vst3go/pkg/vst3"
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

//export GoComponentSetState
func GoComponentSetState(componentPtr unsafe.Pointer, state unsafe.Pointer) C.Steinberg_tresult {
	id := uintptr(componentPtr)
	wrapper := getComponent(id)
	if wrapper == nil {
		return C.Steinberg_tresult(2)
	}
	
	stream := vst3.NewStreamWrapper(state)
	if stream == nil {
		return C.Steinberg_tresult(2)
	}
	
	// Read state data
	stateData, err := readStateFromStream(stream, wrapper.component)
	if err != nil {
		return C.Steinberg_tresult(1)
	}
	
	err = wrapper.component.SetState(stateData)
	if err != nil {
		return C.Steinberg_tresult(1)
	}
	return C.Steinberg_tresult(0)
}

//export GoComponentGetState
func GoComponentGetState(componentPtr unsafe.Pointer, state unsafe.Pointer) C.Steinberg_tresult {
	id := uintptr(componentPtr)
	wrapper := getComponent(id)
	if wrapper == nil {
		return C.Steinberg_tresult(2)
	}
	
	stream := vst3.NewStreamWrapper(state)
	if stream == nil {
		return C.Steinberg_tresult(2)
	}
	
	// Get state data
	stateData, err := wrapper.component.GetState()
	if err != nil {
		return C.Steinberg_tresult(1)
	}
	
	// Write state to stream
	err = writeStateToStream(stream, stateData, wrapper.component)
	if err != nil {
		return C.Steinberg_tresult(1)
	}
	return C.Steinberg_tresult(0)
}

// Helper function to write state to stream
func writeStateToStream(stream *vst3.StreamWrapper, stateData []byte, component Component) error {
	// Write a simple header/version
	if err := stream.WriteString("VST3GO_STATE_V1"); err != nil {
		return err
	}
	
	// Write parameter count
	paramCount := component.GetParameterCount()
	if err := stream.WriteInt32(paramCount); err != nil {
		return err
	}
	
	// Write each parameter value
	for i := int32(0); i < paramCount; i++ {
		info, err := component.GetParameterInfo(i)
		if err != nil {
			continue
		}
		
		// Write parameter ID and value
		if err := stream.WriteInt32(int32(info.ID)); err != nil {
			return err
		}
		
		value := component.GetParamNormalized(uint32(info.ID))
		if err := stream.WriteFloat64(value); err != nil {
			return err
		}
	}
	
	// Write any custom state data
	if len(stateData) > 0 {
		if err := stream.WriteInt32(int32(len(stateData))); err != nil {
			return err
		}
		_, err := stream.Write(stateData)
		return err
	} else {
		// No custom data
		return stream.WriteInt32(0)
	}
}

// Helper function to read state from stream
func readStateFromStream(stream *vst3.StreamWrapper, component Component) ([]byte, error) {
	// Read and verify header
	header, err := stream.ReadString()
	if err != nil {
		return nil, err
	}
	if header != "VST3GO_STATE_V1" {
		return nil, vst3.ErrNotImplemented
	}
	
	// Read parameter count
	paramCount, err := stream.ReadInt32()
	if err != nil {
		return nil, err
	}
	
	// Read each parameter value
	for i := int32(0); i < paramCount; i++ {
		paramID, err := stream.ReadInt32()
		if err != nil {
			return nil, err
		}
		
		value, err := stream.ReadFloat64()
		if err != nil {
			return nil, err
		}
		
		// Set parameter value
		component.SetParamNormalized(uint32(paramID), value)
	}
	
	// Read custom state data length
	customDataLen, err := stream.ReadInt32()
	if err != nil {
		return nil, err
	}
	
	if customDataLen > 0 {
		// Read custom data
		customData := make([]byte, customDataLen)
		n, err := stream.Read(customData)
		if err != nil {
			return nil, err
		}
		if n != customDataLen {
			return nil, vst3.ErrNotImplemented
		}
		return customData, nil
	}
	
	return nil, nil
}