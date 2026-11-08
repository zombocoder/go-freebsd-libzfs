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
