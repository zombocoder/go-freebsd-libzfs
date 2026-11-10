//go:build freebsd

package zfs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// Destructive tests create a file-backed pool named gozfs_<prefix>_<pid>
// under /var/tmp and destroy it afterwards. They run only as root with
// ZFS_TEST_DESTRUCTIVE=1, e.g.:
//
//	doas env ZFS_TEST_DESTRUCTIVE=1 go test ./zfs -run UserProp

func run(t *testing.T, name string, args ...string) string {
	t.Helper()
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

var poolCounter atomic.Int32

// newTestPool creates a single-disk file-backed pool and registers cleanup
// once the pool exists. The pool is mounted below a temporary directory,
// never at /<name>.
func newTestPool(t *testing.T) string {
	t.Helper()
	if os.Getenv("ZFS_TEST_DESTRUCTIVE") != "1" {
		t.Skip("set ZFS_TEST_DESTRUCTIVE=1 to run tests that create pools")
	}
	if os.Geteuid() != 0 {
		t.Skip("creating pools requires root")
	}
	base, err := os.MkdirTemp("/var/tmp", "gozfs-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(base) })

	name := fmt.Sprintf("gozfs_zfs_%d_%d", os.Getpid(), poolCounter.Add(1))
	vdev := filepath.Join(base, "disk0")
	run(t, "truncate", "-s", "128m", vdev)
	run(t, "zpool", "create", "-O", "mountpoint="+filepath.Join(base, "mnt"), name, vdev)

	t.Cleanup(func() {
		if out, err := exec.Command("zpool", "destroy", "-f", name).CombinedOutput(); err != nil {
			t.Errorf("zpool destroy %s: %v\n%s (destroy it manually)", name, err, out)
		}
	})
	return name
}
