package plugin

// #cgo CFLAGS: -I../../include
// #include "../../include/vst3/vst3_c_api.h"
// #include <stdlib.h>
// #include <string.h>
import "C"
import (
	"unsafe"
)

// Plugin interface that users will implement
type Plugin interface {
	GetPluginName() string
	GetVendorName() string
	GetPluginVersion() string
	GetUID() [16]byte
	CreateComponent() Component
}

// FactoryInfo provides information about the plugin factory
type FactoryInfo struct {
	Vendor string
	URL    string
	Email  string
}

// Global plugin instance (set by user)
var globalPlugin Plugin
var globalFactoryInfo = FactoryInfo{
	Vendor: "VST3Go",
	URL:    "https://github.com/vst3go",
	Email:  "info@vst3go.com",
}

// RegisterPlugin sets the global plugin instance
func RegisterPlugin(p Plugin) {
	globalPlugin = p
}

// SetFactoryInfo allows customizing factory information
func SetFactoryInfo(info FactoryInfo) {
	globalFactoryInfo = info
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
	
	// Copy UID
	uid := globalPlugin.GetUID()
	C.memcpy(unsafe.Pointer(cid), unsafe.Pointer(&uid[0]), 16)
	
	// Set cardinality
	*cardinality = C.Steinberg_PClassInfo_ClassCardinality_kManyInstances
	
	// Set category and name
	C.strcpy(category, C.CString("Audio Module Class"))
	C.strcpy(name, C.CString(globalPlugin.GetPluginName()))
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
	
	// Create component instance
	// For now, return nil - will implement component wrapper later
	return nil
}