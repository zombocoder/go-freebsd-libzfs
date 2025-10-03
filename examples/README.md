# Examples

This directory contains example programs demonstrating the FreeBSD Go ZFS library.

## Prerequisites

- FreeBSD 14.3+ with OpenZFS
- Go 1.22+
- libzfs and libnvpair libraries
- Root privileges (for ZFS operations)

## Running Examples

All examples require root privileges to access ZFS:

```bash
sudo go run examples/basic/main.go
```

## Available Examples

### 1. Basic (`examples/basic/`)

Comprehensive test of all major library features:

- Version detection and capability probing
- Pool listing and properties
- Dataset listing and properties
- Snapshot listing

```bash
sudo go run examples/basic/main.go
```

### 2. Pool Properties (`examples/pool-properties/`)

Detailed pool property inspection:

```bash
sudo go run examples/pool-properties/main.go <pool-name>
```

Example:

```bash
sudo go run examples/pool-properties/main.go zroot
```

### 3. Dataset Properties (`examples/dataset-properties/`)

Detailed dataset property inspection:

```bash
sudo go run examples/dataset-properties/main.go <dataset-name>
```

Example:

```bash
sudo go run examples/dataset-properties/main.go zroot/home
```

### 4. List Snapshots (`examples/list-snapshots/`)

Comprehensive snapshot listing with grouping:

```bash
# List all snapshots
sudo go run examples/list-snapshots/main.go

# List snapshots for specific parent dataset
sudo go run examples/list-snapshots/main.go zroot/home
```

### 5. Error Handling (`examples/error-handling/`)

Demonstrates proper error handling patterns:

```bash
sudo go run examples/error-handling/main.go
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

**Solution:** Run with `sudo` - ZFS operations require root privileges.

### libzfs Not Found

```
Error: failed to initialize libzfs driver
```

**Solution:** Ensure libzfs is installed: `pkg install openzfs`

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
