# VST3Go Latency Management Implementation TODO

This document provides a detailed implementation guide for the VST3Go latency management system. Each section references the relevant planning documents for additional context and technical details.

## Phase 1: Core Buffer Implementation

### 1.1 Create WriteAheadBuffer Structure
**Reference**: [ring-buffer-explanation.md](./ring-buffer-explanation.md)
- [ ] Create `pkg/dsp/buffer/writeahead.go`
- [ ] Define WriteAheadBuffer struct with:
  - `data []float32` - Pre-allocated circular buffer
  - `readPos uint64` - Atomic read position
  - `writePos uint64` - Atomic write position  
  - `size uint32` - Buffer size (power of 2)
  - `mask uint32` - Size-1 for fast modulo
  - `latencySamples uint32` - Fixed 50ms in samples
- [ ] Implement constructor `NewWriteAheadBuffer(sampleRate float64, channels int)`
  - Calculate latency samples: `50ms * sampleRate / 1000`
  - Set buffer size to 4x latency (nearest power of 2)
  - Pre-allocate buffer with zeros
  - Initialize write position 50ms ahead of read

### 1.2 Implement Core Buffer Operations
**Reference**: [shared-buffer-architecture.md](./shared-buffer-architecture.md)
- [ ] Implement `Write(samples []float32) error`
  - Use atomic.LoadUint64 for position reads
  - Check available space before writing
  - Copy samples with wrap-around handling
  - Update write position atomically
- [ ] Implement `Read(output []float32) int`
  - Enforce minimum read-write gap via `maintainDelay()`
  - Copy samples with wrap-around handling
  - Update read position atomically
  - Return actual samples read
- [ ] Add `maintainDelay()` private method
  - Calculate current gap between read/write
  - Adjust read position if gap < latencySamples
  - Track adjustments for monitoring

### 1.3 Add Buffer Health Monitoring
**Reference**: [latency-enforcement-mechanisms.md](./latency-enforcement-mechanisms.md)
- [ ] Add statistics tracking:
  - `underruns uint64` - Count of buffer underruns
  - `overruns uint64` - Count of buffer overruns
  - `adjustments uint64` - Count of position adjustments
- [ ] Implement `GetBufferHealth() BufferStats`
  - Current fill percentage
  - Underrun/overrun counts
  - Average processing time
- [ ] Add debug/monitoring methods:
  - `GetCurrentLatency() time.Duration`
  - `GetBufferUtilization() float32`

### 1.4 Write Comprehensive Tests
- [ ] Create `pkg/dsp/buffer/writeahead_test.go`
- [ ] Test concurrent read/write operations
- [ ] Test wrap-around behavior
- [ ] Test latency maintenance under stress
- [ ] Benchmark performance vs unbuffered processing

## Phase 2: Plugin Integration Layer

### 2.1 Create BufferedProcessor Wrapper
**Reference**: [GC-compatible-latency-management.md](./GC-compatible-latency-management.md)
- [ ] Create `pkg/plugin/buffered_processor.go`
- [ ] Define BufferedProcessor struct:
  ```go
  type BufferedProcessor struct {
      wrapped Processor
      buffers []*WriteAheadBuffer
      workerCtx context.Context
      workerCancel context.CancelFunc
      midiQueue *MIDIEventQueue
  }
  ```
- [ ] Implement Processor interface methods
- [ ] Add factory function `NewBufferedProcessor(p Processor, sampleRate float64)`

### 2.2 Implement Worker Goroutine
**Reference**: [shared-buffer-architecture.md](./shared-buffer-architecture.md) - Worker Goroutine section
- [ ] Create worker goroutine in constructor
- [ ] Implement adaptive processing loop:
  ```go
  func (bp *BufferedProcessor) processingWorker() {
      ticker := time.NewTicker(5 * time.Millisecond)
      for {
          select {
          case <-ticker.C:
              bp.processAdaptive()
          case <-bp.workerCtx.Done():
              return
          }
      }
  }
  ```
- [ ] Implement `processAdaptive()` method:
  - Check buffer fill levels
  - Process 1-4 chunks based on thresholds
  - Skip tick if buffers > 80% full

### 2.3 Implement MIDI Event Handling
**Reference**: [vst3-latency-communication.md](./vst3-latency-communication.md) - MIDI Synchronization section
- [ ] Create MIDI event queue with timing adjustment
- [ ] Implement `QueueMIDIEvent(event MIDIEvent)`
  - Adjust event offset by latencySamples
  - Queue for processing at correct time
- [ ] Process queued MIDI events in worker
- [ ] Ensure MIDI-audio synchronization

### 2.4 Add Initialization and Cleanup
- [ ] Implement proper initialization sequence:
  - Pre-fill buffers with 50ms of silence
  - Start worker goroutine
  - Wait for buffers to reach target fill
- [ ] Implement cleanup in `Release()`:
  - Stop worker goroutine gracefully
  - Drain remaining audio from buffers
  - Release buffer memory

## Phase 3: VST3 Integration

### 3.1 Update Processor Interface
**Reference**: [vst3-latency-communication.md](./vst3-latency-communication.md)
- [ ] Add to `pkg/framework/plugin/base.go`:
  ```go
  type Processor interface {
      // ... existing methods ...
      GetLatencySamples() uint32
  }
  ```
- [ ] Implement in BaseProcessor (return 0)
- [ ] Implement in BufferedProcessor (return latencySamples)

### 3.2 Implement C Bridge Functions
**Reference**: [vst3-latency-communication.md](./vst3-latency-communication.md) - C Bridge section
- [ ] Add to `bridge/component.c`:
  ```c
  uint32_t goGetLatencySamples(void* goProcessor) {
      return GoGetLatencySamples(goProcessor);
  }
  ```
- [ ] Update `ProcessorVTable` to include latency method
- [ ] Ensure proper calling convention

### 3.3 Add Go Export Functions
**Reference**: [vst3-latency-communication.md](./vst3-latency-communication.md) - Go Export section
- [ ] Add to `pkg/plugin/wrapper_audio.go`:
  ```go
  //export GoGetLatencySamples
  func GoGetLatencySamples(processorPtr unsafe.Pointer) C.uint32_t {
      processor := getProcessor(processorPtr)
      return C.uint32_t(processor.GetLatencySamples())
  }
  ```
- [ ] Update wrapper to detect BufferedProcessor
- [ ] Ensure latency is reported to host

### 3.4 Update Plugin Creation
- [ ] Modify plugin factory to optionally wrap with BufferedProcessor
- [ ] Add configuration option for enabling/disabling buffering
- [ ] Document when buffering should be used

## Phase 4: Enforcement & Testing

### 4.1 Implement Enforcement Mechanisms
**Reference**: [latency-enforcement-mechanisms.md](./latency-enforcement-mechanisms.md)
- [ ] Add timestamp-based sample aging:
  - Track write timestamps for each sample
  - Verify samples are 50ms old before reading
  - Output silence for "future" samples
- [ ] Implement strict position enforcement:
  - Check gap on every read operation
  - Log warnings for violations
  - Auto-correct positions when needed

### 4.2 Create Comprehensive Test Suite
- [ ] Create integration tests:
  - Test with simulated GC pauses
  - Verify no audio glitches during 20ms pauses
  - Test MIDI-audio synchronization
- [ ] Add timing verification tests:
  - Measure actual latency in practice
  - Verify consistent 50ms reporting
  - Test with various sample rates

### 4.3 Performance Benchmarking
- [ ] Create benchmark suite comparing:
  - Buffered vs unbuffered processing
  - CPU usage with buffering
  - Memory allocation patterns
- [ ] Profile GC behavior:
  - Measure pause times with buffering
  - Verify no allocations in audio thread
  - Document performance characteristics

### 4.4 Documentation and Examples
- [ ] Update main documentation with buffering guide
- [ ] Create example plugin using BufferedProcessor
- [ ] Document best practices for GC-friendly processing
- [ ] Add troubleshooting guide for latency issues

## Phase 5: Optimization and Polish

### 5.1 Memory and Cache Optimization
- [ ] Ensure cache-line alignment for atomic variables
- [ ] Optimize memory layout for performance
- [ ] Consider SIMD optimizations for bulk copies

### 5.2 Advanced Features
- [ ] Add dynamic latency adjustment (if needed)
- [ ] Implement buffer size adaptation based on GC behavior
- [ ] Add telemetry for production monitoring

### 5.3 Final Integration Testing
- [ ] Test with popular DAWs (Ableton, Logic, Reaper, etc.)
- [ ] Verify PDC works correctly
- [ ] Test in various plugin hosting scenarios
- [ ] Ensure compatibility with automation

## Implementation Notes

### Priority Order
1. Core buffer implementation (Phase 1) - Foundation
2. VST3 integration (Phase 3.1-3.3) - Critical for host communication
3. Plugin integration (Phase 2) - Makes buffering usable
4. Enforcement & testing (Phase 4) - Ensures reliability
5. Optimization (Phase 5) - Performance tuning

### Key Success Metrics
- Zero audio glitches during 20ms GC pauses
- Consistent 50ms latency reporting
- < 5% CPU overhead from buffering
- Perfect MIDI-audio synchronization

### Risk Mitigation
- Start with conservative buffer sizes
- Add extensive logging during development
- Test with artificial GC pressure
- Have fallback to non-buffered mode