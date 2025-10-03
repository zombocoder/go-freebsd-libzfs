# Contributing to go-freebsd-libzfs

Thank you for your interest in contributing to the FreeBSD ZFS Go library! This document provides guidelines and information for contributors.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Contributing Guidelines](#contributing-guidelines)
- [Code Style](#code-style)
- [Testing](#testing)
- [Submitting Changes](#submitting-changes)
- [Architecture Overview](#architecture-overview)

## Code of Conduct

This project adheres to a code of conduct that we expect all contributors to follow. Please be respectful and constructive in all interactions.

## Getting Started

### Prerequisites

- FreeBSD 13.2+ with OpenZFS support
- Go 1.22+
- Root privileges for testing (ZFS operations require elevated permissions)
- OpenZFS development headers:
  ```bash
  # Ensure you have the source tree or headers available
  ls /usr/src/sys/contrib/openzfs/include/
  ```

### Development Setup

1. **Fork and Clone**
   ```bash
   git clone https://github.com/your-username/go-freebsd-libzfs.git
   cd go-freebsd-libzfs
   ```

2. **Install Dependencies**
   ```bash
   go mod download
   ```

3. **Verify Build**
   ```bash
   make build
   ```

4. **Run Tests**
   ```bash
   # Unit tests (no root required)
   make test
   
   # Integration tests (requires root)
   sudo make integration-test
   ```

## Contributing Guidelines

### What We're Looking For

- **Bug fixes** - Corrections to existing functionality
- **Feature enhancements** - Improvements to current features
- **Documentation** - Better examples, API docs, guides
- **Performance improvements** - Optimizations and benchmarks
- **Test coverage** - Additional test cases and edge case handling

### What We're NOT Looking For

- **Cross-platform support** - This library is FreeBSD-specific by design
- **CLI wrappers** - We provide direct API access, not command-line interfaces
- **Breaking API changes** - Backward compatibility is important

### Areas Needing Contribution

#### High Priority
- **Send/Receive Operations** - ZFS streaming functionality
- **Encryption Support** - Key management and encrypted datasets
- **Event System** - ZFS event subscription and monitoring
- **Performance Benchmarks** - Comprehensive benchmarking suite

#### Medium Priority
- **Additional Examples** - More real-world usage patterns
- **Documentation** - API documentation improvements
- **Error Handling** - Enhanced error types and context
- **Testing** - Edge cases and error conditions

#### Low Priority
- **IOCTL Driver** - Pure Go implementation without CGO
- **Advanced Pool Features** - Additional pool management operations

## Code Style

### Go Code Standards

- Follow standard Go conventions and `gofmt` formatting
- Use meaningful variable and function names
- Write comprehensive comments for exported functions
- Include context.Context in all public APIs
- Use structured error handling with wrapped errors

```go
// Good: Clear, documented function with context
func (c *Client) CreateDataset(ctx context.Context, name string, dsType DatasetType) error {
    if c.d == nil {
        return fmt.Errorf("client is closed")
    }
    
    return c.d.CreateDataset(ctx, name, dsType, nil)
}

// Bad: Missing context, unclear naming
func (c *Client) Create(name string, typ int) error {
    return c.d.CreateDataset(context.Background(), name, DatasetType(typ), nil)
}
```

### CGO Code Standards

- **Memory Safety**: Always use `defer C.free()` for allocated C strings
- **Error Handling**: Propagate libzfs errno and descriptions
- **Resource Cleanup**: Ensure all ZFS handles are properly closed
- **Null Checking**: Always verify C pointers before use

```c
// Good: Proper resource management
int go_zfs_create_example(libzfs_handle_t* hdl, const char* name) {
    zfs_handle_t* zhp = zfs_open(hdl, name, ZFS_TYPE_FILESYSTEM);
    if (zhp == NULL) {
        return -1;
    }
    
    int result = zfs_mount(zhp, NULL, 0);
    zfs_close(zhp);  // Always cleanup
    return result;
}
```

### Package Structure

- **Public APIs**: Only in `/zfs/`, `/zpool/`, `/version/`, `/errors/` packages
- **Internal Implementation**: In `/internal/` packages only
- **Examples**: Self-contained programs in `/examples/`
- **Tests**: Co-located with source files (`*_test.go`)

## Testing

### Test Types

1. **Unit Tests** (`*_test.go`)
   - No external dependencies
   - Test public API contracts
   - Mock internal dependencies
   - Run with `go test ./...`

2. **Integration Tests** (`/integration-tests/`)
   - Require actual ZFS system
   - Test real operations with safety checks
   - Run with root privileges only
   - Use temporary datasets/pools when possible

3. **Example Programs** (`/examples/`)
   - Serve as both documentation and integration tests
   - Must be safe to run on development systems
   - Include error handling and cleanup

### Running Tests

```bash
# Unit tests (safe, no root required)
make test

# Integration tests (requires root, uses real ZFS)
make integration-test

# Build all examples
make examples

# Run specific example
sudo go run examples/clone-operations/main.go
```

### Writing Tests

```go
func TestCreateDataset(t *testing.T) {
    ctx := context.Background()
    
    client, err := zfs.New(ctx)
    if err != nil {
        t.Fatalf("Failed to create client: %v", err)
    }
    defer client.Close()
    
    // Test implementation...
}
```

## Submitting Changes

### Before Submitting

1. **Code Quality**
   ```bash
   # Format code
   go fmt ./...
   
   # Run linter
   golangci-lint run
   
   # Run tests
   make test
   ```

2. **Documentation**
   - Update README.md if adding features
   - Add/update code comments
   - Create example if appropriate
   - Update CHANGELOG.md

3. **Compatibility**
   - Ensure no breaking API changes
   - Test on FreeBSD 13.2+ and 14.x
   - Verify CGO memory management

### Pull Request Process

1. **Create Feature Branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make Changes**
   - Implement feature with tests
   - Update documentation
   - Ensure CI passes

3. **Submit Pull Request**
   - Clear description of changes
   - Reference any related issues
   - Include testing instructions
   - Add examples if applicable

### Pull Request Template

```markdown
## Description
Brief description of changes and motivation.

## Changes Made
- [ ] Feature implementation
- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] Examples created/updated

## Testing
- [ ] Unit tests pass
- [ ] Integration tests pass (if applicable)
- [ ] Manual testing performed

## Compatibility
- [ ] No breaking API changes
- [ ] Backward compatible
- [ ] FreeBSD 13.2+ tested
```

## Architecture Overview

### Package Organization

```
/zfs/           - Public dataset API (filesystems, volumes, snapshots, clones)
/zpool/         - Public pool API (creation, import, export, management)
/version/       - Version detection and capability probing
/errors/        - Structured error types
/internal/
  /driver/      - Driver abstraction layer
    /libzfs_driver.go  - CGO implementation (current)
    /ioctl_driver.go   - Pure Go implementation (future)
  /cgo/         - C code for libzfs integration
/examples/      - Example programs and documentation
/integration-tests/ - Integration test suite
```

### Design Principles

1. **Driver Abstraction**: Support both CGO and pure Go implementations
2. **Type Safety**: Strong typing throughout the API
3. **Context Awareness**: All operations support cancellation
4. **Memory Safety**: Careful CGO resource management
5. **FreeBSD Focus**: No cross-platform compromises

### Adding New Features

1. **C Layer** (`/internal/cgo/zfs_c.go`)
   - Add C wrapper functions
   - Ensure proper error handling
   - Add forward declarations

2. **Driver Interface** (`/internal/driver/driver.go`)
   - Extend interface with new methods
   - Add necessary data structures

3. **Driver Implementation** (`/internal/driver/libzfs_driver.go`)
   - Implement interface methods
   - Handle errors and resource cleanup
   - Add stub to IOCTL driver

4. **Public API** (`/zfs/` or `/zpool/`)
   - Add user-friendly wrapper functions
   - Provide comprehensive documentation
   - Include usage examples

5. **Testing and Documentation**
   - Add unit tests
   - Create integration tests
   - Update documentation
   - Add example programs

## Questions?

- **Issues**: Use GitHub issues for bug reports and feature requests
- **Discussions**: Use GitHub discussions for questions and design discussions
- **Security**: Contact maintainers privately for security issues

Thank you for contributing to go-freebsd-libzfs!