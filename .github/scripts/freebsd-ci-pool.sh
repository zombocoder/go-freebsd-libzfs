#!/bin/sh
# Loads ZFS and creates a small file-backed pool "citank" with a few datasets,
# a volume and a snapshot, so the read-only integration tests have something
# to compare against. Must run as root on a throwaway machine (CI VM).
#
# Destructive tests create and destroy their own pools (named gozfs_*) and do
# not touch citank.
set -eu

POOL=citank
DIR=/var/tmp/citank

kldload -n zfs

if zpool list -H "$POOL" >/dev/null 2>&1; then
	echo "pool $POOL already exists"
	exit 0
fi

mkdir -p "$DIR"
truncate -s 512m "$DIR/disk0"
zpool create -O mountpoint="/$POOL" "$POOL" "$DIR/disk0"
zfs create -o compression=lz4 "$POOL/data"
zfs create "$POOL/data/child"
zfs create -V 16m "$POOL/vol"
zfs snapshot "$POOL/data@ci"

zpool list "$POOL"
zfs list -r -t all "$POOL"
