//go:build freebsd

package zpool

import (
	"context"
	"reflect"
	"testing"
)

// These tests create real pools; see requireDestructive in zpool_import_test.go.

func discover(t *testing.T, opts DiscoverOptions) []ImportablePool {
	t.Helper()
	pools, err := newClient(t).Discover(context.Background(), opts)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	return pools
}

func findPool(pools []ImportablePool, name string) *ImportablePool {
	for i := range pools {
		if pools[i].Name == name {
			return &pools[i]
		}
	}
	return nil
}

func TestDiscoverFindsExportedPool(t *testing.T) {
	p := newTestPool(t)
	guid := p.guid(t)
	p.export(t)

	got := findPool(discover(t, DiscoverOptions{SearchPaths: []string{p.dir}}), p.name)
	if got == nil {
		t.Fatalf("exported pool %s not discovered", p.name)
	}
	want := ImportablePool{
		Name:        p.name,
		GUID:        guid,
		State:       "EXPORTED",
		Health:      "ONLINE",
		DevicePaths: p.vdevs,
	}
	if !reflect.DeepEqual(*got, want) {
		t.Errorf("Discover returned\n %+v\nwant\n %+v", *got, want)
	}
}

func TestDiscoverReportsAllMirrorDevices(t *testing.T) {
	p := createTestPool(t, "gozfs_mirror", 2, true)
	p.export(t)

	got := findPool(discover(t, DiscoverOptions{SearchPaths: []string{p.dir}}), p.name)
	if got == nil {
		t.Fatalf("exported pool %s not discovered", p.name)
	}
	if !reflect.DeepEqual(got.DevicePaths, p.vdevs) {
		t.Errorf("DevicePaths = %v, want %v", got.DevicePaths, p.vdevs)
	}
}

func TestDiscoverSkipsImportedPools(t *testing.T) {
	p := newTestPool(t)

	if got := findPool(discover(t, DiscoverOptions{SearchPaths: []string{p.dir}}), p.name); got != nil {
		t.Errorf("imported pool %s reported as importable: %+v", p.name, *got)
	}
}

func TestDiscoverFiltersByPoolName(t *testing.T) {
	p := newTestPool(t)
	p.export(t)

	if pools := discover(t, DiscoverOptions{SearchPaths: []string{p.dir}, PoolName: "gozfs_other"}); len(pools) != 0 {
		t.Errorf("Discover with PoolName=gozfs_other returned %+v, want none", pools)
	}
	pools := discover(t, DiscoverOptions{SearchPaths: []string{p.dir}, PoolName: p.name})
	if len(pools) != 1 || pools[0].Name != p.name {
		t.Errorf("Discover with PoolName=%s returned %+v, want only that pool", p.name, pools)
	}
}

func TestDiscoverDestroyedPoolsOnlyWhenRequested(t *testing.T) {
	p := newTestPool(t)
	run(t, "zpool", "destroy", p.name)

	if got := findPool(discover(t, DiscoverOptions{SearchPaths: []string{p.dir}}), p.name); got != nil {
		t.Errorf("destroyed pool reported without IncludeDestroyed: %+v", *got)
	}

	got := findPool(discover(t, DiscoverOptions{SearchPaths: []string{p.dir}, IncludeDestroyed: true}), p.name)
	if got == nil {
		t.Fatalf("destroyed pool %s not discovered with IncludeDestroyed", p.name)
	}
	if got.State != "DESTROYED" {
		t.Errorf("State = %q, want DESTROYED", got.State)
	}
}

func TestDiscoveredPoolCanBeImported(t *testing.T) {
	p := newTestPool(t)
	p.export(t)
	c := newClient(t)

	pools, err := c.Discover(context.Background(), DiscoverOptions{SearchPaths: []string{p.dir}})
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if findPool(pools, p.name) == nil {
		t.Fatalf("pool %s not discovered", p.name)
	}
	if err := c.Import(context.Background(), p.name, ImportOptions{SearchPaths: []string{p.dir}}); err != nil {
		t.Fatalf("Import after Discover: %v", err)
	}
	if !poolImported(p.name) {
		t.Errorf("pool %s not imported", p.name)
	}
}
