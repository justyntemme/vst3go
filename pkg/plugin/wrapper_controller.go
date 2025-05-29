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

// Helper function to copy Go string to VST3 UTF-16 string
func copyStringToVST3String(src string, dst *C.Steinberg_Vst_TChar, maxLen int) {
	// VST3 uses UTF-16 (TChar = int16)
	// For now, do simple ASCII conversion
	srcBytes := []byte(src)
	dstSlice := (*[1 << 30]C.Steinberg_Vst_TChar)(unsafe.Pointer(dst))[:maxLen:maxLen]
	
	i := 0
	for i < len(srcBytes) && i < maxLen-1 {
		dstSlice[i] = C.Steinberg_Vst_TChar(srcBytes[i])
		i++
	}
	// Null terminate
	dstSlice[i] = 0
}

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
		return 0
	}
	
	if wrapper.component == nil {
		return 0
	}
	
	count := wrapper.component.GetParameterCount()
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
	
	// VST3 uses UTF-16 strings (Steinberg_Vst_TChar = int16)
	// For now, we'll do simple ASCII conversion
	copyStringToVST3String(paramInfo.Title, &info.title[0], 128)
	copyStringToVST3String(paramInfo.ShortTitle, &info.shortTitle[0], 128)
	copyStringToVST3String(paramInfo.Units, &info.units[0], 128)
	
	info.stepCount = C.int32_t(paramInfo.StepCount)
	info.defaultNormalizedValue = C.Steinberg_Vst_ParamValue(paramInfo.DefaultValue)
	info.unitId = C.Steinberg_Vst_UnitID(paramInfo.UnitID)
	info.flags = C.int32_t(paramInfo.Flags)
	
	return C.Steinberg_tresult(0)
}

//export GoEditControllerGetParamStringByValue
func GoEditControllerGetParamStringByValue(componentPtr unsafe.Pointer, id C.Steinberg_Vst_ParamID, valueNormalized C.Steinberg_Vst_ParamValue, string *C.Steinberg_Vst_TChar) C.Steinberg_tresult {
	idVal := uintptr(componentPtr)
	wrapper := getComponent(idVal)
	if wrapper == nil || string == nil {
		return C.Steinberg_tresult(2)
	}
	
	// Get string representation from component
	str, err := wrapper.component.GetParamStringByValue(uint32(id), float64(valueNormalized))
	if err != nil {
		// If not implemented, format the plain value
		plain := wrapper.component.NormalizedParamToPlain(uint32(id), float64(valueNormalized))
		
		// Get parameter info to check for units
		paramInfo, err := wrapper.component.GetParameterInfo(findParameterIndex(wrapper.component, uint32(id)))
		if err == nil && paramInfo.Units != "" {
			str = formatValueWithUnit(plain, paramInfo.Units)
		} else {
			str = formatValue(plain)
		}
	}
	
	// Copy to VST3 string
	copyStringToVST3String(str, string, 128)
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

// Helper function to find parameter index by ID  
func findParameterIndex(component Component, id uint32) int32 {
	count := component.GetParameterCount()
	for i := int32(0); i < count; i++ {
		info, err := component.GetParameterInfo(i)
		if err == nil && info.ID == id {
			return i
		}
	}
	return -1
}

// Helper function to format a value
func formatValue(value float64) string {
	// Format with appropriate precision
	if value == float64(int(value)) {
		return fmt.Sprintf("%.0f", value)
	}
	return fmt.Sprintf("%.2f", value)
}

// Helper function to format a value with unit
func formatValueWithUnit(value float64, unit string) string {
	return fmt.Sprintf("%s %s", formatValue(value), unit)
}