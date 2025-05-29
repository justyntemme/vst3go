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
extern Steinberg_tresult GoComponentSetState(void* component, void* state);
extern Steinberg_tresult GoComponentGetState(void* component, void* state);

// Go callback declarations for IAudioProcessor
extern Steinberg_tresult GoAudioSetBusArrangements(void* component, void* inputs, int32_t numIns, void* outputs, int32_t numOuts);
extern Steinberg_tresult GoAudioGetBusArrangement(void* component, int32_t dir, int32_t index, void* arr);
extern Steinberg_tresult GoAudioCanProcessSampleSize(void* component, int32_t symbolicSampleSize);
extern uint32_t GoAudioGetLatencySamples(void* component);
extern Steinberg_tresult GoAudioSetupProcessing(void* component, void* setup);
extern Steinberg_tresult GoAudioSetProcessing(void* component, int32_t state);
extern Steinberg_tresult GoAudioProcess(void* component, void* data);
extern uint32_t GoAudioGetTailSamples(void* component);

// Go callback declarations for IEditController
extern Steinberg_tresult GoEditControllerSetComponentState(void* component, void* state);
extern Steinberg_tresult GoEditControllerSetState(void* component, void* state);
extern Steinberg_tresult GoEditControllerGetState(void* component, void* state);
extern int32_t GoEditControllerGetParameterCount(void* component);
extern Steinberg_tresult GoEditControllerGetParameterInfo(void* component, int32_t paramIndex, struct Steinberg_Vst_ParameterInfo* info);
extern Steinberg_tresult GoEditControllerGetParamStringByValue(void* component, Steinberg_Vst_ParamID id, Steinberg_Vst_ParamValue valueNormalized, Steinberg_Vst_TChar* string);
extern Steinberg_tresult GoEditControllerGetParamValueByString(void* component, Steinberg_Vst_ParamID id, Steinberg_Vst_TChar* string, Steinberg_Vst_ParamValue* valueNormalized);
extern double GoEditControllerNormalizedParamToPlain(void* component, uint32_t id, double valueNormalized);
extern double GoEditControllerPlainParamToNormalized(void* component, uint32_t id, double plainValue);
extern double GoEditControllerGetParamNormalized(void* component, uint32_t id);
extern Steinberg_tresult GoEditControllerSetParamNormalized(void* component, uint32_t id, double value);
extern Steinberg_tresult GoEditControllerSetComponentHandler(void* component, void* handler);
extern void* GoEditControllerCreateView(void* component, char* name);

// Go component lifecycle
extern void GoReleaseComponent(void* component);

#endif // VST3GO_COMPONENT_H