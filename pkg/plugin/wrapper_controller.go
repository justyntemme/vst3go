package plugin

// #cgo CFLAGS: -I../../include
// #include "../../include/vst3/vst3_c_api.h"
// #include <stdlib.h>
// #include <string.h>
import "C"
import (
	"fmt"
	"unsafe"
)

// IEditController callbacks

//export GoEditControllerSetComponentState
func GoEditControllerSetComponentState(componentPtr unsafe.Pointer, state unsafe.Pointer) C.Steinberg_tresult {
	// TODO: Implement
	return C.Steinberg_tresult(0)
}

//export GoEditControllerSetState
func GoEditControllerSetState(componentPtr unsafe.Pointer, state unsafe.Pointer) C.Steinberg_tresult {
	// TODO: Implement
	return C.Steinberg_tresult(0)
}

//export GoEditControllerGetState
func GoEditControllerGetState(componentPtr unsafe.Pointer, state unsafe.Pointer) C.Steinberg_tresult {
	// TODO: Implement
	return C.Steinberg_tresult(0)
}

//export GoEditControllerGetParameterCount
func GoEditControllerGetParameterCount(componentPtr unsafe.Pointer) C.int32_t {
	id := uintptr(componentPtr)
	wrapper := getComponent(id)
	if wrapper == nil {
		fmt.Printf("GoEditControllerGetParameterCount: wrapper is nil for id %v\n", id)
		return 0
	}
	
	if wrapper.component == nil {
		fmt.Printf("GoEditControllerGetParameterCount: wrapper.component is nil\n")
		return 0
	}
	
	count := wrapper.component.GetParameterCount()
	fmt.Printf("GoEditControllerGetParameterCount: returning %d parameters\n", count)
	return C.int32_t(count)
}

//export GoEditControllerGetParameterInfo
func GoEditControllerGetParameterInfo(componentPtr unsafe.Pointer, paramIndex C.int32_t, info *C.struct_Steinberg_Vst_ParameterInfo) C.Steinberg_tresult {
	id := uintptr(componentPtr)
	wrapper := getComponent(id)
	if wrapper == nil || info == nil {
		return C.Steinberg_tresult(2)
	}
	
	paramInfo, err := wrapper.component.GetParameterInfo(int32(paramIndex))
	if err != nil {
		return C.Steinberg_tresult(1)
	}
	
	// Copy parameter info to C struct
	info.id = C.Steinberg_Vst_ParamID(paramInfo.ID)
	C.strncpy((*C.char)(unsafe.Pointer(&info.title[0])), C.CString(paramInfo.Title), 128)
	C.strncpy((*C.char)(unsafe.Pointer(&info.shortTitle[0])), C.CString(paramInfo.ShortTitle), 128)
	C.strncpy((*C.char)(unsafe.Pointer(&info.units[0])), C.CString(paramInfo.Units), 128)
	info.stepCount = C.int32_t(paramInfo.StepCount)
	info.defaultNormalizedValue = C.Steinberg_Vst_ParamValue(paramInfo.DefaultValue)
	info.unitId = C.Steinberg_Vst_UnitID(paramInfo.UnitID)
	info.flags = C.int32_t(paramInfo.Flags)
	
	return C.Steinberg_tresult(0)
}

//export GoEditControllerGetParamStringByValue
func GoEditControllerGetParamStringByValue(componentPtr unsafe.Pointer, id C.Steinberg_Vst_ParamID, valueNormalized C.Steinberg_Vst_ParamValue, string *C.Steinberg_Vst_TChar) C.Steinberg_tresult {
	// TODO: Implement parameter value to string conversion
	return C.Steinberg_tresult(0)
}

//export GoEditControllerGetParamValueByString
func GoEditControllerGetParamValueByString(componentPtr unsafe.Pointer, id C.Steinberg_Vst_ParamID, string *C.Steinberg_Vst_TChar, valueNormalized *C.Steinberg_Vst_ParamValue) C.Steinberg_tresult {
	// TODO: Implement string to parameter value conversion
	return C.Steinberg_tresult(1)
}

//export GoEditControllerNormalizedParamToPlain
func GoEditControllerNormalizedParamToPlain(componentPtr unsafe.Pointer, id C.Steinberg_Vst_ParamID, valueNormalized C.Steinberg_Vst_ParamValue) C.Steinberg_Vst_ParamValue {
	idVal := uintptr(componentPtr)
	wrapper := getComponent(idVal)
	if wrapper == nil {
		return valueNormalized
	}
	
	plain := wrapper.component.NormalizedParamToPlain(uint32(id), float64(valueNormalized))
	return C.Steinberg_Vst_ParamValue(plain)
}

//export GoEditControllerPlainParamToNormalized
func GoEditControllerPlainParamToNormalized(componentPtr unsafe.Pointer, id C.Steinberg_Vst_ParamID, plainValue C.Steinberg_Vst_ParamValue) C.Steinberg_Vst_ParamValue {
	idVal := uintptr(componentPtr)
	wrapper := getComponent(idVal)
	if wrapper == nil {
		return plainValue
	}
	
	normalized := wrapper.component.PlainParamToNormalized(uint32(id), float64(plainValue))
	return C.Steinberg_Vst_ParamValue(normalized)
}

//export GoEditControllerGetParamNormalized
func GoEditControllerGetParamNormalized(componentPtr unsafe.Pointer, id C.Steinberg_Vst_ParamID) C.Steinberg_Vst_ParamValue {
	idVal := uintptr(componentPtr)
	wrapper := getComponent(idVal)
	if wrapper == nil {
		return 0
	}
	
	value := wrapper.component.GetParamNormalized(uint32(id))
	return C.Steinberg_Vst_ParamValue(value)
}

//export GoEditControllerSetParamNormalized
func GoEditControllerSetParamNormalized(componentPtr unsafe.Pointer, id C.Steinberg_Vst_ParamID, value C.Steinberg_Vst_ParamValue) C.Steinberg_tresult {
	idVal := uintptr(componentPtr)
	wrapper := getComponent(idVal)
	if wrapper == nil {
		return C.Steinberg_tresult(2)
	}
	
	err := wrapper.component.SetParamNormalized(uint32(id), float64(value))
	if err != nil {
		return C.Steinberg_tresult(1)
	}
	return C.Steinberg_tresult(0)
}

//export GoEditControllerSetComponentHandler
func GoEditControllerSetComponentHandler(componentPtr unsafe.Pointer, handler unsafe.Pointer) C.Steinberg_tresult {
	// TODO: Store component handler for parameter change notifications
	return C.Steinberg_tresult(0)
}

//export GoEditControllerCreateView
func GoEditControllerCreateView(componentPtr unsafe.Pointer, name *C.char) unsafe.Pointer {
	// No UI support yet
	return nil
}