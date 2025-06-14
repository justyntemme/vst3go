# C Bridge Abstraction

## Overview

The VST3Go framework now provides a clean abstraction over the C bridge implementation. Plugin developers no longer need to directly import C files or deal with CGO directives in their plugin code.

## Previous Approach (Deprecated)

Previously, every plugin had to include C bridge imports directly:

```go
// #cgo CFLAGS: -I../../include
// #include "../../bridge/bridge.c"
// #include "../../bridge/component.c"
import "C"
```

This approach had several issues:
- Exposed internal implementation details
- Made plugins dependent on the bridge file structure
- Complicated refactoring and maintenance
- Violated the principle of abstraction layers

## New Approach

Plugins now use a simple import statement:

```go
import (
    // ... other imports ...
    
    // Import C bridge - required for VST3 plugin to work
    _ "github.com/justyntemme/vst3go/pkg/plugin/cbridge"
)
```

## Benefits

1. **Clean Abstraction**: The C bridge is completely hidden from plugin developers
2. **Better Maintainability**: Bridge implementation can be changed without affecting plugins
3. **Simplified Plugin Code**: No CGO directives or C includes in plugin files
4. **Consistent Pattern**: All plugins use the same import pattern

## Migration Guide

To migrate existing plugins:

1. Remove all C-related comments and imports:
   ```go
   // Remove these lines:
   // #cgo CFLAGS: -I../../include
   // #include "../../bridge/bridge.c"
   // #include "../../bridge/component.c"
   import "C"
   ```

2. Add the cbridge import:
   ```go
   import (
       // ... other imports ...
       _ "github.com/justyntemme/vst3go/pkg/plugin/cbridge"
   )
   ```

## Implementation Details

The `cbridge` package (`pkg/plugin/cbridge/bridge.go`) contains all the necessary C imports and CGO directives. This centralizes the C bridge in one location, making it easier to maintain and modify.

## Future Improvements

This abstraction opens the door for future enhancements:
- Alternative bridge implementations
- Platform-specific optimizations
- Easier testing with mock bridges
- Potential pure-Go implementations

## Compatibility

All existing functionality remains unchanged. The only difference is where the C bridge is imported, not how it works.