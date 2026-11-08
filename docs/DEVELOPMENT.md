# Development Guide

This guide provides detailed information for developers working on the go-freebsd-libzfs library.

## Table of Contents

- [Development Environment](#development-environment)
- [Architecture Overview](#architecture-overview)
- [Adding New Features](#adding-new-features)
- [CGO Integration](#cgo-integration)
- [Testing Strategy](#testing-strategy)
- [Debugging](#debugging)
- [Performance Considerations](#performance-considerations)
- [Release Process](#release-process)

## Development Environment

### Prerequisites

- FreeBSD 14 or 15 (OpenZFS 2.2+)
- Go 1.22+
- Git
- Development tools: `make`, `gcc`
- Root access for testing (ZFS operations require privileges)

### Setup

1. **Install OpenZFS Development Headers**

   ```bash
   # Check whether the headers are available
   ls /usr/src/sys/contrib/openzfs/include/libzfs.h

   # If not, extract them from the src.txz of the running release
   # (see CONTRIBUTING.md), or let the CI script do it:
   doas sh .github/scripts/freebsd-ci-prepare.sh
   ```

2. **Clone and Setup**

   ```bash
   git clone https://github.com/zombocoder/go-freebsd-libzfs.git
   cd go-freebsd-libzfs
   go mod download
   ```

3. **Verify Build**
   ```bash
   make build
   make test
   ```

### Development Tools

- **Editor**: VS Code with Go extension recommended
- **Linter**: `go vet` (run by CI)
- **Formatter**: `gofmt` and `goimports`
- **Documentation**: `godoc`

## Architecture Overview

### Package Structure

```
/
├── zfs/              # Public ZFS dataset API
├── zpool/            # Public ZFS pool API
├── version/          # Version detection
├── errors/           # Error types
├── internal/
│   ├── driver/       # Driver abstraction
│   └── cgo/          # C integration layer
├── examples/         # Example programs
├── docs/             # Documentation
└── .github/          # CI workflow and scripts
```

### Driver Abstraction

The library uses a driver pattern to support multiple implementations:

```go
type Driver interface {
    // Core operations
    ListDatasets(ctx context.Context, recursive bool) ([]Dataset, error)
    CreateDataset(ctx context.Context, name string, dsType DatasetType, props map[string]string) error

    // Clone operations
    CreateClone(ctx context.Context, snapshot, clone string, props map[string]string) error
    PromoteClone(ctx context.Context, clone string) error

    // Property operations
    GetProperty(ctx context.Context, dataset, property string) (*Property, error)
    SetProperty(ctx context.Context, dataset, property, value string) error
}
```

**Current Implementations:**

- `libzfsDriver`: Production CGO implementation using libzfs
- `ioctlDriver`: Future pure-Go implementation (planned)

### Data Flow

```
Public API (zfs/zfs.go)
       ↓
Driver Interface (internal/driver/driver.go)
       ↓
CGO Implementation (internal/driver/libzfs_driver.go)
       ↓
C Wrapper Functions (internal/cgo/zfs_c.go)
       ↓
libzfs Library
```

## Adding New Features

### Step-by-Step Process

1. **Define C Wrapper Functions** (`internal/cgo/zfs_c.go`)
2. **Extend Driver Interface** (`internal/driver/driver.go`)
3. **Implement libzfs Driver** (`internal/driver/libzfs_driver.go`)
4. **Add Stub to ioctl Driver** (`internal/driver/ioctl_driver.go`)
5. **Create Public API** (`zfs/zfs.go` or `zpool/zpool.go`)
6. **Add Tests and Examples**

### Example: Adding a New Operation

Let's add a "rename dataset" feature:

#### 1. C Wrapper Function

```c
// internal/cgo/zfs_c.go
int go_zfs_rename(libzfs_handle_t* hdl, const char* oldname, const char* newname, int flags) {
    zfs_handle_t* zhp = zfs_open(hdl, oldname, ZFS_TYPE_DATASET);
    if (zhp == NULL) {
        return -1;
    }

    int ret = zfs_rename(zhp, newname, flags);
    zfs_close(zhp);
    return ret;
}
```

#### 2. Driver Interface

```go
// internal/driver/driver.go
type Driver interface {
    // ... existing methods
    RenameDataset(ctx context.Context, oldName, newName string, force bool) error
}
```

#### 3. libzfs Implementation

```go
// internal/driver/libzfs_driver.go
func (d *libzfsDriver) RenameDataset(ctx context.Context, oldName, newName string, force bool) error {
    if d.handle == nil {
        return fmt.Errorf("driver is closed")
    }

    cOldName := C.CString(oldName)
    defer C.free(unsafe.Pointer(cOldName))

    cNewName := C.CString(newName)
    defer C.free(unsafe.Pointer(cNewName))

    flags := C.int(0)
    if force {
        flags |= C.ZFS_RENAME_FORCE
    }

    ret := C.go_zfs_rename(d.handle, cOldName, cNewName, flags)
    if ret != 0 {
        return fmt.Errorf("failed to rename dataset %s to %s: %w",
            oldName, newName, d.getLastError())
    }

    return nil
}
```

#### 4. ioctl Stub

```go
// internal/driver/ioctl_driver.go
func (d *ioctlDriver) RenameDataset(ctx context.Context, oldName, newName string, force bool) error {
    return fmt.Errorf("ioctl driver not implemented")
}
```

#### 5. Public API

```go
// zfs/zfs.go
func (c *Client) RenameDataset(ctx context.Context, oldName, newName string, force bool) error {
    if c.d == nil {
        return fmt.Errorf("client is closed")
    }

    return c.d.RenameDataset(ctx, oldName, newName, force)
}
```

## CGO Integration

### Memory Management Rules

1. **Always free C strings**: Use `defer C.free()`
2. **Close ZFS handles**: Prevent resource leaks
3. **Check null pointers**: Validate before use
4. **Handle errors properly**: Convert errno to Go errors

### Example Pattern

```go
func (d *libzfsDriver) exampleOperation(name string) error {
    // Convert Go string to C string
    cName := C.CString(name)
    defer C.free(unsafe.Pointer(cName)) // Critical: free memory

    // Open ZFS handle
    zhp := C.zfs_open(d.handle, cName, C.ZFS_TYPE_DATASET)
    if zhp == nil {
        return d.getLastError() // Handle error
    }
    defer C.zfs_close(zhp) // Critical: close handle

    // Perform operation
    ret := C.zfs_some_operation(zhp)
    if ret != 0 {
        return d.getLastError()
    }

    return nil
}
```

### Error Handling

```go
func (d *libzfsDriver) getLastError() error {
    errno := C.libzfs_errno(d.handle)
    msg := C.GoString(C.libzfs_error_description(d.handle))

    return &errors.ZFSError{
        Errno:   int(errno),
        Message: msg,
    }
}
```

### nvlist Handling

```go
func createNVList(props map[string]string) (*C.nvlist_t, error) {
    var nvl *C.nvlist_t

    ret := C.nvlist_alloc(&nvl, C.NV_UNIQUE_NAME, 0)
    if ret != 0 {
        return nil, fmt.Errorf("failed to allocate nvlist")
    }

    for key, value := range props {
        cKey := C.CString(key)
        cValue := C.CString(value)

        ret := C.nvlist_add_string(nvl, cKey, cValue)

        C.free(unsafe.Pointer(cKey))
        C.free(unsafe.Pointer(cValue))

        if ret != 0 {
            C.nvlist_free(nvl)
            return nil, fmt.Errorf("failed to add property %s", key)
        }
    }

    return nvl, nil
}
```

## Testing Strategy

Tests live next to the code they test (`*_test.go`) and come in three kinds:

1. **Unit tests** — pure Go, no ZFS needed (e.g. `errors`, string mappings in
   `internal/driver`).
2. **Read-only integration tests** — run against the first imported pool and
   compare the library with the `zfs`/`zpool` commands (GUIDs, property
   sources, listings, versions). They run as any user and skip without pools.
   Examples: `zfs/zfs_integration_test.go`, `zpool/zpool_integration_test.go`.
3. **Destructive integration tests** — create their own file-backed pools
   (`gozfs_*` under `/var/tmp`) and export, import, offline, corrupt, scrub and
   destroy them. They run only as root with `ZFS_TEST_DESTRUCTIVE=1`.
   Examples: `zpool/zpool_import_test.go`, `zpool_discover_test.go`,
   `zpool_status_test.go`.

```bash
make test                     # unit + read-only
doas make test-destructive    # everything, packages one at a time
```

### Destructive test pattern

```go
func TestImportExportedPool(t *testing.T) {
	p := newTestPool(t) // skips unless root + ZFS_TEST_DESTRUCTIVE=1; cleans up
	p.export(t)

	err := newClient(t).Import(context.Background(), p.name,
		ImportOptions{SearchPaths: []string{p.dir}})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if !poolImported(p.name) {
		t.Fatalf("pool %s not imported", p.name)
	}
}
```

`createTestPool(t, prefix, disks, mirror)` builds multi-disk and mirror pools;
`run(t, cmd, args...)` runs a command and fails the test on error. Scan states
can be observed deterministically by freezing scans with
`vfs.zfs.scan_suspend_progress` (see `suspendScans`).

CI runs all three kinds on FreeBSD 14.5 and 15.1 VMs, see
[CONTRIBUTING.md](../CONTRIBUTING.md#continuous-integration).

## Debugging

### CGO Debugging

1. **Enable Debug Output**

   ```bash
   export CGO_CFLAGS="-g -O0"
   export CGO_LDFLAGS="-g"
   go build -gcflags="all=-N -l"
   ```

2. **Use GDB**
   ```bash
   gdb ./your-program
   (gdb) set environment CGO_CFLAGS="-g -O0"
   (gdb) run
   ```

### ZFS Error Debugging

```go
func debugZFSError(err error) {
    if zfsErr, ok := err.(*errors.ZFSError); ok {
        fmt.Printf("ZFS Error Details:\n")
        fmt.Printf("  Errno: %d\n", zfsErr.Errno)
        fmt.Printf("  Message: %s\n", zfsErr.Message)
        fmt.Printf("  Operation: %s\n", zfsErr.Operation)
    }
}
```

### Tracing Function Calls

```go
func (d *libzfsDriver) CreateDataset(ctx context.Context, name string, dsType DatasetType, props map[string]string) error {
    start := time.Now()
    defer func() {
        duration := time.Since(start)
        log.Printf("CreateDataset(%s, %s) took %v", name, dsType, duration)
    }()

    // Implementation...
}
```

## Performance Considerations

### Memory Usage

1. **Limit concurrent operations**: Too many concurrent ZFS operations can exhaust memory
2. **Close handles promptly**: Don't hold ZFS handles longer than necessary
3. **Batch operations**: Group related operations when possible

### CGO Overhead

1. **Minimize CGO calls**: Batch multiple properties in single call
2. **Reuse connections**: Keep clients open for multiple operations
3. **Avoid frequent switches**: Stay in Go or C domain when possible

### Example: Efficient Property Retrieval

```go
// Inefficient: Multiple CGO calls
compression, _ := client.GetStringProperty(ctx, dataset, "compression")
recordsize, _ := client.GetStringProperty(ctx, dataset, "recordsize")
used, _ := client.GetStringProperty(ctx, dataset, "used")

// Efficient: Single CGO call
props, _ := client.GetProperties(ctx, dataset, "compression", "recordsize", "used")
compression := props["compression"].Value
recordsize := props["recordsize"].Value
used := props["used"].Value
```

## Release Process

### Version Management

Use semantic versioning (semver):

- `v1.0.0`: Major release with breaking changes
- `v1.1.0`: Minor release with new features
- `v1.0.1`: Patch release with bug fixes

### Pre-release Checklist

1. **Code Quality**

   - [ ] All tests pass
   - [ ] CI is green on all FreeBSD versions
   - [ ] Documentation updated

2. **Testing**

   - [ ] Unit tests pass
   - [ ] Integration tests pass on target FreeBSD versions
   - [ ] Example programs work correctly
   - [ ] Performance regressions checked

3. **Documentation**
   - [ ] API documentation updated
   - [ ] CHANGELOG.md updated
   - [ ] README.md reflects new features
   - [ ] Examples updated if needed

### Release Commands

```bash
# Update version
git tag v1.2.0

# Create release notes
git log --oneline v1.1.0..v1.2.0 > RELEASE_NOTES.md

# Push tag
git push origin v1.2.0

# Create GitHub release
gh release create v1.2.0 -F RELEASE_NOTES.md
```

### Go Module Publishing

The library is automatically available through Go modules:

```bash
# Users can get specific version
go get github.com/zombocoder/go-freebsd-libzfs@v1.2.0

# Or latest
go get github.com/zombocoder/go-freebsd-libzfs@latest
```

## Common Issues and Solutions

### Build Issues

**Problem**: CGO compilation fails with missing headers

```
fatal error: 'libzfs.h' file not found
```

**Solution**: Install the OpenZFS headers from the `src.txz` of the running
release (see [CONTRIBUTING.md](../CONTRIBUTING.md#openzfs-headers)), then verify:

```bash
ls /usr/src/sys/contrib/openzfs/include/libzfs.h
```

**Problem**: Linker errors for libzfs functions

```
undefined reference to `zfs_create'
```

**Solution**: Ensure libzfs is installed and linked correctly

```bash
# Check library exists
ls -la /usr/lib/libzfs.so*

# Add to CGO flags if needed
export CGO_LDFLAGS="-L/usr/lib -lzfs -lnvpair"
```

### Runtime Issues

**Problem**: Permission denied errors

```
operation not permitted: insufficient privileges
```

**Solution**: Run with elevated privileges

```bash
# Use doas or sudo
doas go run main.go

# Or set capabilities (if supported)
setfacl -m u:username:rwx /dev/zfs
```

**Problem**: Memory leaks in CGO code

```
runtime: memory corruption detected
```

**Solution**: Audit C memory management

- Ensure all `C.CString()` calls have matching `C.free()`
- Close all ZFS handles with `zfs_close()`
- Free nvlists with `nvlist_free()`

## Contributing Guidelines

### Code Style

- Follow standard Go conventions
- Use `gofmt` (enforced by CI)
- Write comprehensive comments for exported functions
- Include context.Context in all public APIs

### Pull Request Process

1. Create feature branch from `main`
2. Implement feature with tests
3. Update documentation
4. Ensure CI passes
5. Request review from maintainers

### Commit Message Format

```
component: brief description

Longer description of the change, including:
- What was changed
- Why it was changed
- Any breaking changes
- References to issues

Fixes #123
```

Example:

```
zfs: add dataset rename functionality

Implements dataset renaming through new RenameDataset API.
Includes support for force rename flag and proper error
handling for invalid operations.

The change is backward compatible and includes comprehensive
tests and documentation updates.

Fixes #45
```
