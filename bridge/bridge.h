#ifndef VST3GO_BRIDGE_H
#define VST3GO_BRIDGE_H

#include "../include/vst3/vst3_c_api.h"

// Go callback functions (will be implemented in Go with //export)

// Factory callbacks
extern void GoGetFactoryInfo(char* vendor, char* url, char* email, int32_t* flags);
extern int32_t GoCountClasses();
extern void GoGetClassInfo(int32_t index, char* cid, int32_t* cardinality, char* category, char* name);
extern void* GoCreateInstance(char* cid, char* iid);

// Parameter automation helper functions
int32_t getParameterChangeCount(void* inputParameterChanges);
void* getParameterData(void* inputParameterChanges, int32_t index);
uint32_t getParameterId(void* paramQueue);
int32_t getPointCount(void* paramQueue);
int32_t getPoint(void* paramQueue, int32_t index, int32_t* sampleOffset, double* value);

// Event processing helper functions
int32_t getEventCount(void* eventList);
int32_t getEvent(void* eventList, int32_t index, struct Steinberg_Vst_Event* event);
uint16_t getEventType(struct Steinberg_Vst_Event* event);
struct Steinberg_Vst_NoteOnEvent* getNoteOnEvent(struct Steinberg_Vst_Event* event);
struct Steinberg_Vst_NoteOffEvent* getNoteOffEvent(struct Steinberg_Vst_Event* event);

#endif // VST3GO_BRIDGE_H