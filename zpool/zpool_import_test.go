//go:build freebsd

package zpool

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// These tests create, export and destroy real pools backed by files.
// They run only as root with ZFS_TEST_DESTRUCTIVE=1, e.g.:
//
//	doas env ZFS_TEST_DESTRUCTIVE=1 go test ./zpool -run Import

func requireDestructive(t *testing.T) {
	t.Helper()
	if os.Getenv("ZFS_TEST_DESTRUCTIVE") != "1" {
		t.Skip("set ZFS_TEST_DESTRUCTIVE=1 to run tests that create pools")
	}
	if os.Geteuid() != 0 {
		t.Skip("pool tests require root")
	}
}

func run(t *testing.T, name string, args ...string) string {
	t.Helper()
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// testPool is a file-backed pool created for a single test.
type testPool struct {
	name    string
	dir     string   // directory holding the vdev files
	vdevs   []string // paths of the vdev files
	mntRoot string   // mountpoint of the root dataset
}

// newTestPool creates a single-disk file-backed pool and registers cleanup
// that destroys it under its original name and any extra names the test may
// rename it to.
func newTestPool(t *testing.T, extraNames ...string) *testPool {
	t.Helper()
	return createTestPool(t, "gozfs_test", 1, false, extraNames...)
}

// createTestPool creates a pool named <prefix>_<pid> from disks file vdevs,
// as a mirror when mirror is true.
func createTestPool(t *testing.T, prefix string, disks int, mirror bool, extraNames ...string) *testPool {
	t.Helper()
	requireDestructive(t)

	base, err := os.MkdirTemp("/var/tmp", "gozfs-test-")
	if err != nil {
		t.Fatal(err)
	}
	p := &testPool{
		name:    fmt.Sprintf("%s_%d", prefix, os.Getpid()),
		dir:     filepath.Join(base, "vdevs"),
		mntRoot: filepath.Join(base, "mnt"),
	}
	if err := os.MkdirAll(p.dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < disks; i++ {
		vdev := filepath.Join(p.dir, fmt.Sprintf("disk%d", i))
		run(t, "truncate", "-s", "128m", vdev)
		p.vdevs = append(p.vdevs, vdev)
	}

	t.Cleanup(func() {
		for _, n := range append([]string{p.name}, extraNames...) {
			_ = exec.Command("zpool", "destroy", "-f", n).Run()
		}
		_ = os.RemoveAll(base)
	})

	args := []string{"create", "-O", "mountpoint=" + p.mntRoot, p.name}
	if mirror {
		args = append(args, "mirror")
	}
	run(t, "zpool", append(args, p.vdevs...)...)
	return p
}

func (p *testPool) guid(t *testing.T) uint64 {
	t.Helper()
	g, err := strconv.ParseUint(run(t, "zpool", "get", "-Hp", "-o", "value", "guid", p.name), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func (p *testPool) export(t *testing.T) {
	t.Helper()
	run(t, "zpool", "export", p.name)
}

func poolImported(name string) bool {
	return exec.Command("zpool", "list", "-H", name).Run() == nil
}

func TestImportExportedPool(t *testing.T) {
	p := newTestPool(t)
	p.export(t)

	err := newClient(t).Import(context.Background(), p.name, ImportOptions{SearchPaths: []string{p.dir}})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if !poolImported(p.name) {
		t.Fatalf("pool %s not imported", p.name)
	}
	if got := run(t, "zpool", "list", "-H", "-o", "health", p.name); got != "ONLINE" {
		t.Errorf("health = %q, want ONLINE", got)
	}
}

func TestImportMountsDatasets(t *testing.T) {
	p := newTestPool(t)
	p.export(t)

	if err := newClient(t).Import(context.Background(), p.name, ImportOptions{SearchPaths: []string{p.dir}}); err != nil {
		t.Fatalf("Import: %v", err)
	}
	if got := run(t, "zfs", "get", "-H", "-o", "value", "mounted", p.name); got != "yes" {
		t.Errorf("root dataset mounted = %q, want yes", got)
	}
}

func TestImportWithNewName(t *testing.T) {
	newName := fmt.Sprintf("gozfs_renamed_%d", os.Getpid())
	p := newTestPool(t, newName)
	p.export(t)

	err := newClient(t).Import(context.Background(), p.name, ImportOptions{
		SearchPaths: []string{p.dir},
		NewName:     newName,
	})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if !poolImported(newName) {
		t.Errorf("pool not imported as %s", newName)
	}
	if poolImported(p.name) {
		t.Errorf("pool still imported under old name %s", p.name)
	}
}

func TestImportWithAltRoot(t *testing.T) {
	p := newTestPool(t)
	p.export(t)
	altRoot := filepath.Join(filepath.Dir(p.dir), "altroot")

	err := newClient(t).Import(context.Background(), p.name, ImportOptions{
		SearchPaths: []string{p.dir},
		AltRoot:     altRoot,
	})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if got := run(t, "zpool", "get", "-H", "-o", "value", "altroot", p.name); got != altRoot {
		t.Errorf("altroot = %q, want %q", got, altRoot)
	}
}

func TestImportDestroyedPool(t *testing.T) {
	p := newTestPool(t)
	run(t, "zpool", "destroy", p.name)

	c := newClient(t)
	opts := ImportOptions{SearchPaths: []string{p.dir}}
	if err := c.Import(context.Background(), p.name, opts); err == nil {
		t.Fatal("Import of destroyed pool without Destroyed succeeded, want error")
	}

	opts.Destroyed = true
	if err := c.Import(context.Background(), p.name, opts); err != nil {
		t.Fatalf("Import with Destroyed: %v", err)
	}
	if !poolImported(p.name) {
		t.Errorf("destroyed pool %s not imported", p.name)
	}
}

func TestImportUnknownPoolFails(t *testing.T) {
	p := newTestPool(t)
	err := newClient(t).Import(context.Background(), "gozfs_no_such_pool", ImportOptions{SearchPaths: []string{p.dir}})
	if err == nil {
		t.Fatal("Import of unknown pool succeeded, want error")
	}
	if !strings.Contains(err.Error(), "no such pool") {
		t.Errorf("error = %q, want it to mention \"no such pool\"", err)
	}
}

func TestImportUnknownPoolDefaultSearchPathsFails(t *testing.T) {
	requireDestructive(t)
	err := newClient(t).Import(context.Background(), "gozfs_no_such_pool", ImportOptions{})
	if err == nil {
		t.Fatal("Import of unknown pool succeeded, want error")
	}
	if !strings.Contains(err.Error(), "no such pool") {
		t.Errorf("error = %q, want it to mention \"no such pool\"", err)
	}
}
