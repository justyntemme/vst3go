# vst3gi Development Guardrails

Critical constraints for maintaining architectural integrity in development.

## 🚫 Architecture Anti-Patterns (NEVER)

### 1. C Bridge Complexity (Keep C Bridge Simple)
- C bridge code should be minimal, thin wrappers only
- No business logic in C bridge layer
- No framework features in C bridge layer
- Go-side registration systems belong in Go framework layers, not C bridge

### 2. Over-Abstraction
- Don't hide vst3 concepts so deeply that they become inaccessible
- Avoid creating proprietary abstractions when Go/vst3 idioms work better
- Don't force developers into rigid patterns when flexibility is needed
- Preserve escape hatches to underlying vst3 functionality

### 3. Placeholder Code
- No incomplete implementations with placeholder comments
- Either implement fully or return `nil` for unsupported extensions

### 4. Backwards Compatibility
- No API versioning for internal changes
- Make breaking changes without deprecation
- Delete old code entirely when refactoring

## ✅ Required Patterns

``## 3. C Bridge Architecture (Layer 1 Only)
- **Bridge Philosophy**: Minimal, direct mapping between vst3 C API and Go
- **No Business Logic**: C bridge contains zero business logic or framework features
- **Direct Mapping**: Each vst3 C function maps to exactly one Go function call
- **Manifest-Driven**: Plugin discovery uses JSON manifests + minimal C bridge

### 4. Go Framework Architecture (Layers 2-4)
- **Layer 2**: Go-native abstractions that feel natural to Go developers
- **Layer 3**: Rich DSP and audio processing utilities
- **Layer 4**: Developer convenience tools (builders, generators, templates)

### 5. Framework Convenience
- Provide sensible defaults that can be overridden
- Make common tasks simple, complex tasks possible
- Extract patterns into reusable, composable components
- Include comprehensive DSP utilities and audio processing helpers
- Generate boilerplate code rather than making developers write it

## 🔒 Build & Development Standards

### Build System
- Use `make install` exclusively (never CMake)
- Test with `validator`
- All example plugins should be maintained and tested

### Code Quality
- No placeholder implementations
- Complete error handling (no silent failures)
- Thread-safe parameter access

### POC Development
- Breaking changes encouraged to find right architecture
- Update existing examples instead of creating new ones
- Delete old code entirely when refactoring

## 🎯 Architecture Goals

**Primary**: vst3Go is a Go-native framework that simplifies VST3 plugin development  
**Secondary**: Zero plugin-specific C code required  
**Tertiary**: Layered architecture - simple bridge foundation with helpful abstractions on top

**Framework Philosophy**:
- **Layer 1**: Direct vst3 bridge (minimal C bridge, not framework)
- **Layer 2**: Go-idiomatic abstractions (param management, event handling, state)
- **Layer 3**: High-level helpers (DSP utilities, common audio patterns)
- **Layer 4**: Developer conveniences (builders, templates, generators)

**C Bridge vs Go Framework Distinction**:
- **C Bridge (Layer 1)**: Simple, thin, minimal - just bridges vst3 C API to Go
- **Go Framework (Layers 2-4)**: Rich, comprehensive, developer-friendly

**Framework Goals**:
- Make audio processing development faster and more enjoyable for Go developers
- Abstract away tedious vst3 boilerplate while preserving full access to underlying concepts
- Provide rich DSP and audio processing utilities out of the box
- Enable developers to focus on their creative audio algorithms, not infrastructure

**Anti-Goals**:
- Creating a rigid framework that boxes developers into specific patterns
- Hiding so much complexity that debugging becomes impossible
- Forcing developers to learn framework-specific concepts instead of standard audio development
- Over-abstracting to the point where simple tasks become complicated

## 🎵 Framework Features & DSP Package

### DSP Package Goals
- Provide common audio processing building blocks (filters, oscillators, envelopes)
- Include standard audio utility functions (gain, mixing, format conversion)
- Offer high-performance, zero-allocation audio processing primitives
- Support both real-time and offline audio processing workflows
- Enable rapid prototyping of audio effects and instruments

### Framework Convenience Features

### Developer Experience Priorities
1. **Fast Iteration**: Changes to audio processing code should compile and test quickly
2. **Clear Debugging**: Audio processing issues should be easy to diagnose and fix
3. **Performance**: Framework overhead should be minimal in audio processing paths
4. **Documentation**: Comprehensive examples and API documentation
5. **Extensibility**: Developers can drop down to lower layers when needed

## 📦 Package Organization Principles

### Domain-Driven Design
- Group related types and functions by domain (like Go's `net`, `io`, `http`)
- Keep types and their methods in the same package
- Avoid generic "helpers" or "utils" packages

### Naming Conventions
- Package names should be short, lowercase, singular nouns
- Avoid redundancy: `param.Info` not `param.ParamInfo`
- Functions should read naturally: `param.Format()` not `FormatParameter()`

## 🐹 Go Idiom Requirements

### Error Handling
- Return `error` not `bool` for operations that can fail
- Use custom error types with `Unwrap()` support
- Wrap errors with context: `fmt.Errorf("operation failed: %w", err)`

### Interface Design
- Keep interfaces small and focused (1-3 methods ideal)
- Accept interfaces, return concrete types
- Use standard library interfaces where applicable (`io.Reader`, `io.Writer`)

### API Design Patterns
- Use functional options for extensible APIs
- Builder pattern for complex struct creation
- Context support for cancellation/timeouts
- Method chaining where it improves readability

### Concurrency
- Protect shared state with appropriate synchronization
- Use channels for communication between goroutines
- Design APIs to be safe for concurrent use

## 🚨 Red Flags - Stop If You See

1. **Adding complexity to C bridge layer** (business logic, framework features, etc.)
2. Over-abstracting to the point where vst3 concepts become inaccessible
3. Adding Go registration when manifests exist  
4. Writing TODO comments
5. Worrying about backwards compatibility
6. Creating rigid patterns that box developers in
7. Bypassing the layered architecture (jumping from Layer 4 to Layer 1)
8. Duplicating code between examples
9. Implementing framework functionality in examples instead of packages
10. Copy-pasting code instead of creating reusable components
11. Making simple audio tasks unnecessarily complicated

**When in doubt**: Does this make audio development in Go faster and more enjoyable?

## ❌ Code Duplication Anti-Patterns

### NEVER duplicate these between plugins:
- Parameter creation/management boilerplate
- State saving/loading logic
- Event processing patterns
- Extension initialization
- Common DSP operations
- Error handling patterns
- Logging setup

### Examples should ONLY contain:
- Plugin-specific constants (ID, name, version)
- Unique audio processing algorithms
- Custom parameter behavior
- Plugin-specific UI or visualization logic

### Framework Development Guidelines:

#### If you find yourself:
- **Copy-pasting between examples** → Extract to framework package
- **Writing complex boilerplate** → Create builder or template
- **Repeating initialization patterns** → Add to base plugin class
- **Implementing common DSP** → Add to dsp package
- **Solving the same problem twice** → Create reusable component

#### When adding framework features:
- **Start simple**: Solve the immediate need first
- **Make it reusable**: Consider how others might use it differently  
- **Provide escape hatches**: Allow access to lower layers when needed
- **Document with examples**: Show both simple and advanced usage
- **Performance matters**: Audio processing code should be efficient

#### Framework evolution:
- **Listen to users**: Framework should solve real developer pain points
- **Iterate quickly**: It's better to ship something useful and improve it
- **Stay modular**: Features should be composable, not monolithic
- **Preserve choice**: Don't force developers into specific patterns

---

## 🎯 **Key Architectural Principle Summary**

### C Bridge Layer (Layer 1): "Bridge Not Framework"
- **Purpose**: Minimal, direct vst3 C API to Go mapping
- **Philosophy**: Thin, simple, generated wrappers only
- **NO**: Business logic, framework features, abstractions
- **YES**: Direct function mapping, manifest discovery, minimal C interop

### Go Framework Layers (Layers 2-4): "Rich Framework"
- **Purpose**: Comprehensive, developer-friendly audio development framework
- **Philosophy**: Make Go audio development fast, enjoyable, and powerful
- **YES**: Rich abstractions, DSP utilities, builders, templates, convenience APIs
- **YES**: Business logic, state management, parameter builders, audio processing helpers

**The distinction is critical**: Keep the C bridge minimal and direct, while making the Go framework rich and comprehensive.
