# go-freebsd-libzfs

[![CI](https://github.com/zombocoder/go-freebsd-libzfs/actions/workflows/ci.yml/badge.svg)](https://github.com/zombocoder/go-freebsd-libzfs/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/zombocoder/go-freebsd-libzfs.svg)](https://pkg.go.dev/github.com/zombocoder/go-freebsd-libzfs)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-%3E%3D1.22-00ADD8.svg)](https://golang.org/)
[![FreeBSD](https://img.shields.io/badge/FreeBSD-14%20%7C%2015-red.svg)](https://www.freebsd.org/)

A FreeBSD-native Go library for managing ZFS pools and datasets through libzfs (via cgo).

## Overview

This library provides idiomatic Go bindings for FreeBSD's OpenZFS stack that call libzfs directly — **no `zfs`/`zpool` CLI calls**. It covers datasets, snapshots, clones, properties, pool import/export, pool health and status, and scrubs through a **type-safe, context-aware API**.

**Platform:** FreeBSD 14 and 15 (OpenZFS 2.2+)  
**Tested in CI on:** FreeBSD 14.5 and 15.1, against real ZFS pools in QEMU VMs  
**Go Version:** 1.22+

> The project is pre-1.0: the API may still change between minor versions.
> See the [changelog](CHANGELOG.md) and [releases](https://github.com/zombocoder/go-freebsd-libzfs/releases).

## Key Features

**Tested Against Real ZFS** - Integration tests compare results with `zfs`/`zpool` on throwaway pools  
**Complete Dataset Support** - Filesystems, volumes, snapshots, and bookmarks  
**Advanced Snapshots** - Create, list, rollback, destroy with hierarchical filtering  
**Clone Operations** - Create, promote, manage, and query clone relationships  
**Property Management** - Type-safe get/set operations with source tracking, including user properties  
**Pool Operations** - Discover, import, export, create, destroy  
**Pool Health** - `zpool status`-level detail: vdev tree, error counters, scrub/resilver progress, status codes  
**Scrub Control** - Start, pause, resume and stop scrubs  
**Feature Detection** - Runtime capability probing and version detection  
**Memory Safe** - Every C allocation freed, every libzfs handle closed  
**Type Safe** - Strongly typed APIs throughout with comprehensive error handling  
**No CLI Shelling** - Direct libzfs integration  
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

- FreeBSD 14 or 15 with OpenZFS 2.2+ (libzfs, libzutil and libnvpair are part of the base system)
- Go 1.22+
- OpenZFS headers in `/usr/src/sys/contrib/openzfs` for compilation (see below)

## Installation

```bash
go get github.com/zombocoder/go-freebsd-libzfs
```

The cgo code needs the OpenZFS headers from the FreeBSD source tree. If
`/usr/src/sys/contrib/openzfs/include/libzfs.h` is missing, extract them from
the `src.txz` of the release you run:

```bash
rel=$(freebsd-version -u | sed -E 's/-p[0-9]+$//')
fetch "https://download.freebsd.org/releases/$(uname -m)/${rel}/src.txz"
doas tar -xf src.txz -C / \
    --include './usr/src/sys/contrib/openzfs/include/*' \
    --include './usr/src/sys/contrib/openzfs/lib/libspl/include/*'
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

### Pool Health, Status and Scrub

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

    st, err := client.GetStatus(ctx, "zroot")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("%s: %s (%s)\n", st.Pool.Name, st.Pool.Health, st.Problem)
    if st.MsgID != "" {
        fmt.Printf("see https://openzfs.github.io/openzfs-docs/msg/%s\n", st.MsgID)
    }

    // Walk the vdev tree and report devices with errors
    var walk func(v zpool.VDevTree)
    walk = func(v zpool.VDevTree) {
        if v.Stats.ReadErrors+v.Stats.WriteErrors+v.Stats.ChecksumErrors > 0 {
            fmt.Printf("%s %s: read %d write %d cksum %d\n", v.Name, v.State,
                v.Stats.ReadErrors, v.Stats.WriteErrors, v.Stats.ChecksumErrors)
        }
        for _, c := range v.Children {
            walk(c)
        }
    }
    walk(st.Config)

    if st.Scan != nil {
        fmt.Printf("last %s: %s, %d errors\n", st.Scan.Function, st.Scan.State, st.Scan.Errors)
    }

    // Start a scrub (requires root); PauseScrub / StopScrub control it
    if err := client.StartScrub(ctx, "zroot"); err != nil {
        log.Fatal(err)
    }
}
```

### Discovering and Importing Pools

```go
// Like `zpool import -d /var/tmp/disks`
pools, err := client.Discover(ctx, zpool.DiscoverOptions{
    SearchPaths: []string{"/var/tmp/disks"},
})
if err != nil {
    log.Fatal(err)
}
for _, p := range pools {
    fmt.Printf("%s (%d): %s, %s, devices %v\n", p.Name, p.GUID, p.State, p.Health, p.DevicePaths)
}

// Like `zpool import -d /var/tmp/disks -R /mnt tank tank2`
err = client.Import(ctx, "tank", zpool.ImportOptions{
    SearchPaths: []string{"/var/tmp/disks"},
    AltRoot:     "/mnt",
    NewName:     "tank2",
})
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

#### User properties

User properties are string annotations whose names contain a colon. They can
be set with `SetProperty` or at creation (`Create*`, `CreateClone`), and read
back with `GetUserProperty`, `UserProperties` or `GetProperties`:

```go
_ = client.SetProperty(ctx, "tank/jobs/c1", "microvm-runner:agent", "agent-1")

value, ok, err := client.GetUserProperty(ctx, "tank/jobs/c1", "microvm-runner:agent")
// value == "agent-1", ok == true; ok is false if the property is not set.
// GetUserProperty also returns inherited values; for ownership checks use
// GetLocalUserProperty, which reports ok only for values set on the dataset itself.

all, err := client.UserProperties(ctx, "tank/jobs/c1") // map[name]{Value, Source}

// Filesystems and volumes below tank/jobs whose property is set locally to "agent-1"
owned, err := client.FindByUserProperty(ctx, "tank/jobs", "microvm-runner:agent", "agent-1")
```

`GetProperties` fails with `errors.ErrInvalidArgument` for a name that is
neither a native nor a user property.

## Examples

The [`examples/`](examples/) directory has 12 runnable programs, from read-only
listing to pool import and scrubbing, including:

- **`basic`** — version, capabilities, pools, datasets, snapshots
- **`pool-status`** — `zpool status`-style health report, exits 1 if a pool is unhealthy
- **`pool-import`** — list importable pools and import them
- **`scrub`** — start, pause, stop and monitor scrubs
- **`snapshot-operations`**, **`clone-operations`**, **`dataset-operations`** — dataset lifecycle

```bash
go run ./examples/pool-status
doas go run ./examples/scrub start -w zroot
```

See [examples/README.md](examples/README.md) for all of them and a recipe for
trying the pool examples on a throwaway file-backed pool.

## Testing

```bash
make test                     # unit and read-only integration tests, any user
doas make test-destructive    # also tests that create throwaway pools (root)
```

Integration tests compare the library with the `zfs`/`zpool` commands. Tests
that change pools create their own small file-backed pools under `/var/tmp`
and remove them afterwards; they only run as root with `ZFS_TEST_DESTRUCTIVE=1`.
CI runs the full suite on FreeBSD 14.5 and 15.1. See
[CONTRIBUTING.md](CONTRIBUTING.md#running-tests) for details.

## Status

### Implemented

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

- **Pool Listing** - List all pools with GUID, health and state
- **Pool Discovery** - Find pools available for import, including destroyed pools
- **Pool Import/Export** - Import (search paths, new name, alternate root, force, destroyed) and export
- **Pool Status** - Vdev tree with states and error counters, spares, cache, data errors, status code and message id
- **Scrub** - Start, pause, resume, stop; progress and results
- **Pool Destruction** - Destroy pools

#### **Vdev Management** (driver level, not yet exposed in the public `zpool` API)

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

#### **Quality**

- **Type Safety** - Strongly typed APIs for all ZFS operations
- **Memory Safety** - Proper CGO resource management and cleanup
- **Deduplication** - Intelligent duplicate removal in comprehensive iterations
- **Error Handling** - Comprehensive error propagation with ZFS errno details
- **Context Support** - All operations accept context.Context for cancellation

#### **Testing & Examples**

- **Integration tests** against real ZFS, compared with `zfs`/`zpool` output
- **Destructive tests** on throwaway file-backed pools: import/export, discovery, degraded and corrupted mirrors, scrubs
- **CI** on FreeBSD 14.5 and 15.1 (QEMU VMs)
- **12 example programs**

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
- Vdev management in the public `zpool` API
- ARC statistics
- Performance benchmarks and optimization
- IOCTL driver completion for CGO-free operation

## Design Principles

- **Context-aware:** All operations accept `context.Context` for cancellation
- **Strongly typed:** Properties and enums are type-safe
- **Zero external deps:** No shelling out to CLI tools
- **Memory safe:** Careful CGO memory management
- **FreeBSD native:** No cross-platform compromises

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for
the development setup, how tests work and what is in scope, and follow the
[Code of Conduct](CODE_OF_CONDUCT.md). Report security issues privately as
described in [SECURITY.md](SECURITY.md).

## License

MIT, see [LICENSE](LICENSE).
