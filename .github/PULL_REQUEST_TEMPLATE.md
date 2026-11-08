## What and why

<!-- What does this change do, and what problem does it solve? Link issues with "Fixes #123". -->

## How it was tested

<!-- CI runs everything on FreeBSD 14.x and 15.x VMs. If you also tested locally, say how, e.g.:
     doas env ZFS_TEST_DESTRUCTIVE=1 go test -p 1 ./... on FreeBSD 15.1 -->

## Checklist

- [ ] New behavior is covered by tests (a failing test first, then the fix)
- [ ] Tests that create or modify pools use the throwaway file-backed pool helpers and are gated by `ZFS_TEST_DESTRUCTIVE=1`
- [ ] cgo code frees every C allocation and closes every libzfs handle
- [ ] Public API changes are documented (doc comments, README, examples) and listed in `CHANGELOG.md`
- [ ] `gofmt -l .` prints nothing
