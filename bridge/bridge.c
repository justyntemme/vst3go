#include "bridge.h"
#include "component.h"
#include <string.h>
#include <stdlib.h>
#include <stdio.h>

// Debug logging
#ifdef DEBUG_VST3GO
#define DBG_LOG(fmt, ...) fprintf(stderr, "[VST3GO] " fmt "\n", ##__VA_ARGS__)
#else
#define DBG_LOG(fmt, ...)
#endif

// Reference counting for our factory
typedef struct {
    struct Steinberg_IPluginFactoryVtbl* vtbl;
    int refCount;
} PluginFactory;

// Forward declarations for vtable functions
static Steinberg_tresult SMTG_STDMETHODCALLTYPE factory_queryInterface(void* thisInterface, const Steinberg_TUID iid, void** obj);
static Steinberg_uint32 SMTG_STDMETHODCALLTYPE factory_addRef(void* thisInterface);
static Steinberg_uint32 SMTG_STDMETHODCALLTYPE factory_release(void* thisInterface);
static Steinberg_tresult SMTG_STDMETHODCALLTYPE factory_getFactoryInfo(void* thisInterface, struct Steinberg_PFactoryInfo* info);
static Steinberg_int32 SMTG_STDMETHODCALLTYPE factory_countClasses(void* thisInterface);
static Steinberg_tresult SMTG_STDMETHODCALLTYPE factory_getClassInfo(void* thisInterface, Steinberg_int32 index, struct Steinberg_PClassInfo* info);
static Steinberg_tresult SMTG_STDMETHODCALLTYPE factory_createInstance(void* thisInterface, Steinberg_FIDString cid, Steinberg_FIDString iid, void** obj);

// Factory vtable
static struct Steinberg_IPluginFactoryVtbl factoryVtbl = {
    factory_queryInterface,
    factory_addRef,
    factory_release,
    factory_getFactoryInfo,
    factory_countClasses,
    factory_getClassInfo,
    factory_createInstance
};

// Global factory instance
static PluginFactory* globalFactory = NULL;

// VST3 SDK entry point - this is what hosts look for
__attribute__((visibility("default")))
struct Steinberg_IPluginFactory* GetPluginFactory() {
    DBG_LOG("GetPluginFactory called");
    if (!globalFactory) {
        globalFactory = (PluginFactory*)malloc(sizeof(PluginFactory));
        globalFactory->vtbl = &factoryVtbl;
        globalFactory->refCount = 1;
        DBG_LOG("GetPluginFactory: Created factory at %p", globalFactory);
    }
    return (struct Steinberg_IPluginFactory*)globalFactory;
}

// Module initialization state
static int moduleInitialized = 0;

// Linux-specific module entry points
#ifdef __linux__
__attribute__((visibility("default")))
int ModuleEntry(void* sharedLibraryHandle) {
    if (moduleInitialized) {
        return 1; // true
    }
    
    // Module initialization
    // Note: Go runtime is already initialized by the shared library
    moduleInitialized = 1;
    return 1; // true
}

__attribute__((visibility("default")))
int ModuleExit() {
    // Module cleanup - don't free the factory here as it has its own reference counting
    moduleInitialized = 0;
    return 1; // true
}
#endif

// IUnknown implementation
static Steinberg_tresult SMTG_STDMETHODCALLTYPE factory_queryInterface(void* thisInterface, const Steinberg_TUID iid, void** obj) {
    if (memcmp(iid, Steinberg_FUnknown_iid, sizeof(Steinberg_TUID)) == 0 ||
        memcmp(iid, Steinberg_IPluginFactory_iid, sizeof(Steinberg_TUID)) == 0) {
        *obj = thisInterface;
        factory_addRef(thisInterface);
        return ((Steinberg_tresult)0);
    }
    *obj = NULL;
    return ((Steinberg_tresult)-1);
}

static Steinberg_uint32 SMTG_STDMETHODCALLTYPE factory_addRef(void* thisInterface) {
    PluginFactory* factory = (PluginFactory*)thisInterface;
    return ++factory->refCount;
}

static Steinberg_uint32 SMTG_STDMETHODCALLTYPE factory_release(void* thisInterface) {
    PluginFactory* factory = (PluginFactory*)thisInterface;
    if (--factory->refCount == 0) {
        free(factory);
        return 0;
    }
    return factory->refCount;
}

// IPluginFactory implementation - these will call into Go
static Steinberg_tresult SMTG_STDMETHODCALLTYPE factory_getFactoryInfo(void* thisInterface, struct Steinberg_PFactoryInfo* info) {
    GoGetFactoryInfo(info->vendor, info->url, info->email, &info->flags);
    return ((Steinberg_tresult)0);
}

static Steinberg_int32 SMTG_STDMETHODCALLTYPE factory_countClasses(void* thisInterface) {
    return GoCountClasses();
}

static Steinberg_tresult SMTG_STDMETHODCALLTYPE factory_getClassInfo(void* thisInterface, Steinberg_int32 index, struct Steinberg_PClassInfo* info) {
    if (index >= GoCountClasses()) {
        return ((Steinberg_tresult)1);
    }
    GoGetClassInfo(index, (char*)info->cid, &info->cardinality, info->category, info->name);
    return ((Steinberg_tresult)0);
}

static Steinberg_tresult SMTG_STDMETHODCALLTYPE factory_createInstance(void* thisInterface, Steinberg_FIDString cid, Steinberg_FIDString iid, void** obj) {
    DBG_LOG("factory_createInstance called");
    // Create instance through Go
    void* instance = GoCreateInstance(cid, iid);
    if (!instance) {
        DBG_LOG("factory_createInstance: GoCreateInstance returned NULL");
        *obj = NULL;
        return ((Steinberg_tresult)-1); // kNoInterface
    }
    
    DBG_LOG("factory_createInstance: Created instance at %p", instance);
    *obj = instance;
    return ((Steinberg_tresult)0);
}

// Parameter automation helper functions implementation

int32_t getParameterChangeCount(void* inputParameterChanges) {
    if (!inputParameterChanges) {
        DBG_LOG("getParameterChangeCount: inputParameterChanges is NULL");
        return 0;
    }
    
    struct Steinberg_Vst_IParameterChanges* changes = (struct Steinberg_Vst_IParameterChanges*)inputParameterChanges;
    if (!changes->lpVtbl || !changes->lpVtbl->getParameterCount) {
        DBG_LOG("getParameterChangeCount: vtable or method is NULL");
        return 0;
    }
    
    int32_t count = changes->lpVtbl->getParameterCount(changes);
    // DBG_LOG("getParameterChangeCount: returning %d parameters", count);
    return count;
}

void* getParameterData(void* inputParameterChanges, int32_t index) {
    if (!inputParameterChanges) {
        DBG_LOG("getParameterData: inputParameterChanges is NULL");
        return NULL;
    }
    
    struct Steinberg_Vst_IParameterChanges* changes = (struct Steinberg_Vst_IParameterChanges*)inputParameterChanges;
    if (!changes->lpVtbl || !changes->lpVtbl->getParameterData) {
        DBG_LOG("getParameterData: vtable or method is NULL");
        return NULL;
    }
    
    struct Steinberg_Vst_IParamValueQueue* queue = changes->lpVtbl->getParameterData(changes, index);
    DBG_LOG("getParameterData: index=%d, returning queue=%p", index, queue);
    return queue;
}

uint32_t getParameterId(void* paramQueue) {
    if (!paramQueue) {
        DBG_LOG("getParameterId: paramQueue is NULL");
        return 0;
    }
    
    struct Steinberg_Vst_IParamValueQueue* queue = (struct Steinberg_Vst_IParamValueQueue*)paramQueue;
    if (!queue->lpVtbl || !queue->lpVtbl->getParameterId) {
        DBG_LOG("getParameterId: vtable or method is NULL");
        return 0;
    }
    
    uint32_t paramId = queue->lpVtbl->getParameterId(queue);
    DBG_LOG("getParameterId: returning paramId=%u", paramId);
    return paramId;
}

int32_t getPointCount(void* paramQueue) {
    if (!paramQueue) {
        DBG_LOG("getPointCount: paramQueue is NULL");
        return 0;
    }
    
    struct Steinberg_Vst_IParamValueQueue* queue = (struct Steinberg_Vst_IParamValueQueue*)paramQueue;
    if (!queue->lpVtbl || !queue->lpVtbl->getPointCount) {
        DBG_LOG("getPointCount: vtable or method is NULL");
        return 0;
    }
    
    int32_t count = queue->lpVtbl->getPointCount(queue);
    DBG_LOG("getPointCount: returning %d points", count);
    return count;
}

int32_t getPoint(void* paramQueue, int32_t index, int32_t* sampleOffset, double* value) {
    if (!paramQueue) {
        DBG_LOG("getPoint: paramQueue is NULL");
        return 1; // kResultFalse
    }
    
    if (!sampleOffset || !value) {
        DBG_LOG("getPoint: sampleOffset or value pointer is NULL");
        return 1; // kResultFalse
    }
    
    struct Steinberg_Vst_IParamValueQueue* queue = (struct Steinberg_Vst_IParamValueQueue*)paramQueue;
    if (!queue->lpVtbl || !queue->lpVtbl->getPoint) {
        DBG_LOG("getPoint: vtable or method is NULL");
        return 1; // kResultFalse
    }
    
    // VST3 uses ParamValue which is double
    Steinberg_Vst_ParamValue vstValue;
    Steinberg_tresult result = queue->lpVtbl->getPoint(queue, index, sampleOffset, &vstValue);
    
    if (result == 0) { // kResultOk
        *value = vstValue;
        DBG_LOG("getPoint: index=%d, sampleOffset=%d, value=%.6f", index, *sampleOffset, *value);
    } else {
        DBG_LOG("getPoint: failed with result=%d", result);
    }
    
    return result;
}

// Event processing helper functions
int32_t getEventCount(void* eventList) {
    if (!eventList) {
        // DBG_LOG("getEventCount: eventList is NULL");
        return 0;
    }
    
    struct Steinberg_Vst_IEventList* list = (struct Steinberg_Vst_IEventList*)eventList;
    if (!list->lpVtbl || !list->lpVtbl->getEventCount) {
        // DBG_LOG("getEventCount: vtable or method is NULL");
        return 0;
    }
    
    int32_t count = list->lpVtbl->getEventCount(list);
    // DBG_LOG("getEventCount: returning %d events", count);
    return count;
}

int32_t getEvent(void* eventList, int32_t index, struct Steinberg_Vst_Event* event) {
    if (!eventList || !event) {
        DBG_LOG("getEvent: eventList or event is NULL");
        return 1; // kResultFalse
    }
    
    struct Steinberg_Vst_IEventList* list = (struct Steinberg_Vst_IEventList*)eventList;
    if (!list->lpVtbl || !list->lpVtbl->getEvent) {
        DBG_LOG("getEvent: vtable or method is NULL");
        return 1; // kResultFalse
    }
    
    Steinberg_tresult result = list->lpVtbl->getEvent(list, index, event);
    if (result == 0) { // kResultOk
        DBG_LOG("getEvent: got event at index %d, type=%d", index, event->type);
    } else {
        DBG_LOG("getEvent: failed with result=%d", result);
    }
    
    return result;
}

uint16_t getEventType(struct Steinberg_Vst_Event* event) {
    return event->type;
}

struct Steinberg_Vst_NoteOnEvent* getNoteOnEvent(struct Steinberg_Vst_Event* event) {
    return &event->Steinberg_Vst_Event_noteOn;
}

struct Steinberg_Vst_NoteOffEvent* getNoteOffEvent(struct Steinberg_Vst_Event* event) {
    return &event->Steinberg_Vst_Event_noteOff;
}
