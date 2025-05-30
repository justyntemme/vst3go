# VST3Go Documentation

Welcome to the VST3Go documentation. This guide will help you understand the architecture, design decisions, and implementation details of the VST3Go framework.

## Documentation Structure

### Core Documentation

1. **[Architecture Overview](./architecture.md)**  
   Comprehensive overview of the layered architecture, design principles, and data flow.

2. **[Getting Started Guide](./getting-started.md)**  
   Step-by-step tutorial for creating your first VST3 plugin with VST3Go.

### Layer Documentation

3. **[C Bridge Layer](./c-bridge.md)**  
   Details about the minimal C bridge that interfaces with the VST3 C API.

4. **[Framework Core](./framework-core.md)**  
   Go-idiomatic abstractions for VST3 concepts (parameters, processing, buses).

5. **[DSP Library](./dsp-library.md)**  
   Comprehensive audio processing utilities (filters, oscillators, effects).

6. **[Plugin Wrapper](./plugin-wrapper.md)**  
   The glue layer between C bridge and Go framework.

7. **[Developer Tools](./developer-tools.md)**  
   Templates, generators, and utilities for faster development.

## Quick Links

### For Plugin Developers
- [Getting Started](./getting-started.md) - Create your first plugin
- [DSP Library](./dsp-library.md) - Audio processing components
- [Examples](/examples) - Study working plugins

### For Framework Contributors
- [Architecture](./architecture.md) - Understand the design
- [C Bridge](./c-bridge.md) - Low-level interface details
- [Framework Core](./framework-core.md) - Core abstractions

### References
- [VST3 Documentation](https://steinbergmedia.github.io/vst3_dev_portal/)
- [Go Documentation](https://pkg.go.dev/)
- [Project TODO](/TODO.md) - Development roadmap

## Key Concepts

### Zero Allocations
All memory allocation happens during initialization, not during audio processing. This ensures:
- Predictable real-time performance
- No garbage collection pauses
- Consistent low latency

### Layered Architecture
```
Layer 4: Developer Tools
Layer 3: DSP Library
Layer 2: Framework Core
Layer 1: C Bridge
```

Each layer has a specific purpose and clear boundaries.

### Thread Safety
- Parameters use atomic operations
- Audio processing is lock-free
- Configuration uses appropriate synchronization

## Getting Help

1. **Read the Documentation**: Start with the [Getting Started](./getting-started.md) guide
2. **Study Examples**: Look at the [example plugins](/examples)
3. **Check TODO.md**: See what's being worked on
4. **File an Issue**: Report bugs or request features on GitHub

## Contributing

Before contributing, please read:
- [Architecture Overview](./architecture.md) - Understand the design
- [Guardrails](/guardrails.md) - Development principles
- [TODO](/TODO.md) - Current priorities

## License

VST3Go is licensed under the MIT License. The VST3 SDK headers have their own licensing terms.