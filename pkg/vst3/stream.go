package vst3

// #include "../../include/vst3/vst3_c_api.h"
// #include <stdlib.h>
//
// // Helper functions to work with IBStream
// static inline Steinberg_tresult stream_read(struct Steinberg_IBStream* stream, void* buffer, Steinberg_int32 numBytes, Steinberg_int32* numBytesRead) {
//     return stream->lpVtbl->read(stream, buffer, numBytes, numBytesRead);
// }
//
// static inline Steinberg_tresult stream_write(struct Steinberg_IBStream* stream, void* buffer, Steinberg_int32 numBytes, Steinberg_int32* numBytesWritten) {
//     return stream->lpVtbl->write(stream, buffer, numBytes, numBytesWritten);
// }
//
// static inline Steinberg_tresult stream_seek(struct Steinberg_IBStream* stream, Steinberg_int64 pos, Steinberg_int32 mode, Steinberg_int64* result) {
//     return stream->lpVtbl->seek(stream, pos, mode, result);
// }
//
// static inline Steinberg_tresult stream_tell(struct Steinberg_IBStream* stream, Steinberg_int64* pos) {
//     return stream->lpVtbl->tell(stream, pos);
// }
import "C"
import (
	"encoding/binary"
	"unsafe"
)

// StreamWrapper wraps VST3 IBStream for Go usage
type StreamWrapper struct {
	stream *C.struct_Steinberg_IBStream
}

// NewStreamWrapper creates a wrapper for an IBStream
func NewStreamWrapper(streamPtr unsafe.Pointer) *StreamWrapper {
	if streamPtr == nil {
		return nil
	}
	return &StreamWrapper{
		stream: (*C.struct_Steinberg_IBStream)(streamPtr),
	}
}

// Read reads data from the stream
func (s *StreamWrapper) Read(buffer []byte) (int32, error) {
	if len(buffer) == 0 {
		return 0, nil
	}

	var numBytesRead C.Steinberg_int32
	result := C.stream_read(s.stream, unsafe.Pointer(&buffer[0]), C.Steinberg_int32(len(buffer)), &numBytesRead)
	if result != 0 {
		return 0, ErrNotImplemented
	}
	return int32(numBytesRead), nil
}

// Write writes data to the stream
func (s *StreamWrapper) Write(buffer []byte) (int32, error) {
	if len(buffer) == 0 {
		return 0, nil
	}

	var numBytesWritten C.Steinberg_int32
	result := C.stream_write(s.stream, unsafe.Pointer(&buffer[0]), C.Steinberg_int32(len(buffer)), &numBytesWritten)
	if result != 0 {
		return 0, ErrNotImplemented
	}
	return int32(numBytesWritten), nil
}

// WriteInt32 writes an int32 to the stream
func (s *StreamWrapper) WriteInt32(value int32) error {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(value))
	_, err := s.Write(buf)
	return err
}

// ReadInt32 reads an int32 from the stream
func (s *StreamWrapper) ReadInt32() (int32, error) {
	buf := make([]byte, 4)
	n, err := s.Read(buf)
	if err != nil {
		return 0, err
	}
	if n != 4 {
		return 0, ErrNotImplemented
	}
	return int32(binary.LittleEndian.Uint32(buf)), nil
}

// WriteFloat64 writes a float64 to the stream
func (s *StreamWrapper) WriteFloat64(value float64) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, *(*uint64)(unsafe.Pointer(&value)))
	_, err := s.Write(buf)
	return err
}

// ReadFloat64 reads a float64 from the stream
func (s *StreamWrapper) ReadFloat64() (float64, error) {
	buf := make([]byte, 8)
	n, err := s.Read(buf)
	if err != nil {
		return 0, err
	}
	if n != 8 {
		return 0, ErrNotImplemented
	}
	bits := binary.LittleEndian.Uint64(buf)
	return *(*float64)(unsafe.Pointer(&bits)), nil
}

// WriteString writes a string to the stream with length prefix
func (s *StreamWrapper) WriteString(str string) error {
	// Write length first
	if err := s.WriteInt32(int32(len(str))); err != nil {
		return err
	}
	// Write string data
	if str != "" {
		_, err := s.Write([]byte(str))
		return err
	}
	return nil
}

// ReadString reads a string from the stream with length prefix
func (s *StreamWrapper) ReadString() (string, error) {
	// Read length first
	length, err := s.ReadInt32()
	if err != nil {
		return "", err
	}
	if length == 0 {
		return "", nil
	}
	// Read string data
	buf := make([]byte, length)
	n, err := s.Read(buf)
	if err != nil {
		return "", err
	}
	if n != length {
		return "", ErrNotImplemented
	}
	return string(buf), nil
}

// ReadAll reads all remaining data from the stream
func (s *StreamWrapper) ReadAll() ([]byte, error) {
	if s.stream == nil {
		return nil, ErrNotImplemented
	}

	// Try to determine current position and file size
	var currentPos C.int64_t
	result := C.stream_tell(s.stream, &currentPos)
	if result != 0 {
		// If we can't get position, read in chunks
		return s.readAllChunked()
	}

	// Try to seek to end to get size
	var endPos C.int64_t
	result = C.stream_seek(s.stream, 0, 2, &endPos) // SEEK_END = 2
	if result != 0 {
		return s.readAllChunked()
	}

	// Calculate remaining size
	remaining := int32(endPos - currentPos)
	if remaining <= 0 {
		return []byte{}, nil
	}

	// Seek back to current position
	var newPos C.int64_t
	result = C.stream_seek(s.stream, currentPos, 0, &newPos) // SEEK_SET = 0
	if result != 0 {
		return s.readAllChunked()
	}

	// Read all remaining data
	buffer := make([]byte, remaining)
	n, err := s.Read(buffer)
	if err != nil {
		return nil, err
	}

	return buffer[:n], nil
}

// readAllChunked reads data in chunks when seeking is not supported
func (s *StreamWrapper) readAllChunked() ([]byte, error) {
	var result []byte
	chunkSize := int32(4096)

	for {
		chunk := make([]byte, chunkSize)
		n, err := s.Read(chunk)
		if err != nil {
			return nil, err
		}
		if n == 0 {
			break
		}
		result = append(result, chunk[:n]...)
		if n < chunkSize {
			break // End of stream
		}
	}

	return result, nil
}
