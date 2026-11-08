#!/bin/sh
# Prepares a FreeBSD VM for building and testing go-freebsd-libzfs:
#  - installs the Go toolchain;
#  - installs the OpenZFS headers that the cgo code compiles against.
#
# The libzfs headers are not part of the base system. They are taken from the
# src.txz distribution of the release that is running, so they match the
# installed libzfs. Only the two include trees used by the #cgo CFLAGS are
# extracted.
#
# The CI workflow caches the VM image after this script, so keep it limited to
# installing things; per-run state (kernel modules, test pools) belongs in
# freebsd-ci-pool.sh.
set -eu

ASSUME_ALWAYS_YES=yes pkg install -y go

OPENZFS=/usr/src/sys/contrib/openzfs
if [ -d "$OPENZFS/include" ] && [ -d "$OPENZFS/lib/libspl/include" ]; then
	echo "OpenZFS headers already present in $OPENZFS"
	exit 0
fi

release=$(freebsd-version -u | sed -E 's/-p[0-9]+$//')
arch=$(uname -m)
url="https://download.freebsd.org/releases/${arch}/${release}/src.txz"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
echo "Fetching $url"
fetch -o "$tmp/src.txz" "$url"
tar -xf "$tmp/src.txz" -C / \
	--include './usr/src/sys/contrib/openzfs/include/*' \
	--include './usr/src/sys/contrib/openzfs/lib/libspl/include/*'

test -f "$OPENZFS/include/libzfs.h"
echo "Installed OpenZFS headers for $release"
