package plugin

// #cgo CFLAGS: -I../../include
// #include "../../include/vst3/vst3_c_api.h"
//
// // Helper to access channelBuffers32 from the union
// static inline float** getChannelBuffers32(struct Steinberg_Vst_AudioBusBuffers* bus) {
//     return bus->Steinberg_Vst_AudioBusBuffers_channelBuffers32;
// }
import "C"
import "unsafe"

// getChannelBuffers32 extracts the 32-bit channel buffers from an audio bus
func getChannelBuffers32(bus *C.struct_Steinberg_Vst_AudioBusBuffers) **C.float {
	return C.getChannelBuffers32(bus)
}

// copyStringToTChar copies a Go string to a VST3 TChar (UTF16) buffer
func copyStringToTChar(src string, dst *C.Steinberg_Vst_TChar, maxLen int) {
	// Convert to runes for proper Unicode handling
	runes := []rune(src)
	n := len(runes)
	if n > maxLen-1 {
		n = maxLen - 1
	}

	// Copy runes as UTF16
	for i := 0; i < n; i++ {
		*(*C.Steinberg_char16)(unsafe.Pointer(
			uintptr(unsafe.Pointer(dst)) + uintptr(i*2))) = C.Steinberg_char16(runes[i])
	}

	// Null terminate
	*(*C.Steinberg_char16)(unsafe.Pointer(
		uintptr(unsafe.Pointer(dst)) + uintptr(n*2))) = 0
}

// stringFromTChar converts a VST3 TChar (UTF16) buffer to a Go string
func stringFromTChar(src *C.Steinberg_Vst_TChar) string {
	if src == nil {
		return ""
	}

	// Count length
	length := 0
	for {
		ch := *(*C.Steinberg_char16)(unsafe.Pointer(
			uintptr(unsafe.Pointer(src)) + uintptr(length*2)))
		if ch == 0 {
			break
		}
		length++
	}

	// Convert to runes
	runes := make([]rune, length)
	for i := 0; i < length; i++ {
		runes[i] = rune(*(*C.Steinberg_char16)(unsafe.Pointer(
			uintptr(unsafe.Pointer(src)) + uintptr(i*2))))
	}

	return string(runes)
}
