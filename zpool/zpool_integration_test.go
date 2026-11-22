//go:build freebsd

package zpool

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

// firstPool returns the name of an imported pool, ignoring temporary pools
// created by destructive tests (prefix "gozfs_"), or skips the test.
func firstPool(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("zpool", "list", "-H", "-o", "name").Output()
	if err != nil {
		t.Skipf("zpool CLI unavailable: %v", err)
	}
	for _, name := range strings.Fields(string(out)) {
		if !strings.HasPrefix(name, "gozfs_") {
			return name
		}
	}
	t.Skip("no imported pools")
	return ""
}

func newClient(t *testing.T) *Client {
	t.Helper()
	c, err := New(context.Background())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

func TestListReportsPoolGUID(t *testing.T) {
	pool := firstPool(t)
	out, err := exec.Command("zpool", "get", "-Hp", "-o", "value", "guid", pool).Output()
	if err != nil {
		t.Fatalf("zpool get guid: %v", err)
	}
	want, err := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		t.Fatalf("parse guid: %v", err)
	}

	pools, err := newClient(t).List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, p := range pools {
		if p.Name == pool {
			if p.GUID != want {
				t.Errorf("pool %s GUID = %d, want %d", pool, p.GUID, want)
			}
			return
		}
	}
	t.Fatalf("pool %s not returned by List", pool)
}

func TestReadOnlyPoolPropertyHasNoneSource(t *testing.T) {
	pool := firstPool(t)
	props, err := newClient(t).GetProperties(context.Background(), pool, PropertyNameSize)
	if err != nil {
		t.Fatalf("GetProperties: %v", err)
	}
	p, ok := props[PropertyNameSize]
	if !ok {
		t.Fatalf("property %q missing", PropertyNameSize)
	}
	if p.Source != PropertySourceNone {
		t.Errorf("size source = %q, want %q", p.Source, PropertySourceNone)
	}
}

// Read-only: size and allocated must come back as exact numbers (compared
// with `zpool list -Hp`), while Value keeps the human-readable text.
func TestGetUint64PropertyReadsPoolSizes(t *testing.T) {
	pool := firstPool(t)
	c := newClient(t)
	ctx := context.Background()

	size, err := c.GetUint64Property(ctx, pool, PropertyNameSize)
	if err != nil {
		t.Fatalf("GetUint64Property(size): %v", err)
	}
	alloc, err := c.GetUint64Property(ctx, pool, PropertyNameAllocated)
	if err != nil {
		t.Fatalf("GetUint64Property(allocated): %v", err)
	}
	if size == 0 {
		t.Errorf("size = 0")
	}
	if alloc > size {
		t.Errorf("allocated %d > size %d", alloc, size)
	}

	// Compare with the CLI; the pool may change between the two reads, so
	// only the size (which does not change without an expand) is exact.
	out, err := exec.Command("zpool", "list", "-Hp", "-o", "size", pool).Output()
	if err != nil {
		t.Fatalf("zpool list: %v", err)
	}
	want, err := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		t.Fatalf("parse %q: %v", out, err)
	}
	if size != want {
		t.Errorf("size = %d, zpool list -p says %d", size, want)
	}

	props, err := c.GetProperties(ctx, pool, PropertyNameSize)
	if err != nil {
		t.Fatalf("GetProperties: %v", err)
	}
	p := props[PropertyNameSize]
	if !p.IsNumeric || p.Numeric != size {
		t.Errorf("size IsNumeric=%v Numeric=%d, want true/%d", p.IsNumeric, p.Numeric, size)
	}
	if s, ok := p.Value.(string); !ok || s == "" || s == strconv.FormatUint(size, 10) {
		t.Errorf("size Value = %#v, want the human-readable text", p.Value)
	}
}

// Properties beyond the old fixed list are readable, and text properties are
// not numeric.
func TestPoolPropertiesReadAnyNativeProperty(t *testing.T) {
	pool := firstPool(t)
	c := newClient(t)
	ctx := context.Background()

	props, err := c.GetProperties(ctx, pool, "capacity", "fragmentation", "expandsize", "freeing", "guid", "version", "health", "ashift")
	if err != nil {
		t.Fatalf("GetProperties: %v", err)
	}
	if p := props["guid"]; !p.IsNumeric || p.Numeric == 0 {
		t.Errorf("guid = %+v, want numeric non-zero", p)
	}
	if p := props["capacity"]; !p.IsNumeric || p.Numeric > 100 {
		t.Errorf("capacity = %+v, want numeric <= 100", p)
	}
	if p := props["health"]; p.IsNumeric {
		t.Errorf("health is numeric: %+v", p)
	}
	if _, ok := props["ashift"]; !ok {
		t.Errorf("ashift missing")
	}
	if _, err := c.GetStringProperty(ctx, pool, "health"); err != nil {
		t.Errorf("GetStringProperty(health): %v", err)
	}
	if _, err := c.GetUint64Property(ctx, pool, "health"); err == nil {
		t.Errorf("GetUint64Property(health) succeeded, want error")
	}
	if _, err := c.GetProperties(ctx, pool, "no-such-prop"); err == nil {
		t.Errorf("unknown property accepted")
	}
}
