package bridge

// This package provides the C bridge layer for VST3
// It should contain zero business logic - just pure C-to-Go mapping

// #cgo CFLAGS: -I../../include
// #include "../../bridge/bridge.c"
// #include "../../bridge/component.c"
import "C"

// This file exists to ensure the C bridge is compiled as part of this package
// All actual Go implementations are in the plugin package
