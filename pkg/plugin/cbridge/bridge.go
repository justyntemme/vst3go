// Package cbridge provides the C bridge imports required for VST3 plugins.
// This package should be imported by plugin implementations to enable VST3 functionality.
//
// Usage:
//   import _ "github.com/justyntemme/vst3go/pkg/plugin/cbridge"
//
// The underscore import ensures the C bridge is linked without directly using any exports.
package cbridge

// #cgo CFLAGS: -I../../../include
// #include "../../../bridge/bridge.c"
// #include "../../../bridge/component.c"
import "C"