package plugin

// #cgo CFLAGS: -I../../include
// #include "../../include/vst3/vst3_c_api.h"
// #include <stdlib.h>
import "C"
import (
	"unsafe"
	
	"github.com/justyntemme/vst3go/pkg/vst3"
)

// IAudioProcessor callbacks

//export GoAudioSetBusArrangements
func GoAudioSetBusArrangements(componentPtr unsafe.Pointer, inputs unsafe.Pointer, numIns C.int32_t, outputs unsafe.Pointer, numOuts C.int32_t) C.Steinberg_tresult {
	id := uintptr(componentPtr)
	wrapper := getComponent(id)
	if wrapper == nil {
		return C.Steinberg_tresult(2)
	}
	
	// Convert speaker arrangements
	var inputArrs []int64
	if numIns > 0 && inputs != nil {
		inputPtrs := (*[1 << 30]C.Steinberg_Vst_SpeakerArrangement)(inputs)[:numIns:numIns]
		inputArrs = make([]int64, numIns)
		for i := 0; i < int(numIns); i++ {
			inputArrs[i] = int64(inputPtrs[i])
		}
	}
	
	var outputArrs []int64
	if numOuts > 0 && outputs != nil {
		outputPtrs := (*[1 << 30]C.Steinberg_Vst_SpeakerArrangement)(outputs)[:numOuts:numOuts]
		outputArrs = make([]int64, numOuts)
		for i := 0; i < int(numOuts); i++ {
			outputArrs[i] = int64(outputPtrs[i])
		}
	}
	
	err := wrapper.component.SetBusArrangements(inputArrs, outputArrs)
	if err != nil {
		return C.Steinberg_tresult(1)
	}
	return C.Steinberg_tresult(0)
}

//export GoAudioGetBusArrangement
func GoAudioGetBusArrangement(componentPtr unsafe.Pointer, dir, index C.int32_t, arr unsafe.Pointer) C.Steinberg_tresult {
	id := uintptr(componentPtr)
	wrapper := getComponent(id)
	if wrapper == nil {
		return C.Steinberg_tresult(2)
	}
	
	arrangement, err := wrapper.component.GetBusArrangement(int32(dir), int32(index))
	if err != nil {
		return C.Steinberg_tresult(1)
	}
	
	*(*C.Steinberg_Vst_SpeakerArrangement)(arr) = C.Steinberg_Vst_SpeakerArrangement(arrangement)
	return C.Steinberg_tresult(0)
}

//export GoAudioCanProcessSampleSize
func GoAudioCanProcessSampleSize(componentPtr unsafe.Pointer, symbolicSampleSize C.int32_t) C.Steinberg_tresult {
	id := uintptr(componentPtr)
	wrapper := getComponent(id)
	if wrapper == nil {
		return C.Steinberg_tresult(2)
	}
	
	err := wrapper.component.CanProcessSampleSize(int32(symbolicSampleSize))
	if err != nil {
		return C.Steinberg_tresult(1)
	}
	return C.Steinberg_tresult(0)
}

//export GoAudioGetLatencySamples
func GoAudioGetLatencySamples(componentPtr unsafe.Pointer) C.uint32_t {
	id := uintptr(componentPtr)
	wrapper := getComponent(id)
	if wrapper == nil {
		return 0
	}
	
	return C.uint32_t(wrapper.component.GetLatencySamples())
}

//export GoAudioSetupProcessing
func GoAudioSetupProcessing(componentPtr unsafe.Pointer, setup unsafe.Pointer) C.Steinberg_tresult {
	id := uintptr(componentPtr)
	wrapper := getComponent(id)
	if wrapper == nil {
		return C.Steinberg_tresult(2)
	}
	
	// Convert process setup
	cSetup := (*C.struct_Steinberg_Vst_ProcessSetup)(setup)
	goSetup := &vst3.ProcessSetup{
		ProcessMode:        int32(cSetup.processMode),
		SymbolicSampleSize: int32(cSetup.symbolicSampleSize),
		MaxSamplesPerBlock: int32(cSetup.maxSamplesPerBlock),
		SampleRate:         float64(cSetup.sampleRate),
	}
	
	err := wrapper.component.SetupProcessing(goSetup)
	if err != nil {
		return C.Steinberg_tresult(1)
	}
	return C.Steinberg_tresult(0)
}

//export GoAudioSetProcessing
func GoAudioSetProcessing(componentPtr unsafe.Pointer, state C.int32_t) C.Steinberg_tresult {
	id := uintptr(componentPtr)
	wrapper := getComponent(id)
	if wrapper == nil {
		return C.Steinberg_tresult(2)
	}
	
	err := wrapper.component.SetProcessing(state != 0)
	if err != nil {
		return C.Steinberg_tresult(1)
	}
	return C.Steinberg_tresult(0)
}

//export GoAudioProcess
func GoAudioProcess(componentPtr unsafe.Pointer, data unsafe.Pointer) C.Steinberg_tresult {
	id := uintptr(componentPtr)
	wrapper := getComponent(id)
	if wrapper == nil {
		return C.Steinberg_tresult(2)
	}
	
	err := wrapper.component.Process(data)
	if err != nil {
		return C.Steinberg_tresult(1)
	}
	return C.Steinberg_tresult(0)
}

//export GoAudioGetTailSamples
func GoAudioGetTailSamples(componentPtr unsafe.Pointer) C.uint32_t {
	id := uintptr(componentPtr)
	wrapper := getComponent(id)
	if wrapper == nil {
		return 0
	}
	
	return C.uint32_t(wrapper.component.GetTailSamples())
}