# Contributing to go-freebsd-libzfs

Thanks for your interest in improving the library! Bug reports, fixes, new
operations, tests, examples and documentation are all welcome.

By participating you agree to follow the [Code of Conduct](CODE_OF_CONDUCT.md).
Security problems should be reported privately, see [SECURITY.md](SECURITY.md).

## Table of Contents

- [Getting Started](#getting-started)
- [Running Tests](#running-tests)
- [Writing Tests](#writing-tests)
- [Continuous Integration](#continuous-integration)
- [Code Style](#code-style)
- [Submitting Changes](#submitting-changes)
- [Architecture Overview](#architecture-overview)
- [Where Help Is Wanted](#where-help-is-wanted)

## Getting Started

### Prerequisites

- FreeBSD 14 or 15 (OpenZFS 2.2+), amd64 or arm64
- Go 1.22+ (`pkg install go`)
- The OpenZFS headers in `/usr/src/sys/contrib/openzfs` (see below)
- Root (or `doas`) for tests and examples that create or change pools

A FreeBSD virtual machine works fine; tests that modify the system only use
throwaway pools backed by files.

### OpenZFS headers

The cgo code compiles against the libzfs headers from the FreeBSD source tree.
If `/usr/src/sys/contrib/openzfs/include/libzfs.h` does not exist, either
install the full source tree, or extract just the headers from the `src.txz`
of the release you are running:

```sh
rel=$(freebsd-version -u | sed -E 's/-p[0-9]+$//')
fetch "https://download.freebsd.org/releases/$(uname -m)/${rel}/src.txz"
doas tar -xf src.txz -C / \
    --include './usr/src/sys/contrib/openzfs/include/*' \
    --include './usr/src/sys/contrib/openzfs/lib/libspl/include/*'
```

`doas sh .github/scripts/freebsd-ci-prepare.sh` does the same (and installs Go);
it is what CI uses.

### Setup

```sh
git clone https://github.com/<your-username>/go-freebsd-libzfs.git
cd go-freebsd-libzfs
make deps     # checks Go, ZFS module, libzfs and headers
make build
make test
```

## Running Tests

There are two levels of tests:

| Command | Runs as | What it covers |
|---|---|---|
| `make test` | any user | Unit tests and read-only integration tests. The latter compare library results with the `zfs`/`zpool` commands on the first imported pool and skip when there is none. |
| `doas make test-destructive` | root | Everything above plus tests that create, export, import, scrub, damage and destroy pools. Each such test creates its own pool named `gozfs_*` from files under `/var/tmp` and removes it afterwards. |

`test-destructive` sets `ZFS_TEST_DESTRUCTIVE=1` and runs packages one at a
time (`go test -p 1`) because tests observe global pool state. Running a
subset works the usual way:

```sh
doas env ZFS_TEST_DESTRUCTIVE=1 go test -count=1 -run Scrub ./zpool
```

Destructive tests only touch pools they create, but they do run as root and
temporarily change `vfs.zfs.scan_suspend_progress`. Prefer a VM or a test
machine over your workstation.

## Writing Tests

- **Test first.** Write a test that fails for the right reason, then the fix.
- **Compare with the CLI.** Integration tests check the library against
  `zfs`/`zpool` output where possible (e.g. GUIDs, property sources, listings),
  so the expected values come from ZFS itself rather than from the code under
  test.
- **Use the helpers** in `zpool/zpool_import_test.go` for anything that changes
  pools: `requireDestructive`, `newTestPool`, `createTestPool` (single disk or
  mirror) and `run`. Register cleanup with `t.Cleanup` so pools are destroyed
  even when a test fails.
- **Never** modify existing pools, and keep test pools small (128 MB files).
- Read-only tests use `firstPool`, which ignores temporary `gozfs_*` pools.

## Continuous Integration

[`.github/workflows/ci.yml`](.github/workflows/ci.yml) runs on every push to
`main` and every pull request:

- **Lint** (Linux): `gofmt` and `go mod tidy -diff`.
- **FreeBSD 14.5 and 15.1** (QEMU VMs via
  [vmactions/freebsd-vm](https://github.com/vmactions/freebsd-vm)): installs Go
  and the OpenZFS headers, creates a small file-backed pool `citank`
  ([`freebsd-ci-pool.sh`](.github/scripts/freebsd-ci-pool.sh)), then runs
  `go vet`, builds all packages and examples, and runs the full test suite as
  root with `ZFS_TEST_DESTRUCTIVE=1`.

## Code Style

### Go

- Standard Go conventions; `gofmt` is enforced by CI.
- Doc comments on all exported identifiers.
- `context.Context` as the first parameter of every public operation.
- Wrap errors with context (`fmt.Errorf("...: %w", err)`) and include the libzfs
  errno and description when an operation fails in libzfs.

```go
// Good: clear, documented, context-aware
func (c *Client) CreateDataset(ctx context.Context, name string, dsType DatasetType) error {
	if c.d == nil {
		return fmt.Errorf("client is closed")
	}
	return c.d.CreateDataset(ctx, name, dsType, nil)
}
```

### cgo

- **Memory**: every `C.CString`/`C.calloc` has a matching `C.free`; strings
  returned by libzfs that the caller owns (e.g. `zpool_vdev_name`) are freed.
- **Handles**: every `zpool_open`/`zfs_open` has a matching close, usually via
  `defer`.
- **Errors**: propagate `libzfs_errno` and `libzfs_error_description`.
- **Null checks**: check C pointers before use.
- **Locking**: driver methods hold `d.mu` for the whole libzfs interaction; a
  libzfs handle is not safe for concurrent use.
- Small C helpers live in `internal/cgo/zfs_c.go` and are declared `extern` in
  the driver file that uses them.

## Submitting Changes

1. Create a branch from `main`.
2. Make the change with tests; update doc comments, README and examples when
   the public API changes, and add an entry to [CHANGELOG.md](CHANGELOG.md).
3. Run `make check` (and `doas make test-destructive` if you touched code that
   changes pools) on FreeBSD.
4. Open a pull request; the template has a short checklist. CI must be green.

Commit messages: a short `component: summary` line (e.g.
`zpool: add trim support`), a blank line, then what changed and why.

### Scope

- The library is FreeBSD-specific by design; cross-platform layers are out of
  scope.
- It calls libzfs directly. Shelling out to `zfs`/`zpool` is only acceptable in
  tests.
- Avoid breaking public API changes. When one is unavoidable, call it out in the
  pull request and in the changelog.

## Architecture Overview

```
zfs/            Public dataset API (filesystems, volumes, snapshots, clones)
zpool/          Public pool API (list, status, import/export, scrub, ...)
version/        Version detection and capability probing
errors/         Structured error types
internal/
  driver/       Driver interface and implementations
    libzfs_driver.go   cgo implementation (current)
    libzfs_status.go   pool status and scrub
    ioctl_driver.go    pure-ioctl implementation (stubs, planned)
  cgo/          C helper functions around libzfs
examples/       Example programs
.github/        CI workflow, CI scripts, issue and PR templates
```

### Adding an operation

1. **C helper** (`internal/cgo/zfs_c.go`), only if the libzfs call needs
   structs, macros or callbacks that are awkward from Go.
2. **Driver interface** (`internal/driver/driver.go`): method and data types.
3. **libzfs implementation** (`internal/driver/libzfs_*.go`), plus a stub in
   `ioctl_driver.go` returning "not implemented".
4. **Public API** (`zfs/` or `zpool/`): documented wrapper with idiomatic types.
5. **Tests** and, if useful, an **example**.

See [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) for more detail.

## Where Help Is Wanted

- **Send/receive** streams for backup and replication
- **Encryption**: key loading/unloading and encrypted datasets
- **Events**: a `zevent` package for subscribing to ZFS events
- **Vdev management** in the public `zpool` API (the driver already has add,
  attach, detach, replace, online/offline)
- **ARC statistics** (`kstat.zfs.misc.arcstats.*`)
- **ioctl driver**: an implementation without cgo

Questions and design discussions are welcome in GitHub issues.
