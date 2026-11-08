# Security Policy

## Supported versions

Security fixes are made on the `main` branch and released in a new version.
Before 1.0, only the latest release receives fixes.

## Reporting a vulnerability

Please **do not open a public issue** for security problems.

Report them privately through GitHub:
[Report a vulnerability](https://github.com/zombocoder/go-freebsd-libzfs/security/advisories/new)
(Security tab → "Report a vulnerability").

Include what you can of:

- the affected function or package and the commit you tested;
- FreeBSD and OpenZFS versions (`freebsd-version -ku`, `zfs version`);
- steps or a minimal program that reproduces the problem;
- the impact you expect (memory corruption, privilege issue, data loss, ...).

You should get a first response within a week. Once a fix is available, the
advisory is published with credit to the reporter unless you prefer otherwise.

## Scope

This library calls libzfs through cgo and is typically used by privileged
processes. Of particular interest are:

- memory-safety bugs in the cgo layer (`internal/cgo`, `internal/driver`),
  such as use-after-free, leaks of handles or buffers, or out-of-bounds access;
- operations that act on a different pool, dataset or device than requested;
- input that is passed to libzfs without the validation callers would expect.

Bugs in libzfs or OpenZFS itself should be reported to
[OpenZFS](https://github.com/openzfs/zfs/security) or the
[FreeBSD security team](https://www.freebsd.org/security/).
