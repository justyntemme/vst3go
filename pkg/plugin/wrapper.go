package plugin

// #cgo CFLAGS: -I../../include
// #include "../../include/vst3/vst3_c_api.h"
// #include "../../bridge/bridge.h"
// #include "../../bridge/component.h"
// #include <stdlib.h>
// #include <string.h>
//
// // Helper functions to call IComponentHandler methods
// static inline Steinberg_tresult componentHandler_beginEdit(struct Steinberg_Vst_IComponentHandler* handler, Steinberg_Vst_ParamID id) {
//     if (handler && handler->lpVtbl && handler->lpVtbl->beginEdit) {
//         return handler->lpVtbl->beginEdit(handler, id);
//     }
//     return Steinberg_kResultFalse;
// }
//
// static inline Steinberg_tresult componentHandler_performEdit(struct Steinberg_Vst_IComponentHandler* handler, Steinberg_Vst_ParamID id, Steinberg_Vst_ParamValue value) {
//     if (handler && handler->lpVtbl && handler->lpVtbl->performEdit) {
//         return handler->lpVtbl->performEdit(handler, id, value);
//     }
//     return Steinberg_kResultFalse;
// }
//
// static inline Steinberg_tresult componentHandler_endEdit(struct Steinberg_Vst_IComponentHandler* handler, Steinberg_Vst_ParamID id) {
//     if (handler && handler->lpVtbl && handler->lpVtbl->endEdit) {
//         return handler->lpVtbl->endEdit(handler, id);
//     }
//     return Steinberg_kResultFalse;
// }
import "C"
import (
	"sync"
	"unsafe"

	"github.com/justyntemme/vst3go/pkg/vst3"
)

// Component interface that our componentImpl satisfies
type Component interface {
	vst3.IComponent
	vst3.IAudioProcessor
	vst3.IEditController
}

// componentWrapper wraps a Go component for C callbacks
type componentWrapper struct {
	component        Component
	handle           unsafe.Pointer
	id               uintptr
	componentHandler unsafe.Pointer // IComponentHandler from host
	handlerMu        sync.RWMutex   // Protects componentHandler access
}

var (
	// Global map of component wrappers indexed by ID
	components   = make(map[uintptr]*componentWrapper)
	componentsMu sync.RWMutex
	nextID       uintptr = 1
)

// Global plugin instance
var globalPlugin Plugin

// Factory info
type FactoryInfo struct {
	Vendor string
	URL    string
	Email  string
}

var globalFactoryInfo = FactoryInfo{
	Vendor: "VST3Go",
	URL:    "https://vst3go.dev",
	Email:  "info@vst3go.dev",
}

// Configuration for plugin behavior
type Config struct {
	// EnableBuffering wraps processors with BufferedProcessor for GC protection
	// This adds 50ms of latency but protects against audio glitches during GC pauses
	EnableBuffering bool
	
	// BufferChannels specifies the number of channels to buffer (if EnableBuffering is true)
	// Defaults to 2 (stereo) if not specified
	BufferChannels int
}

var globalConfig = Config{
	EnableBuffering: true, // Enabled by default for GC protection
	BufferChannels: 2,
}

// Register sets the global plugin instance
func Register(p Plugin) {
	globalPlugin = p
}

// SetFactoryInfo sets the factory information
func SetFactoryInfo(info FactoryInfo) {
	globalFactoryInfo = info
}

// SetConfig sets the global plugin configuration
func SetConfig(cfg Config) {
	globalConfig = cfg
}

// recoverPanic is a helper to recover from panics in callbacks
func recoverPanic(operation string) {
	if r := recover(); r != nil {
		// Log the panic but don't propagate it to C code
		_ = r
	}
}

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

	if id == 0 {
		return nil
	}

	wrapper, exists := components[id]
	if !exists {
		return nil
	}

	return wrapper
}

// notifyParamBeginEdit notifies the host that parameter editing is beginning
func (w *componentWrapper) notifyParamBeginEdit(paramID uint32) {
	w.handlerMu.RLock()
	handler := w.componentHandler
	w.handlerMu.RUnlock()

	if handler == nil {
		return
	}

	// Call beginEdit through helper function
	C.componentHandler_beginEdit((*C.Steinberg_Vst_IComponentHandler)(handler), C.Steinberg_Vst_ParamID(paramID))
}

// notifyParamPerformEdit notifies the host of a parameter value change
func (w *componentWrapper) notifyParamPerformEdit(paramID uint32, valueNormalized float64) {
	w.handlerMu.RLock()
	handler := w.componentHandler
	w.handlerMu.RUnlock()

	if handler == nil {
		return
	}

	// Call performEdit through helper function
	C.componentHandler_performEdit((*C.Steinberg_Vst_IComponentHandler)(handler), C.Steinberg_Vst_ParamID(paramID), C.Steinberg_Vst_ParamValue(valueNormalized))
}

// notifyParamEndEdit notifies the host that parameter editing has ended
func (w *componentWrapper) notifyParamEndEdit(paramID uint32) {
	w.handlerMu.RLock()
	handler := w.componentHandler
	w.handlerMu.RUnlock()

	if handler == nil {
		return
	}

	// Call endEdit through helper function
	C.componentHandler_endEdit((*C.Steinberg_Vst_IComponentHandler)(handler), C.Steinberg_Vst_ParamID(paramID))
}

//export GoGetFactoryInfo
func GoGetFactoryInfo(vendor, url, email *C.char, flags *C.int32_t) {
	C.strcpy(vendor, C.CString(globalFactoryInfo.Vendor))
	C.strcpy(url, C.CString(globalFactoryInfo.URL))
	C.strcpy(email, C.CString(globalFactoryInfo.Email))
	*flags = C.Steinberg_PFactoryInfo_FactoryFlags_kUnicode
}

//export GoCountClasses
func GoCountClasses() C.int32_t {
	if globalPlugin == nil {
		return 0
	}
	return 1
}

//export GoGetClassInfo
func GoGetClassInfo(index C.int32_t, cid *C.char, cardinality *C.int32_t, category, name *C.char) {
	if globalPlugin == nil || index != 0 {
		return
	}

	info := globalPlugin.GetInfo()

	// Copy UID
	uid := info.UID()
	C.memcpy(unsafe.Pointer(cid), unsafe.Pointer(&uid[0]), 16)

	// Set cardinality
	*cardinality = C.Steinberg_PClassInfo_ClassCardinality_kManyInstances

	// Set category and name
	C.strcpy(category, C.CString("Audio Module Class"))
	C.strcpy(name, C.CString(info.Name))
}

//export GoCreateInstance
func GoCreateInstance(cid *C.char, iid *C.char) unsafe.Pointer {
	if globalPlugin == nil {
		return nil
	}

	// Check if the class ID matches our plugin
	var requestedCID [16]byte
	C.memcpy(unsafe.Pointer(&requestedCID[0]), unsafe.Pointer(cid), 16)

	pluginInfo := globalPlugin.GetInfo()
	pluginUID := pluginInfo.UID()
	if requestedCID != pluginUID {
		return nil
	}

	// Create processor instance
	processor := globalPlugin.CreateProcessor()
	if processor == nil {
		return nil
	}
	
	// Optionally wrap with BufferedProcessor for GC protection
	if globalConfig.EnableBuffering {
		channels := globalConfig.BufferChannels
		if channels <= 0 {
			channels = 2 // Default to stereo
		}
		processor = NewBufferedProcessor(processor, channels)
	}

	// Wrap in component implementation
	component := newComponent(processor)

	// Create wrapper
	wrapper := &componentWrapper{
		component: component,
	}

	// Set wrapper reference in component for notifications
	component.wrapper = wrapper

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

	if id == 0 {
		return
	}

	unregisterComponent(id)
}

// All the IComponent callbacks
//
//export GoComponentInitialize
func GoComponentInitialize(componentPtr unsafe.Pointer, context unsafe.Pointer) C.Steinberg_tresult {
	defer recoverPanic("GoComponentInitialize")

	wrapper := getComponent(uintptr(componentPtr))
	if wrapper == nil {
		return C.Steinberg_tresult(vst3.ResultFalse)
	}

	err := wrapper.component.Initialize(context)
	if err != nil {
		return C.Steinberg_tresult(vst3.ResultFalse)
	}
	return C.Steinberg_tresult(vst3.ResultOK)
}

//export GoComponentTerminate
func GoComponentTerminate(componentPtr unsafe.Pointer) C.Steinberg_tresult {
	defer recoverPanic("GoComponentTerminate")

	wrapper := getComponent(uintptr(componentPtr))
	if wrapper == nil {
		return C.Steinberg_tresult(vst3.ResultFalse)
	}

	err := wrapper.component.Terminate()
	if err != nil {
		return C.Steinberg_tresult(vst3.ResultFalse)
	}
	return C.Steinberg_tresult(vst3.ResultOK)
}

//export GoComponentGetControllerClassId
func GoComponentGetControllerClassId(componentPtr unsafe.Pointer, classId *C.char) {
	wrapper := getComponent(uintptr(componentPtr))
	if wrapper == nil {
		return
	}

	uid := wrapper.component.GetControllerClassID()
	C.memcpy(unsafe.Pointer(classId), unsafe.Pointer(&uid[0]), 16)
}

//export GoComponentSetIoMode
func GoComponentSetIoMode(componentPtr unsafe.Pointer, mode C.int32_t) C.Steinberg_tresult {
	wrapper := getComponent(uintptr(componentPtr))
	if wrapper == nil {
		return C.Steinberg_tresult(vst3.ResultFalse)
	}

	err := wrapper.component.SetIOMode(int32(mode))
	if err != nil {
		return C.Steinberg_tresult(vst3.ResultFalse)
	}
	return C.Steinberg_tresult(vst3.ResultOK)
}

//export GoComponentGetBusCount
func GoComponentGetBusCount(componentPtr unsafe.Pointer, mediaType, dir C.int32_t) C.int32_t {
	wrapper := getComponent(uintptr(componentPtr))
	if wrapper == nil {
		return 0
	}

	return C.int32_t(wrapper.component.GetBusCount(int32(mediaType), int32(dir)))
}

//export GoComponentGetBusInfo
func GoComponentGetBusInfo(componentPtr unsafe.Pointer, mediaType, dir, index C.int32_t, bus unsafe.Pointer) C.Steinberg_tresult {
	wrapper := getComponent(uintptr(componentPtr))
	if wrapper == nil {
		return C.Steinberg_tresult(vst3.ResultFalse)
	}

	info, err := wrapper.component.GetBusInfo(int32(mediaType), int32(dir), int32(index))
	if err != nil || info == nil {
		return C.Steinberg_tresult(vst3.ResultFalse)
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

	return C.Steinberg_tresult(vst3.ResultOK)
}

//export GoComponentActivateBus
func GoComponentActivateBus(componentPtr unsafe.Pointer, mediaType, dir, index, state C.int32_t) C.Steinberg_tresult {
	wrapper := getComponent(uintptr(componentPtr))
	if wrapper == nil {
		return C.Steinberg_tresult(vst3.ResultFalse)
	}

	err := wrapper.component.ActivateBus(int32(mediaType), int32(dir), int32(index), state != 0)
	if err != nil {
		return C.Steinberg_tresult(vst3.ResultFalse)
	}
	return C.Steinberg_tresult(vst3.ResultOK)
}

//export GoComponentSetActive
func GoComponentSetActive(componentPtr unsafe.Pointer, state C.int32_t) C.Steinberg_tresult {
	wrapper := getComponent(uintptr(componentPtr))
	if wrapper == nil {
		return C.Steinberg_tresult(vst3.ResultFalse)
	}

	err := wrapper.component.SetActive(state != 0)
	if err != nil {
		return C.Steinberg_tresult(vst3.ResultFalse)
	}
	return C.Steinberg_tresult(vst3.ResultOK)
}

//export GoComponentSetState
func GoComponentSetState(componentPtr unsafe.Pointer, state unsafe.Pointer) C.Steinberg_tresult {
	wrapper := getComponent(uintptr(componentPtr))
	if wrapper == nil {
		return C.Steinberg_tresult(vst3.ResultFalse)
	}

	// Read from VST3 stream
	streamWrapper := vst3.NewStreamWrapper(state)
	if streamWrapper == nil {
		return C.Steinberg_tresult(vst3.ResultFalse)
	}

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
	if streamWrapper == nil {
		return C.Steinberg_tresult(vst3.ResultFalse)
	}

	_, err = streamWrapper.Write(stateData)
	if err != nil {
		return C.Steinberg_tresult(vst3.ResultFalse)
	}

	return C.Steinberg_tresult(vst3.ResultOK)
}
