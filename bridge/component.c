#include "component.h"
#include <string.h>
#include <stdlib.h>

// Forward declare
typedef struct Component Component;

// Audio processor interface wrapper
typedef struct {
    struct Steinberg_Vst_IAudioProcessorVtbl* lpVtbl;
    Component* component;
} AudioProcessorInterface;

// Edit controller interface wrapper
typedef struct {
    struct Steinberg_Vst_IEditControllerVtbl* lpVtbl;
    Component* component;
} EditControllerInterface;

// Component implementation that wraps Go component
struct Component {
    // IComponent vtable pointer must be first for COM compatibility
    struct Steinberg_Vst_IComponentVtbl* lpVtbl;
    // Audio processor interface
    AudioProcessorInterface audioProcessor;
    // Edit controller interface
    EditControllerInterface editController;
    // Reference count
    int refCount;
    // Go component handle
    void* goComponent;
};

// Forward declarations for IComponent methods
static Steinberg_tresult SMTG_STDMETHODCALLTYPE component_queryInterface(void* thisInterface, const Steinberg_TUID iid, void** obj);
static Steinberg_uint32 SMTG_STDMETHODCALLTYPE component_addRef(void* thisInterface);
static Steinberg_uint32 SMTG_STDMETHODCALLTYPE component_release(void* thisInterface);
static Steinberg_tresult SMTG_STDMETHODCALLTYPE component_initialize(void* thisInterface, struct Steinberg_FUnknown* context);
static Steinberg_tresult SMTG_STDMETHODCALLTYPE component_terminate(void* thisInterface);
static Steinberg_tresult SMTG_STDMETHODCALLTYPE component_getControllerClassId(void* thisInterface, Steinberg_TUID classId);
static Steinberg_tresult SMTG_STDMETHODCALLTYPE component_setIoMode(void* thisInterface, Steinberg_Vst_IoMode mode);
static Steinberg_int32 SMTG_STDMETHODCALLTYPE component_getBusCount(void* thisInterface, Steinberg_Vst_MediaType type, Steinberg_Vst_BusDirection dir);
static Steinberg_tresult SMTG_STDMETHODCALLTYPE component_getBusInfo(void* thisInterface, Steinberg_Vst_MediaType type, Steinberg_Vst_BusDirection dir, Steinberg_int32 index, struct Steinberg_Vst_BusInfo* bus);
static Steinberg_tresult SMTG_STDMETHODCALLTYPE component_getRoutingInfo(void* thisInterface, struct Steinberg_Vst_RoutingInfo* inInfo, struct Steinberg_Vst_RoutingInfo* outInfo);
static Steinberg_tresult SMTG_STDMETHODCALLTYPE component_activateBus(void* thisInterface, Steinberg_Vst_MediaType type, Steinberg_Vst_BusDirection dir, Steinberg_int32 index, Steinberg_TBool state);
static Steinberg_tresult SMTG_STDMETHODCALLTYPE component_setActive(void* thisInterface, Steinberg_TBool state);
static Steinberg_tresult SMTG_STDMETHODCALLTYPE component_setState(void* thisInterface, struct Steinberg_IBStream* state);
static Steinberg_tresult SMTG_STDMETHODCALLTYPE component_getState(void* thisInterface, struct Steinberg_IBStream* state);

// Forward declarations for IAudioProcessor IUnknown methods
static Steinberg_tresult SMTG_STDMETHODCALLTYPE audio_queryInterface(void* thisInterface, const Steinberg_TUID iid, void** obj);
static Steinberg_uint32 SMTG_STDMETHODCALLTYPE audio_addRef(void* thisInterface);
static Steinberg_uint32 SMTG_STDMETHODCALLTYPE audio_release(void* thisInterface);

// Forward declarations for IAudioProcessor methods
static Steinberg_tresult SMTG_STDMETHODCALLTYPE audio_setBusArrangements(void* thisInterface, Steinberg_Vst_SpeakerArrangement* inputs, Steinberg_int32 numIns, Steinberg_Vst_SpeakerArrangement* outputs, Steinberg_int32 numOuts);
static Steinberg_tresult SMTG_STDMETHODCALLTYPE audio_getBusArrangement(void* thisInterface, Steinberg_Vst_BusDirection dir, Steinberg_int32 index, Steinberg_Vst_SpeakerArrangement* arr);
static Steinberg_tresult SMTG_STDMETHODCALLTYPE audio_canProcessSampleSize(void* thisInterface, Steinberg_int32 symbolicSampleSize);
static Steinberg_uint32 SMTG_STDMETHODCALLTYPE audio_getLatencySamples(void* thisInterface);
static Steinberg_tresult SMTG_STDMETHODCALLTYPE audio_setupProcessing(void* thisInterface, struct Steinberg_Vst_ProcessSetup* setup);
static Steinberg_tresult SMTG_STDMETHODCALLTYPE audio_setProcessing(void* thisInterface, Steinberg_TBool state);
static Steinberg_tresult SMTG_STDMETHODCALLTYPE audio_process(void* thisInterface, struct Steinberg_Vst_ProcessData* data);
static Steinberg_uint32 SMTG_STDMETHODCALLTYPE audio_getTailSamples(void* thisInterface);

// Forward declarations for IEditController IUnknown methods
static Steinberg_tresult SMTG_STDMETHODCALLTYPE controller_queryInterface(void* thisInterface, const Steinberg_TUID iid, void** obj);
static Steinberg_uint32 SMTG_STDMETHODCALLTYPE controller_addRef(void* thisInterface);
static Steinberg_uint32 SMTG_STDMETHODCALLTYPE controller_release(void* thisInterface);

// Forward declarations for IEditController IPluginBase methods
static Steinberg_tresult SMTG_STDMETHODCALLTYPE controller_initialize(void* thisInterface, struct Steinberg_FUnknown* context);
static Steinberg_tresult SMTG_STDMETHODCALLTYPE controller_terminate(void* thisInterface);

// Forward declarations for IEditController methods
static Steinberg_tresult SMTG_STDMETHODCALLTYPE controller_setComponentState(void* thisInterface, struct Steinberg_IBStream* state);
static Steinberg_tresult SMTG_STDMETHODCALLTYPE controller_setState(void* thisInterface, struct Steinberg_IBStream* state);
static Steinberg_tresult SMTG_STDMETHODCALLTYPE controller_getState(void* thisInterface, struct Steinberg_IBStream* state);
static Steinberg_int32 SMTG_STDMETHODCALLTYPE controller_getParameterCount(void* thisInterface);
static Steinberg_tresult SMTG_STDMETHODCALLTYPE controller_getParameterInfo(void* thisInterface, Steinberg_int32 paramIndex, struct Steinberg_Vst_ParameterInfo* info);
static Steinberg_tresult SMTG_STDMETHODCALLTYPE controller_getParamStringByValue(void* thisInterface, Steinberg_Vst_ParamID id, Steinberg_Vst_ParamValue valueNormalized, Steinberg_Vst_String128 string);
static Steinberg_tresult SMTG_STDMETHODCALLTYPE controller_getParamValueByString(void* thisInterface, Steinberg_Vst_ParamID id, Steinberg_Vst_TChar* string, Steinberg_Vst_ParamValue* valueNormalized);
static Steinberg_Vst_ParamValue SMTG_STDMETHODCALLTYPE controller_normalizedParamToPlain(void* thisInterface, Steinberg_Vst_ParamID id, Steinberg_Vst_ParamValue valueNormalized);
static Steinberg_Vst_ParamValue SMTG_STDMETHODCALLTYPE controller_plainParamToNormalized(void* thisInterface, Steinberg_Vst_ParamID id, Steinberg_Vst_ParamValue plainValue);
static Steinberg_Vst_ParamValue SMTG_STDMETHODCALLTYPE controller_getParamNormalized(void* thisInterface, Steinberg_Vst_ParamID id);
static Steinberg_tresult SMTG_STDMETHODCALLTYPE controller_setParamNormalized(void* thisInterface, Steinberg_Vst_ParamID id, Steinberg_Vst_ParamValue value);
static Steinberg_tresult SMTG_STDMETHODCALLTYPE controller_setComponentHandler(void* thisInterface, struct Steinberg_Vst_IComponentHandler* handler);
static struct Steinberg_IPlugView* SMTG_STDMETHODCALLTYPE controller_createView(void* thisInterface, Steinberg_FIDString name);

// IComponent vtable
static struct Steinberg_Vst_IComponentVtbl componentVtbl = {
    component_queryInterface,
    component_addRef,
    component_release,
    component_initialize,
    component_terminate,
    component_getControllerClassId,
    component_setIoMode,
    component_getBusCount,
    component_getBusInfo,
    component_getRoutingInfo,
    component_activateBus,
    component_setActive,
    component_setState,
    component_getState
};

// IAudioProcessor vtable  
static struct Steinberg_Vst_IAudioProcessorVtbl audioProcessorVtbl = {
    audio_queryInterface,
    audio_addRef,
    audio_release,
    audio_setBusArrangements,
    audio_getBusArrangement,
    audio_canProcessSampleSize,
    audio_getLatencySamples,
    audio_setupProcessing,
    audio_setProcessing,
    audio_process,
    audio_getTailSamples
};

// IEditController vtable
static struct Steinberg_Vst_IEditControllerVtbl editControllerVtbl = {
    controller_queryInterface,
    controller_addRef,
    controller_release,
    controller_initialize,
    controller_terminate,
    controller_setComponentState,
    controller_setState,
    controller_getState,
    controller_getParameterCount,
    controller_getParameterInfo,
    controller_getParamStringByValue,
    controller_getParamValueByString,
    controller_normalizedParamToPlain,
    controller_plainParamToNormalized,
    controller_getParamNormalized,
    controller_setParamNormalized,
    controller_setComponentHandler,
    controller_createView
};

// Create a new component instance
void* createComponent(void* goComponent) {
    Component* component = (Component*)malloc(sizeof(Component));
    if (!component) return NULL;
    
    component->lpVtbl = &componentVtbl;
    component->audioProcessor.lpVtbl = &audioProcessorVtbl;
    component->audioProcessor.component = component;
    component->editController.lpVtbl = &editControllerVtbl;
    component->editController.component = component;
    component->refCount = 1;
    component->goComponent = goComponent;
    
    return component;
}

// IUnknown implementation
static Steinberg_tresult SMTG_STDMETHODCALLTYPE component_queryInterface(void* thisInterface, const Steinberg_TUID iid, void** obj) {
    Component* component = (Component*)thisInterface;
    
    if (memcmp(iid, Steinberg_FUnknown_iid, sizeof(Steinberg_TUID)) == 0 ||
        memcmp(iid, Steinberg_IPluginBase_iid, sizeof(Steinberg_TUID)) == 0 ||
        memcmp(iid, Steinberg_Vst_IComponent_iid, sizeof(Steinberg_TUID)) == 0) {
        *obj = component; // Return component itself, not vtable pointer
        component_addRef(thisInterface);
        return ((Steinberg_tresult)0);
    }
    
    if (memcmp(iid, Steinberg_Vst_IAudioProcessor_iid, sizeof(Steinberg_TUID)) == 0) {
        *obj = &component->audioProcessor; // Return audio processor interface
        component_addRef(thisInterface);
        return ((Steinberg_tresult)0);
    }
    
    if (memcmp(iid, Steinberg_Vst_IEditController_iid, sizeof(Steinberg_TUID)) == 0) {
        *obj = &component->editController; // Return edit controller interface
        component_addRef(thisInterface);
        return ((Steinberg_tresult)0);
    }
    
    *obj = NULL;
    return ((Steinberg_tresult)-1);
}

static Steinberg_uint32 SMTG_STDMETHODCALLTYPE component_addRef(void* thisInterface) {
    Component* component = (Component*)thisInterface;
    return ++component->refCount;
}

static Steinberg_uint32 SMTG_STDMETHODCALLTYPE component_release(void* thisInterface) {
    Component* component = (Component*)thisInterface;
    if (--component->refCount == 0) {
        // Release Go component
        GoReleaseComponent(component->goComponent);
        free(component);
        return 0;
    }
    return component->refCount;
}

// IPluginBase implementation
static Steinberg_tresult SMTG_STDMETHODCALLTYPE component_initialize(void* thisInterface, struct Steinberg_FUnknown* context) {
    Component* component = (Component*)thisInterface;
    return GoComponentInitialize(component->goComponent, context);
}

static Steinberg_tresult SMTG_STDMETHODCALLTYPE component_terminate(void* thisInterface) {
    Component* component = (Component*)thisInterface;
    return GoComponentTerminate(component->goComponent);
}

// IComponent implementation
static Steinberg_tresult SMTG_STDMETHODCALLTYPE component_getControllerClassId(void* thisInterface, Steinberg_TUID classId) {
    Component* component = (Component*)thisInterface;
    GoComponentGetControllerClassId(component->goComponent, classId);
    return ((Steinberg_tresult)0);
}

static Steinberg_tresult SMTG_STDMETHODCALLTYPE component_setIoMode(void* thisInterface, Steinberg_Vst_IoMode mode) {
    Component* component = (Component*)thisInterface;
    return GoComponentSetIoMode(component->goComponent, mode);
}

static Steinberg_int32 SMTG_STDMETHODCALLTYPE component_getBusCount(void* thisInterface, Steinberg_Vst_MediaType type, Steinberg_Vst_BusDirection dir) {
    Component* component = (Component*)thisInterface;
    return GoComponentGetBusCount(component->goComponent, type, dir);
}

static Steinberg_tresult SMTG_STDMETHODCALLTYPE component_getBusInfo(void* thisInterface, Steinberg_Vst_MediaType type, Steinberg_Vst_BusDirection dir, Steinberg_int32 index, struct Steinberg_Vst_BusInfo* bus) {
    Component* component = (Component*)thisInterface;
    return GoComponentGetBusInfo(component->goComponent, type, dir, index, bus);
}

static Steinberg_tresult SMTG_STDMETHODCALLTYPE component_getRoutingInfo(void* thisInterface, struct Steinberg_Vst_RoutingInfo* inInfo, struct Steinberg_Vst_RoutingInfo* outInfo) {
    // Not implemented
    return ((Steinberg_tresult)3);
}

static Steinberg_tresult SMTG_STDMETHODCALLTYPE component_activateBus(void* thisInterface, Steinberg_Vst_MediaType type, Steinberg_Vst_BusDirection dir, Steinberg_int32 index, Steinberg_TBool state) {
    Component* component = (Component*)thisInterface;
    return GoComponentActivateBus(component->goComponent, type, dir, index, state);
}

static Steinberg_tresult SMTG_STDMETHODCALLTYPE component_setActive(void* thisInterface, Steinberg_TBool state) {
    Component* component = (Component*)thisInterface;
    return GoComponentSetActive(component->goComponent, state);
}

static Steinberg_tresult SMTG_STDMETHODCALLTYPE component_setState(void* thisInterface, struct Steinberg_IBStream* state) {
    // TODO: Implement state handling
    return ((Steinberg_tresult)0);
}

static Steinberg_tresult SMTG_STDMETHODCALLTYPE component_getState(void* thisInterface, struct Steinberg_IBStream* state) {
    // TODO: Implement state handling
    return ((Steinberg_tresult)0);
}

// IAudioProcessor IUnknown implementation
static Steinberg_tresult SMTG_STDMETHODCALLTYPE audio_queryInterface(void* thisInterface, const Steinberg_TUID iid, void** obj) {
    AudioProcessorInterface* audioProc = (AudioProcessorInterface*)thisInterface;
    return component_queryInterface(audioProc->component, iid, obj);
}

static Steinberg_uint32 SMTG_STDMETHODCALLTYPE audio_addRef(void* thisInterface) {
    AudioProcessorInterface* audioProc = (AudioProcessorInterface*)thisInterface;
    return component_addRef(audioProc->component);
}

static Steinberg_uint32 SMTG_STDMETHODCALLTYPE audio_release(void* thisInterface) {
    AudioProcessorInterface* audioProc = (AudioProcessorInterface*)thisInterface;
    return component_release(audioProc->component);
}

// IAudioProcessor implementation
static Steinberg_tresult SMTG_STDMETHODCALLTYPE audio_setBusArrangements(void* thisInterface, Steinberg_Vst_SpeakerArrangement* inputs, Steinberg_int32 numIns, Steinberg_Vst_SpeakerArrangement* outputs, Steinberg_int32 numOuts) {
    AudioProcessorInterface* audioProc = (AudioProcessorInterface*)thisInterface;
    return GoAudioSetBusArrangements(audioProc->component->goComponent, inputs, numIns, outputs, numOuts);
}

static Steinberg_tresult SMTG_STDMETHODCALLTYPE audio_getBusArrangement(void* thisInterface, Steinberg_Vst_BusDirection dir, Steinberg_int32 index, Steinberg_Vst_SpeakerArrangement* arr) {
    AudioProcessorInterface* audioProc = (AudioProcessorInterface*)thisInterface;
    return GoAudioGetBusArrangement(audioProc->component->goComponent, dir, index, arr);
}

static Steinberg_tresult SMTG_STDMETHODCALLTYPE audio_canProcessSampleSize(void* thisInterface, Steinberg_int32 symbolicSampleSize) {
    AudioProcessorInterface* audioProc = (AudioProcessorInterface*)thisInterface;
    return GoAudioCanProcessSampleSize(audioProc->component->goComponent, symbolicSampleSize);
}

static Steinberg_uint32 SMTG_STDMETHODCALLTYPE audio_getLatencySamples(void* thisInterface) {
    AudioProcessorInterface* audioProc = (AudioProcessorInterface*)thisInterface;
    return GoAudioGetLatencySamples(audioProc->component->goComponent);
}

static Steinberg_tresult SMTG_STDMETHODCALLTYPE audio_setupProcessing(void* thisInterface, struct Steinberg_Vst_ProcessSetup* setup) {
    AudioProcessorInterface* audioProc = (AudioProcessorInterface*)thisInterface;
    return GoAudioSetupProcessing(audioProc->component->goComponent, setup);
}

static Steinberg_tresult SMTG_STDMETHODCALLTYPE audio_setProcessing(void* thisInterface, Steinberg_TBool state) {
    AudioProcessorInterface* audioProc = (AudioProcessorInterface*)thisInterface;
    return GoAudioSetProcessing(audioProc->component->goComponent, state);
}

static Steinberg_tresult SMTG_STDMETHODCALLTYPE audio_process(void* thisInterface, struct Steinberg_Vst_ProcessData* data) {
    AudioProcessorInterface* audioProc = (AudioProcessorInterface*)thisInterface;
    return GoAudioProcess(audioProc->component->goComponent, data);
}

static Steinberg_uint32 SMTG_STDMETHODCALLTYPE audio_getTailSamples(void* thisInterface) {
    AudioProcessorInterface* audioProc = (AudioProcessorInterface*)thisInterface;
    return GoAudioGetTailSamples(audioProc->component->goComponent);
}

// IEditController IUnknown implementation
static Steinberg_tresult SMTG_STDMETHODCALLTYPE controller_queryInterface(void* thisInterface, const Steinberg_TUID iid, void** obj) {
    EditControllerInterface* controller = (EditControllerInterface*)thisInterface;
    return component_queryInterface(controller->component, iid, obj);
}

static Steinberg_uint32 SMTG_STDMETHODCALLTYPE controller_addRef(void* thisInterface) {
    EditControllerInterface* controller = (EditControllerInterface*)thisInterface;
    return component_addRef(controller->component);
}

static Steinberg_uint32 SMTG_STDMETHODCALLTYPE controller_release(void* thisInterface) {
    EditControllerInterface* controller = (EditControllerInterface*)thisInterface;
    return component_release(controller->component);
}

// IEditController IPluginBase implementation
static Steinberg_tresult SMTG_STDMETHODCALLTYPE controller_initialize(void* thisInterface, struct Steinberg_FUnknown* context) {
    // Already initialized as component
    return ((Steinberg_tresult)0);
}

static Steinberg_tresult SMTG_STDMETHODCALLTYPE controller_terminate(void* thisInterface) {
    // Will be terminated as component
    return ((Steinberg_tresult)0);
}

// IEditController implementation
static Steinberg_tresult SMTG_STDMETHODCALLTYPE controller_setComponentState(void* thisInterface, struct Steinberg_IBStream* state) {
    EditControllerInterface* controller = (EditControllerInterface*)thisInterface;
    return GoEditControllerSetComponentState(controller->component->goComponent, state);
}

static Steinberg_tresult SMTG_STDMETHODCALLTYPE controller_setState(void* thisInterface, struct Steinberg_IBStream* state) {
    EditControllerInterface* controller = (EditControllerInterface*)thisInterface;
    return GoEditControllerSetState(controller->component->goComponent, state);
}

static Steinberg_tresult SMTG_STDMETHODCALLTYPE controller_getState(void* thisInterface, struct Steinberg_IBStream* state) {
    EditControllerInterface* controller = (EditControllerInterface*)thisInterface;
    return GoEditControllerGetState(controller->component->goComponent, state);
}

static Steinberg_int32 SMTG_STDMETHODCALLTYPE controller_getParameterCount(void* thisInterface) {
    EditControllerInterface* controller = (EditControllerInterface*)thisInterface;
    return GoEditControllerGetParameterCount(controller->component->goComponent);
}

static Steinberg_tresult SMTG_STDMETHODCALLTYPE controller_getParameterInfo(void* thisInterface, Steinberg_int32 paramIndex, struct Steinberg_Vst_ParameterInfo* info) {
    EditControllerInterface* controller = (EditControllerInterface*)thisInterface;
    return GoEditControllerGetParameterInfo(controller->component->goComponent, paramIndex, info);
}

static Steinberg_tresult SMTG_STDMETHODCALLTYPE controller_getParamStringByValue(void* thisInterface, Steinberg_Vst_ParamID id, Steinberg_Vst_ParamValue valueNormalized, Steinberg_Vst_String128 string) {
    EditControllerInterface* controller = (EditControllerInterface*)thisInterface;
    return GoEditControllerGetParamStringByValue(controller->component->goComponent, id, valueNormalized, string);
}

static Steinberg_tresult SMTG_STDMETHODCALLTYPE controller_getParamValueByString(void* thisInterface, Steinberg_Vst_ParamID id, Steinberg_Vst_TChar* string, Steinberg_Vst_ParamValue* valueNormalized) {
    EditControllerInterface* controller = (EditControllerInterface*)thisInterface;
    return GoEditControllerGetParamValueByString(controller->component->goComponent, id, string, valueNormalized);
}

static Steinberg_Vst_ParamValue SMTG_STDMETHODCALLTYPE controller_normalizedParamToPlain(void* thisInterface, Steinberg_Vst_ParamID id, Steinberg_Vst_ParamValue valueNormalized) {
    EditControllerInterface* controller = (EditControllerInterface*)thisInterface;
    return GoEditControllerNormalizedParamToPlain(controller->component->goComponent, id, valueNormalized);
}

static Steinberg_Vst_ParamValue SMTG_STDMETHODCALLTYPE controller_plainParamToNormalized(void* thisInterface, Steinberg_Vst_ParamID id, Steinberg_Vst_ParamValue plainValue) {
    EditControllerInterface* controller = (EditControllerInterface*)thisInterface;
    return GoEditControllerPlainParamToNormalized(controller->component->goComponent, id, plainValue);
}

static Steinberg_Vst_ParamValue SMTG_STDMETHODCALLTYPE controller_getParamNormalized(void* thisInterface, Steinberg_Vst_ParamID id) {
    EditControllerInterface* controller = (EditControllerInterface*)thisInterface;
    return GoEditControllerGetParamNormalized(controller->component->goComponent, id);
}

static Steinberg_tresult SMTG_STDMETHODCALLTYPE controller_setParamNormalized(void* thisInterface, Steinberg_Vst_ParamID id, Steinberg_Vst_ParamValue value) {
    EditControllerInterface* controller = (EditControllerInterface*)thisInterface;
    return GoEditControllerSetParamNormalized(controller->component->goComponent, id, value);
}

static Steinberg_tresult SMTG_STDMETHODCALLTYPE controller_setComponentHandler(void* thisInterface, struct Steinberg_Vst_IComponentHandler* handler) {
    EditControllerInterface* controller = (EditControllerInterface*)thisInterface;
    return GoEditControllerSetComponentHandler(controller->component->goComponent, handler);
}

static struct Steinberg_IPlugView* SMTG_STDMETHODCALLTYPE controller_createView(void* thisInterface, Steinberg_FIDString name) {
    EditControllerInterface* controller = (EditControllerInterface*)thisInterface;
    return GoEditControllerCreateView(controller->component->goComponent, name);
}