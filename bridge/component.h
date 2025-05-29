#ifndef VST3GO_COMPONENT_H
#define VST3GO_COMPONENT_H

#include "../include/vst3/vst3_c_api.h"

// C function to create a component wrapper
void* createComponent(void* goComponent);

// Go callback declarations for IComponent
extern Steinberg_tresult GoComponentInitialize(void* component, void* context);
extern Steinberg_tresult GoComponentTerminate(void* component);
extern void GoComponentGetControllerClassId(void* component, char* classId);
extern Steinberg_tresult GoComponentSetIoMode(void* component, int32_t mode);
extern int32_t GoComponentGetBusCount(void* component, int32_t type, int32_t dir);
extern Steinberg_tresult GoComponentGetBusInfo(void* component, int32_t type, int32_t dir, int32_t index, void* bus);
extern Steinberg_tresult GoComponentActivateBus(void* component, int32_t type, int32_t dir, int32_t index, int32_t state);
extern Steinberg_tresult GoComponentSetActive(void* component, int32_t state);

// Go callback declarations for IAudioProcessor
extern Steinberg_tresult GoAudioSetBusArrangements(void* component, void* inputs, int32_t numIns, void* outputs, int32_t numOuts);
extern Steinberg_tresult GoAudioGetBusArrangement(void* component, int32_t dir, int32_t index, void* arr);
extern Steinberg_tresult GoAudioCanProcessSampleSize(void* component, int32_t symbolicSampleSize);
extern uint32_t GoAudioGetLatencySamples(void* component);
extern Steinberg_tresult GoAudioSetupProcessing(void* component, void* setup);
extern Steinberg_tresult GoAudioSetProcessing(void* component, int32_t state);
extern Steinberg_tresult GoAudioProcess(void* component, void* data);
extern uint32_t GoAudioGetTailSamples(void* component);

// Go component lifecycle
extern void GoReleaseComponent(void* component);

#endif // VST3GO_COMPONENT_H