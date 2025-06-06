# Developer Tools (Layer 4)

## Overview

The Developer Tools layer provides utilities, templates, and helpers to accelerate VST3 plugin development. This layer focuses on developer experience, making it easy to create, debug, and optimize plugins.

## Components

### Plugin Templates

#### Basic Effect Template
```go
// templates/effect.go
type EffectTemplate struct {
    plugin.Base
    params *param.Registry
    // Your DSP components here
}

func NewEffect() *EffectTemplate {
    e := &EffectTemplate{
        params: param.NewRegistry(),
    }
    
    // Standard effect parameters
    e.params.Add(
        param.New(0, "Input Gain").Range(-24, 24).Default(0).Unit("dB"),
        param.New(1, "Output Gain").Range(-24, 24).Default(0).Unit("dB"),
        param.New(2, "Mix").Range(0, 100).Default(100).Unit("%"),
        param.New(3, "Bypass").Range(0, 1).Default(0).Steps(2),
    )
    
    return e
}
```

#### Instrument Template
```go
// templates/instrument.go
type InstrumentTemplate struct {
    plugin.Base
    voices     [MAX_VOICES]Voice
    freeVoices []int
    usedVoices []int
}

// Voice management built-in
func (i *InstrumentTemplate) NoteOn(note, velocity int) {
    voice := i.allocateVoice()
    if voice != nil {
        voice.Start(note, velocity)
    }
}
```

### Code Generators

#### Parameter Generator
Generate parameter definitions from YAML:
```yaml
# params.yaml
parameters:
  - name: Cutoff
    id: 0
    range: [20, 20000]
    default: 1000
    unit: Hz
    flags: [automatable]
    
  - name: Resonance
    id: 1
    range: [0.1, 10]
    default: 1
    flags: [automatable]
```

Generated code:
```go
// Generated by vst3go-paramgen
const (
    ParamCutoff    = 0
    ParamResonance = 1
)

func createParameters() *param.Registry {
    r := param.NewRegistry()
    r.Add(
        param.New(ParamCutoff, "Cutoff").
            Range(20, 20000).
            Default(1000).
            Unit("Hz").
            Flags(param.CanAutomate),
            
        param.New(ParamResonance, "Resonance").
            Range(0.1, 10).
            Default(1).
            Flags(param.CanAutomate),
    )
    return r
}
```

### Debug Utilities

#### Audio Scope
```go
type AudioScope struct {
    buffer   []float32
    position int
}

func (s *AudioScope) Process(input float32) {
    s.buffer[s.position] = input
    s.position = (s.position + 1) % len(s.buffer)
}

func (s *AudioScope) Dump() {
    // Write to debug file or console
}
```

#### Parameter Logger
```go
type ParamLogger struct {
    file     *os.File
    lastTime time.Time
}

func (l *ParamLogger) LogChange(id ParamID, value float64) {
    if l.file != nil {
        fmt.Fprintf(l.file, "%d,%f,%f\n", 
            time.Since(l.lastTime).Microseconds(),
            id, value)
    }
}
```

#### Performance Profiler
```go
type Profiler struct {
    timings map[string]*Timing
}

type Timing struct {
    count    int64
    totalNs  int64
    minNs    int64
    maxNs    int64
}

func (p *Profiler) Start(name string) func() {
    start := time.Now()
    return func() {
        elapsed := time.Since(start).Nanoseconds()
        p.record(name, elapsed)
    }
}

// Usage in ProcessAudio
func (p *Plugin) ProcessAudio(ctx *process.Context) {
    defer profiler.Start("ProcessAudio")()
    
    defer profiler.Start("Filter")()
    p.filter.Process(ctx.Input[0])
    // ...
}
```

### Development Helpers

#### Hot Reload Support
```go
// hotreload/watcher.go
type PluginWatcher struct {
    path     string
    plugin   plugin.Plugin
    onChange func()
}

func (w *PluginWatcher) Start() {
    // Watch for file changes
    // Reload plugin on change
    // Preserve parameter state
}
```

#### Test Host Integration
```go
// testhost/host.go
type TestHost struct {
    plugin    plugin.Plugin
    processor plugin.Processor
    input     [][]float32
    output    [][]float32
}

func (h *TestHost) ProcessFile(inputPath, outputPath string) error {
    // Load audio file
    // Process through plugin
    // Save output
}

func (h *TestHost) Benchmark() {
    // Measure plugin performance
    // Report allocations
    // Check real-time safety
}
```

### Build Tools

#### Makefile Enhancements
```makefile
# Plugin development targets
.PHONY: dev watch test-audio profile

# Development mode with hot reload
dev:
	@echo "Starting development mode..."
	go run tools/devserver/main.go -plugin=$(PLUGIN)

# Watch for changes and rebuild
watch:
	@echo "Watching for changes..."
	watchman-make -p '**/*.go' -t $(PLUGIN)

# Test with audio files
test-audio:
	go run tools/testhost/main.go \
		-plugin=build/$(PLUGIN).so \
		-input=test/audio/input.wav \
		-output=test/audio/output.wav

# Profile plugin performance
profile:
	go test -bench=. -cpuprofile=cpu.prof
	go tool pprof -http=:8080 cpu.prof
```

#### Cross-Platform Build Script
```go
// tools/build/main.go
func main() {
    platforms := []Platform{
        {"linux", "amd64", ".so"},
        {"darwin", "amd64", ".dylib"},
        {"windows", "amd64", ".dll"},
    }
    
    for _, p := range platforms {
        buildForPlatform(p)
        createBundle(p)
        runValidator(p)
    }
}
```

### Documentation Generators

#### Parameter Documentation
```go
// tools/paramdoc/main.go
func generateParamDocs(registry *param.Registry) {
    fmt.Println("# Parameter Reference")
    fmt.Println()
    
    for _, p := range registry.All() {
        fmt.Printf("## %s (ID: %d)\n", p.Name, p.ID)
        fmt.Printf("- Range: %.2f to %.2f %s\n", p.Min, p.Max, p.Unit)
        fmt.Printf("- Default: %.2f\n", p.DefaultValue)
        fmt.Println()
    }
}
```

#### API Documentation
```go
// tools/apidoc/main.go
// Generates documentation from code comments
// Creates interactive examples
// Exports to various formats
```

### Testing Utilities

#### Audio Test Generators
```go
package testgen

// Generate test signals
func Sine(frequency, sampleRate float64, duration time.Duration) []float32
func WhiteNoise(duration time.Duration) []float32
func Sweep(startFreq, endFreq, sampleRate float64, duration time.Duration) []float32
func Impulse(sampleRate float64) []float32
```

#### Automated Testing
```go
// Test parameter automation
func TestParameterAutomation(t *testing.T, p plugin.Processor) {
    // Create automation curves
    // Verify smooth parameter changes
    // Check for zipper noise
}

// Test state persistence
func TestStatePersistence(t *testing.T, p plugin.Processor) {
    // Save state
    // Modify parameters
    // Restore state
    // Verify restoration
}
```

### Performance Tools

#### Memory Profiler
```go
type MemoryProfiler struct {
    baseline runtime.MemStats
}

func (m *MemoryProfiler) Start() {
    runtime.GC()
    runtime.ReadMemStats(&m.baseline)
}

func (m *MemoryProfiler) Report() {
    var current runtime.MemStats
    runtime.ReadMemStats(&current)
    
    fmt.Printf("Allocations: %d\n", 
        current.Mallocs - m.baseline.Mallocs)
    fmt.Printf("Bytes allocated: %d\n",
        current.TotalAlloc - m.baseline.TotalAlloc)
}
```

#### Real-Time Checker
```go
func CheckRealTimeSafety(p plugin.Processor) []Issue {
    issues := []Issue{}
    
    // Check for allocations
    // Check for system calls
    // Check for locks
    // Check for unbounded loops
    
    return issues
}
```

## CLI Tools

### Plugin Scaffold
```bash
vst3go new effect MyReverb --vendor "My Company"
# Creates:
# - myreverb/
#   - main.go
#   - processor.go
#   - params.go
#   - Makefile
#   - README.md
```

### Parameter Designer
```bash
vst3go params design
# Interactive parameter designer
# Preview parameter ranges
# Test value scaling
# Export to code
```

### Bundle Creator
```bash
vst3go bundle create MyPlugin.so
# Creates proper VST3 bundle structure
# Handles platform differences
# Generates metadata
```

## Integration Examples

### VSCode Extension
```json
// .vscode/tasks.json
{
    "version": "2.0.0",
    "tasks": [
        {
            "label": "Build Plugin",
            "type": "shell",
            "command": "make ${input:plugin}",
            "group": "build"
        },
        {
            "label": "Test Audio",
            "type": "shell",
            "command": "make test-audio PLUGIN=${input:plugin}"
        }
    ]
}
```

### GitHub Actions
```yaml
# .github/workflows/plugin.yml
name: Plugin CI
on: [push, pull_request]

jobs:
  test:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
      - run: make all-examples
      - run: make test-validate
```

## Best Practices

1. **Use Templates**: Start from proven patterns
2. **Profile Early**: Measure performance regularly
3. **Test Automation**: Automated parameter testing
4. **Debug Builds**: Include debugging aids
5. **Documentation**: Generate from code

## Future Tools

- Visual parameter editor
- Plugin analyzer (CPU, memory)
- Automated compatibility testing
- Cloud-based build service
- Plugin marketplace integration