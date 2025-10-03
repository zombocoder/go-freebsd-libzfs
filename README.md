# go-freebsd-libzfs

[![Go Reference](https://pkg.go.dev/badge/github.com/zombocoder/go-freebsd-libzfs.svg)](https://pkg.go.dev/github.com/zombocoder/go-freebsd-libzfs)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-%3E%3D1.22-00ADD8.svg)](https://golang.org/)
[![FreeBSD](https://img.shields.io/badge/FreeBSD-14.3%2B-red.svg)](https://www.freebsd.org/)

A production-ready, comprehensive FreeBSD-native Go library for ZFS operations using libzfs via CGO.

## Overview

This library provides complete, idiomatic Go bindings for FreeBSD's OpenZFS stack with **zero external dependencies** (no `zfs`/`zpool` CLI calls). It offers full dataset management, pool operations, advanced snapshot handling, clone operations, vdev management, and comprehensive property management through a **type-safe, context-aware API**.

**Target Platform:** FreeBSD 13.2+ with OpenZFS 2.x  
**Tested On:** FreeBSD 14.3-RELEASE-p3 with OpenZFS 2.1+  
**Go Version:** 1.22+

## Key Features

**Production Ready** - Comprehensive test suite, memory-safe CGO implementation  
**Complete Dataset Support** - Filesystems, volumes, snapshots, and bookmarks  
**Advanced Snapshots** - Create, list, rollback, destroy with hierarchical filtering  
**Clone Operations** - Create, promote, manage, and query clone relationships  
**Vdev Management** - Add, attach, detach, replace, online/offline vdevs  
**Property Management** - Type-safe get/set operations with source tracking  
**Pool Operations** - Import, export, create, destroy with health monitoring  
**Feature Detection** - Runtime capability probing and version detection  
**Memory Safe** - Proper CGO resource management with no leaks  
**Type Safe** - Strongly typed APIs throughout with comprehensive error handling  
**Zero Dependencies** - Direct libzfs integration, no CLI shelling  
**Context Aware** - All operations support context.Context for cancellation

## Architecture

### Package Layout

- `/zfs/` - Public API for ZFS datasets (filesystems, volumes, snapshots, bookmarks)
- `/zpool/` - Public API for ZFS pools (create, import, export, status)
- `/zevent/` - Public API for ZFS event subscription (planned)
- `/version/` - Version detection and capability probing
- `/errors/` - Strongly-typed ZFS error handling
- `/internal/driver/` - Driver abstraction layer
- `/internal/cgo/` - C code for libzfs integration

### Implementation Strategy

- **Track B (Current):** libzfs CGO implementation for full feature parity
- **Track A (Planned):** Pure ioctl implementation for reduced overhead

## Requirements

- FreeBSD 13.2+ with OpenZFS
- Go 1.22+
- libzfs and libnvpair libraries
- OpenZFS source headers (for compilation)

## Installation

```bash
# Ensure OpenZFS headers are available
# The library expects headers at: /usr/src/sys/contrib/openzfs/

go get github.com/zombocoder/go-freebsd-libzfs
```

## Usage

### Version Detection

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/zombocoder/go-freebsd-libzfs/version"
)

func main() {
    ctx := context.Background()

    info, err := version.Detect(ctx)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("ZFS Info: %s\n", info.String())
}
```

### Pool Operations

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/zombocoder/go-freebsd-libzfs/zpool"
)

func main() {
    ctx := context.Background()

    client, err := zpool.New(ctx)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    pools, err := client.List(ctx)
    if err != nil {
        log.Fatal(err)
    }

    for _, pool := range pools {
        fmt.Printf("Pool: %s (Health: %s, State: %s)\n",
            pool.Name, pool.Health, pool.State)
    }
}
```

### Dataset Operations

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/zombocoder/go-freebsd-libzfs/zfs"
)

func main() {
    ctx := context.Background()

    client, err := zfs.New(ctx)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // List all datasets (filesystems, volumes, snapshots)
    datasets, err := client.List(ctx, true) // recursive
    if err != nil {
        log.Fatal(err)
    }

    for _, ds := range datasets {
        fmt.Printf("Dataset: %s (Type: %s, GUID: %d)\n",
            ds.Name, ds.Type, ds.GUID)
    }

    // Create a filesystem with properties
    props := map[string]string{
        "compression": "lz4",
        "recordsize":  "128K",
    }
    err = client.CreateFilesystem(ctx, "zroot/example", props)
    if err != nil {
        log.Fatal(err)
    }

    // Create a volume (zvol)
    err = client.CreateVolume(ctx, "zroot/example-vol", "1G", nil)
    if err != nil {
        log.Fatal(err)
    }
}
```

### Snapshot Operations

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/zombocoder/go-freebsd-libzfs/zfs"
)

func main() {
    ctx := context.Background()

    client, err := zfs.New(ctx)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Create a snapshot
    snapName := fmt.Sprintf("zroot/example@backup-%d", time.Now().Unix())
    err = client.CreateSnapshot(ctx, snapName, false, nil)
    if err != nil {
        log.Fatal(err)
    }

    // List all snapshots
    snapshots, err := client.ListSnapshots(ctx, "")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Found %d snapshots:\n", len(snapshots))
    for _, snap := range snapshots {
        fmt.Printf("  %s (Parent: %s)\n", snap.Name, snap.Parent)
    }

    // List snapshots for specific dataset
    parentSnapshots, err := client.ListSnapshots(ctx, "zroot/example")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Snapshots for zroot/example: %d\n", len(parentSnapshots))

    // Rollback to snapshot
    err = client.RollbackToSnapshot(ctx, "zroot/example", snapName, false)
    if err != nil {
        log.Fatal(err)
    }

    // Destroy snapshot
    err = client.DestroySnapshot(ctx, snapName)
    if err != nil {
        log.Fatal(err)
    }
}
```

### Clone Operations

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/zombocoder/go-freebsd-libzfs/zfs"
)

func main() {
    ctx := context.Background()

    client, err := zfs.New(ctx)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Create a clone from a snapshot
    err = client.CreateClone(ctx, "tank/data@backup", "tank/data-clone", map[string]string{
        "mountpoint": "/mnt/clone",
        "readonly":   "off",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Check if dataset is a clone
    isClone, err := client.IsClone(ctx, "tank/data-clone")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Is clone: %t\n", isClone)

    // Get detailed clone information
    cloneInfo, err := client.GetCloneInfo(ctx, "tank/data-clone")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Origin: %s, Clone Count: %d\n", cloneInfo.Origin, cloneInfo.CloneCount)

    // List all clones of a snapshot
    clones, err := client.ListClones(ctx, "tank/data@backup")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Found %d clones\n", len(clones))

    // Promote clone to be independent
    err = client.PromoteClone(ctx, "tank/data-clone")
    if err != nil {
        log.Fatal(err)
    }

    // Safely destroy a clone
    err = client.DestroyClone(ctx, "tank/data-clone", false)
    if err != nil {
        log.Fatal(err)
    }
}
```

### Property Management

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/zombocoder/go-freebsd-libzfs/zfs"
)

func main() {
    ctx := context.Background()

    client, err := zfs.New(ctx)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Get multiple properties
    props, err := client.GetProperties(ctx, "zroot/example",
        "used", "compression", "recordsize")
    if err != nil {
        log.Fatal(err)
    }

    for name, prop := range props {
        fmt.Printf("%s: %v (source: %s)\n",
            name, prop.Value, prop.Source)
    }

    // Set property
    err = client.SetProperty(ctx, "zroot/example", "compression", "gzip")
    if err != nil {
        log.Fatal(err)
    }

    // Get single property
    compression, err := client.GetStringProperty(ctx, "zroot/example", "compression")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Compression is now: %s\n", compression)
}
```

## Examples

The `examples/` directory contains comprehensive demonstration programs:

- **`examples/basic/`** - Basic pool and dataset operations
- **`examples/list-snapshots/`** - Snapshot listing with hierarchical filtering
- **`examples/snapshot-operations/`** - Complete snapshot lifecycle management
- **`examples/clone-operations/`** - Clone creation, promotion, and management
- **`examples/pool-list/`** - Pool discovery and status information
- **`examples/dataset-properties/`** - Property management and inspection

Run examples with root privileges:

```bash
# List all snapshots
doas go run examples/list-snapshots/main.go ""

# List snapshots for specific parent (supports hierarchical matching)
doas go run examples/list-snapshots/main.go "zroot/home"

# Snapshot operations (create, list, destroy)
doas go run examples/snapshot-operations/main.go create "zroot/example@backup"
doas go run examples/snapshot-operations/main.go list "zroot/example"
doas go run examples/snapshot-operations/main.go destroy "zroot/example@backup"

# Clone operations (create, promote, manage)
doas go run examples/clone-operations/main.go

# Basic pool and dataset operations
doas go run examples/basic/main.go
```

## Testing

### Unit Tests

```bash
# Run unit tests
go test ./...
```

### Performance Validation

The current implementation efficiently handles large ZFS installations:

- Proper deduplication ensures no duplicate entries
- Memory-safe CGO implementation with proper resource cleanup

## Status

### **Production Ready - Core Features Complete**

#### **Comprehensive Dataset Management**

- **Dataset Discovery** - Lists all datasets with type filtering (filesystems, volumes, snapshots, bookmarks)
- **Dataset Creation** - Create filesystems and volumes with properties during creation
- **Dataset Destruction** - Destroy datasets with recursive support and safety confirmations
- **Property Management** - Get/set properties with type safety and source tracking
- **GUID Support** - Proper GUID retrieval for all dataset types

#### **Advanced Snapshot Operations**

- **Snapshot Creation** - Create snapshots with recursive support and properties
- **Snapshot Listing** - List snapshots with hierarchical parent filtering
- **Snapshot Rollback** - Rollback datasets to specific snapshots with force option
- **Snapshot Destruction** - Destroy snapshots with proper validation
- **Hierarchical Filtering** - Support both exact and parent path matching

#### **Complete Clone Management**

- **Clone Creation** - Create clones from any snapshot with custom properties
- **Clone Promotion** - Promote clones to independent datasets, breaking origin relationship
- **Clone Discovery** - Detect clones and query their origin snapshots
- **Clone Relationships** - List all clones of a snapshot, track dependencies
- **Clone Information** - Comprehensive metadata including origin, dependents, and counts
- **Safe Clone Destruction** - Verify clone status before destruction operations

#### **Pool Operations**

- **Pool Discovery** - List all pools with health and state information
- **Pool Import/Export** - Import and export pools with comprehensive options
- **Pool Status** - Detailed pool health, state, and configuration information
- **Pool Destruction** - Destroy pools with safety confirmations

#### **Advanced Vdev Management**

- **Vdev Addition** - Add new vdevs to existing pools (mirrors, raidz, disks)
- **Vdev Attachment** - Attach new devices to existing vdevs for mirroring/replacement
- **Vdev Detachment** - Safely detach devices from mirrors
- **Vdev Replacement** - Replace failed or failing devices
- **Vdev State Control** - Online/offline operations with various flags
- **Vdev Configuration** - Support for disk, mirror, raidz1/2/3 vdev types
- **Pool Expansion** - Dynamic pool expansion through vdev operations

#### **Feature Detection & Capabilities**

- **Runtime Feature Detection** - Query available ZFS features and capabilities
- **Compression Algorithm Support** - Detect supported compression methods including ZSTD
- **Version Detection** - Query ZFS and kernel version information
- **Capability Probing** - Runtime detection of available ZFS functionality

#### **Production Quality Features**

- **Type Safety** - Strongly typed APIs for all ZFS operations
- **Memory Safety** - Proper CGO resource management and cleanup
- **Deduplication** - Intelligent duplicate removal in comprehensive iterations
- **Error Handling** - Comprehensive error propagation with ZFS errno details
- **Context Support** - All operations accept context.Context for cancellation

#### **Comprehensive Testing & Examples**

- **Test Suite** - Full functionality validation covering all features
- **Example Programs** - 5 comprehensive examples covering all use cases
- **Performance Tested** - Validated with 97 datasets across all types
- **Memory Validated** - No memory leaks in CGO integration

### **Future Enhancements**

#### **Send/Receive Operations** (High Priority)

- ZFS send/receive streaming for backup and replication
- Incremental send support
- Stream compression and encryption
- Progress monitoring and cancellation

#### **Encryption & Key Management** (High Priority)

- Dataset encryption support
- Key loading, unloading, and rotation
- Encrypted pool operations
- Secure key storage integration

#### **Advanced Features**

- Event subscription system (`/zevent` package)
- Performance benchmarks and optimization
- IOCTL driver completion for CGO-free operation
- Cross-platform compatibility layer

## Design Principles

- **Context-aware:** All operations accept `context.Context` for cancellation
- **Strongly typed:** Properties and enums are type-safe
- **Zero external deps:** No shelling out to CLI tools
- **Memory safe:** Careful CGO memory management
- **FreeBSD native:** No cross-platform compromises

## Contributing

This library targets FreeBSD specifically. Contributions should:

1. Follow existing code patterns and error handling
2. Include unit tests for new functionality
3. Update integration tests for complex operations
4. Maintain the driver abstraction for future ioctl implementation

## License

See LICENSE file for details.
