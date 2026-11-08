//go:build freebsd

package zfs

import (
	"context"
	"os/exec"
	"sort"
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

// cliSource returns the property source as reported by `zfs get`,
// normalised to this package's PropertySource values.
func cliSource(t *testing.T, dataset, prop string) PropertySource {
	t.Helper()
	out, err := exec.Command("zfs", "get", "-H", "-o", "source", prop, dataset).Output()
	if err != nil {
		t.Fatalf("zfs get %s %s: %v", prop, dataset, err)
	}
	s := strings.TrimSpace(string(out))
	switch {
	case s == "-":
		return PropertySourceNone
	case strings.HasPrefix(s, "inherited"):
		return PropertySourceInherited
	default:
		return PropertySource(s)
	}
}

func TestReadOnlyDatasetPropertyHasNoneSource(t *testing.T) {
	pool := firstPool(t)
	props, err := newClient(t).GetProperties(context.Background(), pool, PropertyNameUsed)
	if err != nil {
		t.Fatalf("GetProperties: %v", err)
	}
	if got := props[PropertyNameUsed].Source; got != PropertySourceNone {
		t.Errorf("used source = %q, want %q", got, PropertySourceNone)
	}
}

func TestDatasetPropertySourceMatchesCLI(t *testing.T) {
	pool := firstPool(t)
	c := newClient(t)
	for _, prop := range []string{PropertyNameMountpoint, PropertyNameCompression, PropertyNameRecordsize, PropertyNameQuota} {
		t.Run(prop, func(t *testing.T) {
			props, err := c.GetProperties(context.Background(), pool, prop)
			if err != nil {
				t.Fatalf("GetProperties: %v", err)
			}
			want := cliSource(t, pool, prop)
			if got := props[prop].Source; got != want {
				t.Errorf("%s source = %q, want %q", prop, got, want)
			}
		})
	}
}

func TestListNonRecursiveReturnsOnlyRootDatasets(t *testing.T) {
	firstPool(t)
	datasets, err := newClient(t).List(context.Background(), false)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(datasets) == 0 {
		t.Fatal("List(false) returned no datasets")
	}
	for _, d := range datasets {
		if strings.ContainsAny(d.Name, "/@#") {
			t.Errorf("List(false) returned non-root dataset %q", d.Name)
		}
	}
}

func TestListRecursiveMatchesCLI(t *testing.T) {
	firstPool(t)
	out, err := exec.Command("zfs", "list", "-H", "-o", "name", "-t", "filesystem,volume,snapshot").Output()
	if err != nil {
		t.Fatalf("zfs list: %v", err)
	}
	want := strings.Fields(string(out))
	sort.Strings(want)

	datasets, err := newClient(t).List(context.Background(), true)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	got := make([]string, 0, len(datasets))
	for _, d := range datasets {
		got = append(got, d.Name)
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("List(true) names (sorted by name expected):\n got: %v\nwant: %v", got, want)
	}
}

func TestListInPoolNonRecursiveReturnsOnlyPoolRoot(t *testing.T) {
	pool := firstPool(t)
	datasets, err := newClient(t).ListInPool(context.Background(), pool, false)
	if err != nil {
		t.Fatalf("ListInPool: %v", err)
	}
	if len(datasets) != 1 || datasets[0].Name != pool {
		names := make([]string, 0, len(datasets))
		for _, d := range datasets {
			names = append(names, d.Name)
		}
		t.Errorf("ListInPool(%q, false) = %v, want [%s]", pool, names, pool)
	}
}

func TestListInPoolRecursiveMatchesCLI(t *testing.T) {
	pool := firstPool(t)
	out, err := exec.Command("zfs", "list", "-H", "-o", "name", "-r", "-t", "filesystem,volume,snapshot", pool).Output()
	if err != nil {
		t.Fatalf("zfs list: %v", err)
	}
	want := strings.Fields(string(out))
	sort.Strings(want)

	datasets, err := newClient(t).ListInPool(context.Background(), pool, true)
	if err != nil {
		t.Fatalf("ListInPool: %v", err)
	}
	got := make([]string, 0, len(datasets))
	for _, d := range datasets {
		got = append(got, d.Name)
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("ListInPool(%q, true):\n got: %v\nwant: %v", pool, got, want)
	}
}
