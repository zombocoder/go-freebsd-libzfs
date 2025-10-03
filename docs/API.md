# API Reference

This document provides comprehensive API documentation for the go-freebsd-libzfs library.

## Table of Contents

- [Client Initialization](#client-initialization)
- [Pool Operations](#pool-operations)
- [Dataset Operations](#dataset-operations)
- [Snapshot Operations](#snapshot-operations)
- [Clone Operations](#clone-operations)
- [Property Management](#property-management)
- [Version and Capabilities](#version-and-capabilities)
- [Error Handling](#error-handling)
- [Data Structures](#data-structures)

## Client Initialization

### zfs.New(ctx context.Context) (*Client, error)

Creates a new ZFS client for dataset operations.

```go
client, err := zfs.New(ctx)
if err != nil {
    return fmt.Errorf("failed to create ZFS client: %w", err)
}
defer client.Close()
```

**Parameters:**
- `ctx`: Context for cancellation and timeouts

**Returns:**
- `*Client`: ZFS client instance
- `error`: Error if initialization fails

### zpool.New(ctx context.Context) (*Client, error)

Creates a new ZPool client for pool operations.

```go
client, err := zpool.New(ctx)
if err != nil {
    return fmt.Errorf("failed to create ZPool client: %w", err)
}
defer client.Close()
```

## Pool Operations

### client.List(ctx context.Context) ([]Pool, error)

Lists all ZFS pools with their status information.

```go
pools, err := client.List(ctx)
if err != nil {
    return fmt.Errorf("failed to list pools: %w", err)
}

for _, pool := range pools {
    fmt.Printf("Pool: %s (Health: %s, State: %s)\n", 
        pool.Name, pool.Health, pool.State)
}
```

**Returns:**
- `[]Pool`: Slice of Pool structures
- `error`: Error if operation fails

### client.Import(ctx context.Context, name string) error

Imports a ZFS pool that was previously exported.

```go
err := client.Import(ctx, "tank")
if err != nil {
    return fmt.Errorf("failed to import pool: %w", err)
}
```

### client.Export(ctx context.Context, name string, force bool) error

Exports a ZFS pool, making it unavailable until imported again.

```go
err := client.Export(ctx, "tank", false)
if err != nil {
    return fmt.Errorf("failed to export pool: %w", err)
}
```

**Parameters:**
- `name`: Name of the pool to export
- `force`: Force export even if datasets are in use

## Dataset Operations

### client.List(ctx context.Context, recursive bool) ([]Dataset, error)

Lists all ZFS datasets (filesystems, volumes, snapshots).

```go
datasets, err := client.List(ctx, true) // recursive
if err != nil {
    return fmt.Errorf("failed to list datasets: %w", err)
}

for _, ds := range datasets {
    fmt.Printf("Dataset: %s (Type: %s, GUID: %d)\n", 
        ds.Name, ds.Type, ds.GUID)
}
```

**Parameters:**
- `recursive`: Include child datasets

### client.CreateFilesystem(ctx context.Context, name string, props map[string]string) error

Creates a new ZFS filesystem with optional properties.

```go
props := map[string]string{
    "compression": "lz4",
    "recordsize":  "128K",
    "mountpoint":  "/mnt/data",
}
err := client.CreateFilesystem(ctx, "tank/data", props)
```

### client.CreateVolume(ctx context.Context, name, size string, props map[string]string) error

Creates a new ZFS volume (zvol) with specified size.

```go
err := client.CreateVolume(ctx, "tank/vm-disk", "10G", map[string]string{
    "volblocksize": "16K",
})
```

**Parameters:**
- `name`: Name of the volume
- `size`: Volume size (e.g., "10G", "1T")
- `props`: Optional properties

### client.Destroy(ctx context.Context, name string, recursive bool) error

Destroys a ZFS dataset.

```go
err := client.Destroy(ctx, "tank/old-data", false)
if err != nil {
    return fmt.Errorf("failed to destroy dataset: %w", err)
}
```

**Parameters:**
- `recursive`: Destroy child datasets as well

## Snapshot Operations

### client.CreateSnapshot(ctx context.Context, name string, recursive bool, props map[string]string) error

Creates a ZFS snapshot.

```go
snapName := "tank/data@backup-" + time.Now().Format("20060102-150405")
err := client.CreateSnapshot(ctx, snapName, false, nil)
```

**Parameters:**
- `name`: Snapshot name in format `dataset@snapshot`
- `recursive`: Create snapshots of child datasets
- `props`: Optional snapshot properties

### client.ListSnapshots(ctx context.Context, parent string) ([]Snapshot, error)

Lists snapshots, optionally filtered by parent dataset.

```go
// List all snapshots
allSnapshots, err := client.ListSnapshots(ctx, "")

// List snapshots for specific dataset
dataSnapshots, err := client.ListSnapshots(ctx, "tank/data")
```

**Parameters:**
- `parent`: Parent dataset name, or empty string for all snapshots

### client.RollbackToSnapshot(ctx context.Context, dataset, snapshot string, force bool) error

Rolls back a dataset to a specific snapshot.

```go
err := client.RollbackToSnapshot(ctx, "tank/data", "tank/data@backup", false)
```

**Parameters:**
- `dataset`: Target dataset name
- `snapshot`: Snapshot to rollback to
- `force`: Force rollback, destroying newer snapshots

### client.DestroySnapshot(ctx context.Context, name string) error

Destroys a ZFS snapshot.

```go
err := client.DestroySnapshot(ctx, "tank/data@old-backup")
```

## Clone Operations

### client.CreateClone(ctx context.Context, snapshot, clone string, props map[string]string) error

Creates a clone from a snapshot.

```go
err := client.CreateClone(ctx, "tank/data@backup", "tank/data-clone", map[string]string{
    "mountpoint": "/mnt/clone",
    "readonly":   "off",
})
```

**Parameters:**
- `snapshot`: Source snapshot name
- `clone`: New clone name
- `props`: Optional clone properties

### client.PromoteClone(ctx context.Context, clone string) error

Promotes a clone to be independent of its origin snapshot.

```go
err := client.PromoteClone(ctx, "tank/data-clone")
```

### client.IsClone(ctx context.Context, dataset string) (bool, error)

Checks if a dataset is a clone.

```go
isClone, err := client.IsClone(ctx, "tank/data-clone")
if err != nil {
    return fmt.Errorf("failed to check clone status: %w", err)
}
```

### client.GetCloneInfo(ctx context.Context, dataset string) (*CloneInfo, error)

Gets comprehensive information about a clone.

```go
info, err := client.GetCloneInfo(ctx, "tank/data-clone")
if err != nil {
    return fmt.Errorf("failed to get clone info: %w", err)
}

fmt.Printf("Origin: %s, Clone Count: %d\n", info.Origin, info.CloneCount)
```

### client.ListClones(ctx context.Context, snapshot string) ([]string, error)

Lists all clones of a specific snapshot.

```go
clones, err := client.ListClones(ctx, "tank/data@backup")
if err != nil {
    return fmt.Errorf("failed to list clones: %w", err)
}
```

### client.DestroyClone(ctx context.Context, clone string, force bool) error

Safely destroys a clone after verifying it's actually a clone.

```go
err := client.DestroyClone(ctx, "tank/data-clone", false)
```

## Property Management

### client.GetProperty(ctx context.Context, dataset, property string) (*Property, error)

Gets a single property value with source information.

```go
prop, err := client.GetProperty(ctx, "tank/data", "compression")
if err != nil {
    return fmt.Errorf("failed to get property: %w", err)
}

fmt.Printf("Compression: %v (source: %s)\n", prop.Value, prop.Source)
```

### client.GetProperties(ctx context.Context, dataset string, properties ...string) (map[string]*Property, error)

Gets multiple properties at once.

```go
props, err := client.GetProperties(ctx, "tank/data", 
    "used", "compression", "recordsize")
if err != nil {
    return fmt.Errorf("failed to get properties: %w", err)
}

for name, prop := range props {
    fmt.Printf("%s: %v (source: %s)\n", name, prop.Value, prop.Source)
}
```

### client.GetStringProperty(ctx context.Context, dataset, property string) (string, error)

Gets a property value as a string (convenience method).

```go
compression, err := client.GetStringProperty(ctx, "tank/data", "compression")
if err != nil {
    return fmt.Errorf("failed to get compression: %w", err)
}
```

### client.SetProperty(ctx context.Context, dataset, property, value string) error

Sets a property value.

```go
err := client.SetProperty(ctx, "tank/data", "compression", "gzip")
if err != nil {
    return fmt.Errorf("failed to set compression: %w", err)
}
```

## Version and Capabilities

### version.Detect(ctx context.Context) (*ZFSInfo, error)

Detects ZFS version and capability information.

```go
import "github.com/zombocoder/go-freebsd-libzfs/version"

info, err := version.Detect(ctx)
if err != nil {
    return fmt.Errorf("failed to detect ZFS version: %w", err)
}

fmt.Printf("ZFS Version: %s\n", info.String())
fmt.Printf("Kernel Version: %s\n", info.KernelVersion)
fmt.Printf("Supported Features: %v\n", info.Features)
```

### client.RuntimeInfo(ctx context.Context) (string, string, string, error)

Gets runtime information about the ZFS implementation.

```go
impl, zfsVer, kernel, err := client.RuntimeInfo(ctx)
if err != nil {
    return fmt.Errorf("failed to get runtime info: %w", err)
}

fmt.Printf("Implementation: %s, ZFS: %s, Kernel: %s\n", impl, zfsVer, kernel)
```

## Error Handling

The library provides structured error handling with ZFS-specific error types:

### ZFS Error Types

```go
import "github.com/zombocoder/go-freebsd-libzfs/errors"

// Check for specific ZFS errors
if zfsErr, ok := err.(*errors.ZFSError); ok {
    fmt.Printf("ZFS Error: %s (errno: %d)\n", zfsErr.Message, zfsErr.Errno)
}
```

### Common Error Patterns

```go
// Check if dataset exists
_, err := client.GetProperty(ctx, "tank/nonexistent", "used")
if err != nil {
    if errors.IsDatasetNotFound(err) {
        fmt.Println("Dataset does not exist")
    } else {
        return fmt.Errorf("unexpected error: %w", err)
    }
}
```

## Data Structures

### Dataset

```go
type Dataset struct {
    Name string      // Dataset name
    Type string      // Dataset type: filesystem, volume, snapshot, bookmark
    GUID uint64      // Unique identifier
}
```

### Pool

```go
type Pool struct {
    Name   string    // Pool name
    Health string    // Pool health status
    State  string    // Pool state
    GUID   uint64    // Pool GUID
}
```

### Snapshot

```go
type Snapshot struct {
    Dataset          // Embedded dataset fields
    Parent string    // Parent dataset name
}
```

### Clone

```go
type Clone struct {
    Dataset            // Embedded dataset fields
    Origin      string // Origin snapshot
    IsClone     bool   // Whether this is actually a clone
    CloneCount  int    // Number of clones from this dataset
    Dependents  []string // List of dependent clones
}
```

### CloneInfo

```go
type CloneInfo struct {
    Name         string   // Dataset name
    Type         string   // Dataset type
    GUID         uint64   // Dataset GUID
    Origin       string   // Origin snapshot (if clone)
    IsClone      bool     // Whether this is a clone
    CloneCount   int      // Number of clones
    Dependents   []string // Dependent datasets
}
```

### Property

```go
type Property struct {
    Value  interface{} // Property value (string, uint64, bool)
    Source string      // Property source (local, default, inherited)
}
```

### ZFSInfo

```go
type ZFSInfo struct {
    Version       string            // ZFS version
    KernelVersion string            // Kernel version
    Features      map[string]bool   // Supported features
}
```

## Context Usage

All operations accept `context.Context` for cancellation and timeouts:

```go
// With timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

datasets, err := client.List(ctx, true)

// With cancellation
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Cancel operation from another goroutine
go func() {
    time.Sleep(5 * time.Second)
    cancel()
}()

err := client.CreateSnapshot(ctx, "tank/data@backup", false, nil)
```

## Memory Management

The library handles CGO memory management automatically:

- All C strings are properly freed
- ZFS handles are closed after use
- nvlist structures are properly deallocated
- No manual memory management required

## Thread Safety

The library is thread-safe for concurrent operations:

```go
// Safe to use from multiple goroutines
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(i int) {
        defer wg.Done()
        
        snapName := fmt.Sprintf("tank/data@backup-%d", i)
        err := client.CreateSnapshot(ctx, snapName, false, nil)
        if err != nil {
            log.Printf("Failed to create snapshot %s: %v", snapName, err)
        }
    }(i)
}
wg.Wait()
```

## Best Practices

### Resource Cleanup

Always close clients when done:

```go
client, err := zfs.New(ctx)
if err != nil {
    return err
}
defer client.Close() // Important: always close
```

### Error Handling

Use structured error handling:

```go
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

### Property Validation

Validate properties before setting:

```go
validCompressions := []string{"off", "lz4", "gzip", "zstd"}
if !contains(validCompressions, compression) {
    return fmt.Errorf("invalid compression: %s", compression)
}
```

### Snapshot Naming

Use consistent snapshot naming conventions:

```go
snapName := fmt.Sprintf("%s@%s-%s", 
    dataset, 
    purpose, 
    time.Now().Format("20060102-150405"))
```