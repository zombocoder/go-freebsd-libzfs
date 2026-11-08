# Examples

This directory contains example programs demonstrating the FreeBSD Go ZFS library.

## Prerequisites

- FreeBSD 14 or 15 with OpenZFS 2.2+
- Go 1.22+
- libzfs and libnvpair libraries
- Root privileges (for ZFS operations)

## Running Examples

Read-only examples (listing, properties, status) work as a regular user.
Examples that change pools or datasets (create, destroy, import, export,
scrub, snapshots, clones) require root:

```bash
go run ./examples/basic
doas go run ./examples/scrub start zroot
```

## Available Examples

### Read-only

| Example | What it shows | Usage |
|---|---|---|
| `basic` | Version detection, capabilities, pools, datasets, snapshots | `go run ./examples/basic` |
| `pool-properties` | All properties of a pool with their sources | `go run ./examples/pool-properties zroot` |
| `dataset-properties` | All properties of a dataset with their sources | `go run ./examples/dataset-properties zroot/home` |
| `list-snapshots` | Snapshots grouped by dataset | `go run ./examples/list-snapshots [zroot/home]` |
| `pool-status` | `zpool status`-style health report: status condition, scrub/resilver progress, vdev tree with READ/WRITE/CKSUM errors, spares, cache, data errors. Exits 1 if any pool is unhealthy, so it can be used for monitoring | `go run ./examples/pool-status [pool...]` |
| `error-handling` | Typed errors and error predicates | `go run ./examples/error-handling` |

### Pool management (root)

| Example | What it shows | Usage |
|---|---|---|
| `pool-import` | Discover pools available for import (`zpool import`) and import them, optionally from a directory, under a new name, with an alternate root, forced, or destroyed pools | `pool-import [-d dir]... [-D]` to list; `pool-import [-d dir]... [-D] [-f] [-R altroot] <pool> [newname]` to import |
| `scrub` | Start, pause, resume, stop and monitor scrubs | `scrub start [-w] <pool>`, `scrub pause <pool>`, `scrub stop <pool>`, `scrub status <pool>` |
| `pool-operations` | Status, export, import and destroy (asks for confirmation) | `pool-operations status\|export\|export-force\|import\|destroy <pool>` |

### Dataset management (root)

| Example | What it shows | Usage |
|---|---|---|
| `dataset-operations` | Create filesystems and volumes, destroy, show info | `dataset-operations create-fs zroot/test`, `create-vol zroot/vol1 1G`, `destroy[-recursive] <ds>`, `info <ds>` |
| `snapshot-operations` | Create (recursive), destroy, roll back and list snapshots | `snapshot-operations create zroot/test@backup`, `rollback zroot/test zroot/test@backup`, `list [ds]` |
| `clone-operations` | Create, promote, inspect and destroy clones. Works on the hard-coded `tank/test@snap1` / `tank/test-clone`; edit the names before running | `clone-operations` |

### Trying the pool examples safely

`pool-import` and `scrub` are easiest to try on a throwaway pool backed by files:

```bash
mkdir -p /var/tmp/demo
truncate -s 128m /var/tmp/demo/d0 /var/tmp/demo/d1
doas zpool create demo mirror /var/tmp/demo/d0 /var/tmp/demo/d1

doas go run ./examples/scrub start -w demo
go run ./examples/pool-status demo

doas zpool export demo
go run ./examples/pool-import -d /var/tmp/demo        # lists "demo"
doas go run ./examples/pool-import -d /var/tmp/demo demo

doas zpool destroy demo && rm -r /var/tmp/demo
```

## Building Examples

You can also build the examples as standalone binaries:

```bash
# Build all examples
for dir in examples/*/; do
  if [ -f "$dir/main.go" ]; then
    name=$(basename "$dir")
    go build -o "bin/$name" "$dir/main.go"
  fi
done

# Run built examples
sudo ./bin/basic
sudo ./bin/pool-properties zroot
sudo ./bin/dataset-properties zroot/home
```

## Expected Output

The examples will show:

- ZFS version and capability information
- Available pools with health/state
- Pool and dataset properties
- Snapshot information
- Error handling demonstrations

If no ZFS pools/datasets exist, the examples will still demonstrate the API functionality and error handling.

## Troubleshooting

### Permission Denied

```
Error: permission denied
```

**Solution:** Run with `doas` (or `sudo`) - operations that modify pools or datasets require root privileges.

### libzfs Not Found

```
Error: failed to initialize libzfs driver
```

**Solution:** libzfs is part of the FreeBSD base system; make sure the `zfs` kernel module is loaded (`kldload zfs`) and that you are running on FreeBSD 14 or newer.

### No Pools Found

```
Found 0 pools
```

This is expected if no ZFS pools are configured. The examples will still demonstrate API functionality.

### Compilation Errors

```
Error: undefined: driver.NewLibZFS
```

**Solution:** Ensure you're running on FreeBSD with proper build tags.
