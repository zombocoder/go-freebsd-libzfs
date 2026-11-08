# Changelog

All notable changes to this project are documented here. The format is based
on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project
aims to follow [Semantic Versioning](https://semver.org/) once it is tagged.

## [Unreleased]

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
- Minimum supported platform is FreeBSD 14 (OpenZFS 2.2+).
