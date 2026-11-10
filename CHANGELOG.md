# Changelog

All notable changes to this project are documented here. The format is based
on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project
follows [Semantic Versioning](https://semver.org/). Before 1.0.0, minor
versions may contain breaking API changes; they are listed under "Changed".

## [Unreleased]

### Added

- ZFS user properties (names containing a colon, e.g. `microvm-runner:agent`):
  `zfs.Client.GetUserProperty`, `UserProperties` (value and source:
  local, inherited or received), `GetLocalUserProperty` (only values set on
  the dataset itself) and `FindByUserProperty` (filesystems and volumes below
  a parent whose user property is set locally to a value; one walk of the
  subtree, tolerates concurrent destroys, `ErrDatasetNotFound` if the parent
  is missing). `GetUserProperty` also returns inherited values: use the
  local variants for ownership checks.
- `GetProperties` returns user properties when asked for them by name; an
  unset one is left out of the result.
- `GetProperties` now reads any native ZFS property (previously only nine
  were known), e.g. `atime`, `refreservation`, `volsize`. Values up to 8192
  bytes are no longer truncated at 1024.
- User property names are validated like `zfs_prop_user()` (a colon, only
  lowercase letters, digits and `-_.:`, shorter than 256 bytes; values
  shorter than 8192 bytes without NUL bytes) in `SetProperty`, `Create*`,
  `CreateSnapshot` and `CreateClone`, failing with
  `errors.ErrInvalidArgument` or `errors.ErrNameTooLong`.
- Tests for setting user properties with `SetProperty` and for `CreateClone`
  with user properties and `refreservation`.

### Fixed

- Spurious `signal received` (`EZFS_INTR`) failures: libzfs does not retry an
  ioctl interrupted by a signal, and the Go runtime signals threads at any
  time (preemption, GC). Every driver operation now blocks asynchronous
  signals on its thread while it is inside libzfs, and read-only listings
  (pools, datasets, user-property walks) retry up to three times if libzfs
  still reports an interruption.
- Handle leak in dataset and pool listing: libzfs passes each handle to the
  iteration callback, which must close it; `List`, `ListInPool`,
  `ListSnapshots`, `zpool.Client.List` and friends never did, leaking one
  libzfs handle per dataset/pool per call.
- A failed pool or dataset iteration reports libzfs's error only if libzfs
  raised a new one during the iteration; otherwise it says the iteration was
  aborted, instead of repeating an old error.

### Changed

- Behaviour change: `GetProperties` with a name that is neither a native ZFS
  property nor a user property now fails with `errors.ErrInvalidArgument`;
  it used to silently omit the name. This includes `@` properties such as
  `userused@<user>` and `written@<snap>`, which are not supported. Valid native properties that do not apply
  to the dataset (e.g. `volsize` on a filesystem) are still omitted.
- `internal/driver.Driver` gained `GetUserProperties` and
  `ListDescendantUserProps` (the ioctl driver
  returns "not implemented", like its other methods).

## [0.2.0] - 2026-11-08

### Added

- `zpool.Client.Import` imports exported or destroyed pools (previously a stub),
  mirroring `zpool import`: search directories (`ImportOptions.SearchPaths`,
  like `-d`), new name, alternate root, force, destroyed pools. Datasets are
  mounted after import.
- `zpool.Client.Discover` lists pools available for import with name, GUID,
  state, health and device paths (previously always returned an empty list).
- `zpool.Client.GetStatus` returns the full `zpool status` picture: vdev tree
  with names, states and READ/WRITE/CKSUM counters, spares, cache devices,
  persistent data error count, scrub/resilver progress and the status
  condition with its `ZFS-8000-*` message id (`Status.Problem`, `Status.MsgID`).
- `zpool.Client.StartScrub`, `PauseScrub` (new) and `StopScrub` (previously stubs).
- `PropertySourceNone` for read-only and statistic properties.
- Examples: `pool-status`, `pool-import`, `scrub`.
- Integration tests for pools, datasets, import, discovery, status and scrub.
  Tests that create pools run only as root with `ZFS_TEST_DESTRUCTIVE=1`.
- GitHub Actions CI: gofmt and `go mod tidy` checks, and the full test suite in
  FreeBSD 14.5 and 15.1 VMs under QEMU.
- Community files: code of conduct, security policy, issue and PR templates,
  Dependabot configuration.

### Fixed

- Pool GUID was always 0.
- Property `Source` was always `local`; it now reports `local`, `default`,
  `inherited`, `temporary`, `received` or `none`.
- `Pool.Health` was derived from the status code and reported `ONLINE` for
  e.g. pools with offline devices; it now uses the pool's `health` property.
- `version.ProbeCapabilities` reported pools, datasets, snapshots and clones as
  unsupported.
- `version.Detect` returned placeholder ZFS and kernel versions.
- `zfs.Client.List(ctx, false)` returned every dataset; it now returns only the
  root dataset of each pool. `ListInPool(ctx, pool, true)` returned only the
  pool root, and snapshots of the pool root were missing. Listings are sorted
  by name.

### Changed

- `zpool.ScanStatus.ToProcess` was replaced by `Issued` (bytes verified so far),
  which is what `zpool status` reports. The old field had no libzfs counterpart.
- Minimum supported platform is FreeBSD 14 (OpenZFS 2.2+). Both OpenZFS 2.2
  (FreeBSD 14.x) and 2.3+ (FreeBSD 15.x) are supported.

## [0.1.0] - 2025-10-03

First alpha release: client creation, driver abstraction with the libzfs cgo
backend, and version probing.

[Unreleased]: https://github.com/zombocoder/go-freebsd-libzfs/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/zombocoder/go-freebsd-libzfs/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/zombocoder/go-freebsd-libzfs/releases/tag/v0.1.0
