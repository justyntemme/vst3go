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