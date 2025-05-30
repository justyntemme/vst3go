package vst3

// #include "../../include/vst3/vst3_c_api.h"
import "C"
import "unsafe"

// Result codes - we need to define these as values, not C constants
const (
	ResultOK    = 0
	ResultOk    = 0
	ResultFalse = 1
	ResultTrue  = 0 // Same as Ok
)

// Basic type aliases
type (
	Result     = C.Steinberg_tresult
	TUID       = C.Steinberg_TUID
	FUnknown   = C.struct_Steinberg_FUnknown
	ParamValue = C.Steinberg_Vst_ParamValue
	ParamID    = C.Steinberg_Vst_ParamID
	Sample32   = C.Steinberg_Vst_Sample32
	Sample64   = C.Steinberg_Vst_Sample64
)

// Interface IDs
var (
	IIDFUnknown = [16]byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46,
	}
	IIDIPluginFactory = [16]byte{
		0x7A, 0x4D, 0x81, 0x1C, 0x52, 0x11, 0x4A, 0x1F,
		0xAE, 0xD9, 0xD2, 0xEE, 0x0B, 0x43, 0xBF, 0x9F,
	}
)

// Class categories
const (
	CategoryAudioEffect = "Audio Module Class"
)

// Error codes
type Error int

const (
	ErrNotImplemented  Error = -1
	ErrInvalidArgument Error = -2
)

func (e Error) Error() string {
	switch e {
	case ErrNotImplemented:
		return "not implemented"
	default:
		return "unknown error"
	}
}

// Helper to convert Go interface ID to C TUID
func ToTUID(iid [16]byte) unsafe.Pointer {
	return unsafe.Pointer(&iid[0])
}
