#ifndef VST3GO_BRIDGE_H
#define VST3GO_BRIDGE_H

#include "../include/vst3/vst3_c_api.h"

// Go callback functions (will be implemented in Go with //export)

// Factory callbacks
extern void GoGetFactoryInfo(char* vendor, char* url, char* email, int32_t* flags);
extern int32_t GoCountClasses();
extern void GoGetClassInfo(int32_t index, char* cid, int32_t* cardinality, char* category, char* name);
extern void* GoCreateInstance(const char* cid, const char* iid);

#endif // VST3GO_BRIDGE_H