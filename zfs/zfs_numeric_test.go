//go:build freebsd

package zfs

import (
	"context"
	"errors"
	"strconv"
	"testing"

	zerrors "github.com/zombocoder/go-freebsd-libzfs/errors"
)

func TestNumericPropertiesMatchCLI(t *testing.T) {
	pool := newTestPool(t)
	c := newClient(t)
	ctx := context.Background()

	run(t, "zfs", "create", "-o", "quota=50M", pool+"/fs")
	run(t, "zfs", "create", "-V", "8m", "-o", "refreservation=6M", pool+"/vol")

	for _, tc := range []struct{ ds, prop string }{
		{pool + "/fs", "used"},
		{pool + "/fs", "available"},
		{pool + "/fs", PropertyNameReferenced},
		{pool + "/fs", PropertyNameQuota},
		{pool + "/fs", PropertyNameRecordsize},
		{pool, "available"},
		{pool + "/vol", "volsize"},
		{pool + "/vol", PropertyNameRefreserv},
		{pool + "/vol", "used"},
	} {
		want, err := strconv.ParseUint(run(t, "zfs", "get", "-Hp", "-o", "value", tc.prop, tc.ds), 10, 64)
		if err != nil {
			t.Fatalf("%s %s: cli value: %v", tc.ds, tc.prop, err)
		}
		got, err := c.GetUint64Property(ctx, tc.ds, tc.prop)
		if err != nil {
			t.Errorf("%s %s: GetUint64Property: %v", tc.ds, tc.prop, err)
			continue
		}
		// "available" and "used" can move between the two reads only if the
		// pool changes; the test pool is idle.
		if got != want {
			t.Errorf("%s %s = %d, want %d", tc.ds, tc.prop, got, want)
		}
	}

	if got, _ := c.GetUint64Property(ctx, pool+"/vol", PropertyNameRefreserv); got != 6*1024*1024 {
		t.Errorf("refreservation = %d, want 6291456", got)
	}

	// The string form is unchanged and non-numeric properties say so.
	props, err := c.GetProperties(ctx, pool+"/fs", PropertyNameMountpoint, PropertyNameQuota)
	if err != nil {
		t.Fatal(err)
	}
	if p := props[PropertyNameMountpoint]; p.IsNumeric {
		t.Errorf("mountpoint reported numeric: %+v", p)
	}
	if p := props[PropertyNameQuota]; !p.IsNumeric || p.Numeric != 50*1024*1024 || p.Value != "50M" {
		t.Errorf("quota = %+v, want numeric 52428800 and text 50M", p)
	}
	if _, err := c.GetUint64Property(ctx, pool+"/fs", PropertyNameMountpoint); err == nil {
		t.Error("GetUint64Property(mountpoint): want error")
	}
}

func TestMissingDatasetMatchesErrDatasetNotFound(t *testing.T) {
	pool := newTestPool(t)
	c := newClient(t)
	ctx := context.Background()

	missing := pool + "/missing"
	run(t, "zfs", "create", pool+"/real")
	run(t, "zfs", "snapshot", pool+"/real@s")

	checks := map[string]error{
		"Get": func() error { _, err := c.Get(ctx, missing); return err }(),
		"GetProperties": func() error {
			_, err := c.GetProperties(ctx, missing, PropertyNameUsed)
			return err
		}(),
		"GetUint64Property": func() error {
			_, err := c.GetUint64Property(ctx, missing, PropertyNameUsed)
			return err
		}(),
		"UserProperties": func() error { _, err := c.UserProperties(ctx, missing); return err }(),
		"GetUserProperty": func() error {
			_, _, err := c.GetUserProperty(ctx, missing, propAgent)
			return err
		}(),
		"SetProperty":         c.SetProperty(ctx, missing, "compression", "lz4"),
		"SetProperty(user)":   c.SetProperty(ctx, missing, propAgent, "x"),
		"Destroy":             c.Destroy(ctx, missing, false),
		"Destroy(recursive)":  c.Destroy(ctx, missing, true),
		"DestroySnapshot":     c.DestroySnapshot(ctx, pool+"/real@nosnap"),
		"DestroySnapshot(ds)": c.DestroySnapshot(ctx, missing+"@s"),
		"DestroyClone":        c.DestroyClone(ctx, missing, false),
		"CreateClone":         c.CreateClone(ctx, pool+"/real@nosnap", pool+"/clone", nil),
		"CreateClone(ds)":     c.CreateClone(ctx, missing+"@s", pool+"/clone", nil),
		"CreateClone(parent)": c.CreateClone(ctx, pool+"/real@s", pool+"/noparent/clone", nil),
		"CreateSnapshot":      c.CreateSnapshot(ctx, missing+"@s", false, nil),
		"CreateFilesystem":    c.CreateFilesystem(ctx, pool+"/noparent/fs", nil),
		"FindByUserProperty": func() error {
			_, err := c.FindByUserProperty(ctx, missing, propAgent, "x")
			return err
		}(),
	}
	for name, err := range checks {
		if !errors.Is(err, zerrors.ErrDatasetNotFound) {
			t.Errorf("%s: err = %v, want ErrDatasetNotFound", name, err)
		}
	}

	// Other failures must not look like "not found".
	run(t, "zfs", "create", pool+"/parent")
	run(t, "zfs", "create", pool+"/parent/child")
	if err := c.Destroy(ctx, pool+"/parent", false); err == nil || errors.Is(err, zerrors.ErrDatasetNotFound) {
		t.Errorf("destroying a dataset with children: err = %v, want a non-NotFound error", err)
	}
}
