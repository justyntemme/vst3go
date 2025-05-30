package plugin

// #cgo CFLAGS: -I../../include
// #include "../../include/vst3/vst3_c_api.h"
// 
// // Helper to access channelBuffers32 from the union
// static inline float** getChannelBuffers32(struct Steinberg_Vst_AudioBusBuffers* bus) {
//     return bus->Steinberg_Vst_AudioBusBuffers_channelBuffers32;
// }
import "C"

// getChannelBuffers32 extracts the 32-bit channel buffers from an audio bus
func getChannelBuffers32(bus *C.struct_Steinberg_Vst_AudioBusBuffers) **C.float {
	return C.getChannelBuffers32(bus)
}