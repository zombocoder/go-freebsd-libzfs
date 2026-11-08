//go:build freebsd

package zpool

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// These tests create real pools; see requireDestructive in zpool_import_test.go.

func getStatus(t *testing.T, c *Client, pool string) *Status {
	t.Helper()
	st, err := c.GetStatus(context.Background(), pool)
	if err != nil {
		t.Fatalf("GetStatus(%s): %v", pool, err)
	}
	return st
}

// waitScan polls the pool until its scan reaches want or the timeout expires.
func waitScan(t *testing.T, c *Client, pool string, want ScanState) *ScanStatus {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	for {
		st := getStatus(t, c, pool)
		if st.Scan != nil && st.Scan.State == want {
			return st.Scan
		}
		if time.Now().After(deadline) {
			t.Fatalf("scan of %s did not reach %v; last scan status: %+v", pool, want, st.Scan)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// mirrorLeaves returns the mirror vdev and its leaves of a single-mirror pool.
func mirrorLeaves(t *testing.T, st *Status) (VDevTree, []VDevTree) {
	t.Helper()
	if len(st.Config.Children) != 1 || len(st.Config.Children[0].Children) != 2 {
		t.Fatalf("vdev tree is not a single 2-way mirror: %+v", st.Config)
	}
	return st.Config.Children[0], st.Config.Children[0].Children
}

// suspendScans stops scrubs from making progress for the duration of the test,
// so their running/paused/canceled states can be observed.
func suspendScans(t *testing.T) {
	t.Helper()
	const tunable = "vfs.zfs.scan_suspend_progress"
	old := run(t, "sysctl", "-n", tunable)
	run(t, "sysctl", tunable+"=1")
	t.Cleanup(func() { _ = exec.Command("sysctl", tunable+"="+old).Run() })
}

func TestListReportsDegradedHealth(t *testing.T) {
	p := createTestPool(t, "gozfs_mirror", 2, true)
	run(t, "zpool", "offline", p.name, p.vdevs[1])

	pool, err := newClient(t).Get(context.Background(), p.name)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if pool.Health != HealthDegraded {
		t.Errorf("Health = %q, want %q", pool.Health, HealthDegraded)
	}
}

func TestGetStatusHealthyMirror(t *testing.T) {
	p := createTestPool(t, "gozfs_mirror", 2, true)
	st := getStatus(t, newClient(t), p.name)

	if st.Problem != "OK" || st.MsgID != "" {
		t.Errorf("Problem/MsgID = %q/%q, want OK/\"\"", st.Problem, st.MsgID)
	}
	if st.Scan != nil {
		t.Errorf("Scan = %+v, want nil for a pool that was never scanned", st.Scan)
	}

	root := st.Config
	if root.Name != p.name || root.Type != "root" || root.State != "ONLINE" {
		t.Errorf("root vdev = {Name:%q Type:%q State:%q}, want {%q root ONLINE}", root.Name, root.Type, root.State, p.name)
	}
	if len(root.Children) != 1 {
		t.Fatalf("root has %d children, want 1 mirror", len(root.Children))
	}
	mirror := root.Children[0]
	if mirror.Name != "mirror-0" || mirror.Type != "mirror" || mirror.State != "ONLINE" || mirror.GUID == 0 {
		t.Errorf("mirror vdev = %+v, want mirror-0 ONLINE with a GUID", mirror)
	}
	if len(mirror.Children) != 2 {
		t.Fatalf("mirror has %d children, want 2", len(mirror.Children))
	}
	for i, leaf := range mirror.Children {
		if leaf.Type != "file" || leaf.Path != p.vdevs[i] || leaf.Name != p.vdevs[i] || leaf.State != "ONLINE" {
			t.Errorf("leaf %d = {Name:%q Type:%q Path:%q State:%q}, want file %s ONLINE",
				i, leaf.Name, leaf.Type, leaf.Path, leaf.State, p.vdevs[i])
		}
	}
	if root.Stats.Space == 0 {
		t.Error("root vdev Stats.Space = 0, want pool capacity")
	}
}

func TestGetStatusOfflineDevice(t *testing.T) {
	p := createTestPool(t, "gozfs_mirror", 2, true)
	run(t, "zpool", "offline", p.name, p.vdevs[1])
	st := getStatus(t, newClient(t), p.name)

	if st.Problem != "OFFLINE_DEV" {
		t.Errorf("Problem = %q, want OFFLINE_DEV", st.Problem)
	}
	mirror, leaves := mirrorLeaves(t, st)
	if mirror.State != "DEGRADED" {
		t.Errorf("mirror State = %q, want DEGRADED", mirror.State)
	}
	if got := leaves[1].State; got != "OFFLINE" {
		t.Errorf("offlined leaf State = %q, want OFFLINE", got)
	}
}

func TestGetStatusSparesAndCache(t *testing.T) {
	p := newTestPool(t)
	spare := filepath.Join(p.dir, "spare0")
	cache := filepath.Join(p.dir, "cache0")
	run(t, "truncate", "-s", "128m", spare)
	run(t, "truncate", "-s", "128m", cache)
	run(t, "zpool", "add", p.name, "spare", spare, "cache", cache)

	st := getStatus(t, newClient(t), p.name)
	if len(st.Spares) != 1 || st.Spares[0].Path != spare || st.Spares[0].State != "AVAIL" {
		t.Errorf("Spares = %+v, want one AVAIL spare %s", st.Spares, spare)
	}
	if len(st.Cache) != 1 || st.Cache[0].Path != cache || st.Cache[0].State != "ONLINE" {
		t.Errorf("Cache = %+v, want one ONLINE cache device %s", st.Cache, cache)
	}
	for _, c := range st.Config.Children {
		if c.Path == spare || c.Path == cache {
			t.Errorf("spare/cache device %s listed among data vdevs", c.Path)
		}
	}
}

func TestScrubCompletes(t *testing.T) {
	p := newTestPool(t)
	c := newClient(t)

	if err := c.StartScrub(context.Background(), p.name); err != nil {
		t.Fatalf("StartScrub: %v", err)
	}
	scan := waitScan(t, c, p.name, ScanStateFinished)
	if scan.Function != ScanFunctionScrub {
		t.Errorf("Function = %v, want scrub", scan.Function)
	}
	if scan.EndTime == nil || scan.EndTime.Before(scan.StartTime) {
		t.Errorf("StartTime/EndTime = %v/%v, want end after start", scan.StartTime, scan.EndTime)
	}
	if scan.Errors != 0 {
		t.Errorf("Errors = %d, want 0", scan.Errors)
	}
}

func TestScrubPauseResumeStop(t *testing.T) {
	p := newTestPool(t)
	suspendScans(t)
	c := newClient(t)
	ctx := context.Background()

	if err := c.StartScrub(ctx, p.name); err != nil {
		t.Fatalf("StartScrub: %v", err)
	}
	waitScan(t, c, p.name, ScanStateScanning)

	if err := c.PauseScrub(ctx, p.name); err != nil {
		t.Fatalf("PauseScrub: %v", err)
	}
	waitScan(t, c, p.name, ScanStateSuspended)
	if out := run(t, "zpool", "status", p.name); !strings.Contains(out, "paused") {
		t.Errorf("zpool status does not report a paused scrub:\n%s", out)
	}

	if err := c.StartScrub(ctx, p.name); err != nil {
		t.Fatalf("StartScrub (resume): %v", err)
	}
	waitScan(t, c, p.name, ScanStateScanning)

	if err := c.StopScrub(ctx, p.name); err != nil {
		t.Fatalf("StopScrub: %v", err)
	}
	waitScan(t, c, p.name, ScanStateCanceled)
}

func TestStopScrubWithoutScrubFails(t *testing.T) {
	p := newTestPool(t)
	if err := newClient(t).StopScrub(context.Background(), p.name); err == nil {
		t.Error("StopScrub with no scrub running succeeded, want error")
	}
}

func TestScrubDetectsChecksumErrors(t *testing.T) {
	p := createTestPool(t, "gozfs_mirror", 2, true)
	c := newClient(t)

	// Fill part of the pool with data, flush it to disk, then overwrite the
	// middle of one mirror side (labels live in the first and last 512K).
	run(t, "dd", "if=/dev/random", "of="+filepath.Join(p.mntRoot, "data"), "bs=1m", "count=48")
	run(t, "zpool", "sync", p.name)
	run(t, "dd", "if=/dev/zero", "of="+p.vdevs[1], "bs=1m", "seek=8", "count=100", "conv=notrunc")

	if err := c.StartScrub(context.Background(), p.name); err != nil {
		t.Fatalf("StartScrub: %v", err)
	}
	waitScan(t, c, p.name, ScanStateFinished)

	st := getStatus(t, c, p.name)
	_, leaves := mirrorLeaves(t, st)
	leaf := leaves[1]
	if leaf.Stats.ChecksumErrors == 0 {
		t.Errorf("corrupted leaf ChecksumErrors = 0, want > 0 (stats %+v)", leaf.Stats)
	}
	if st.Problem != "FAILING_DEV" {
		t.Errorf("Problem = %q, want FAILING_DEV", st.Problem)
	}
	if st.MsgID == "" {
		t.Error("MsgID empty, want a ZFS-8000-* message id")
	}
	if st.Errors.DataErrors != 0 {
		t.Errorf("DataErrors = %d, want 0 (mirror repairs the damage)", st.Errors.DataErrors)
	}
	if want := fmt.Sprint(leaf.Stats.ChecksumErrors); !strings.Contains(run(t, "zpool", "status", "-p", p.name), want) {
		t.Errorf("zpool status does not show %s checksum errors", want)
	}
}

func TestGetStatusUnknownPoolFails(t *testing.T) {
	if os.Getenv("ZFS_TEST_DESTRUCTIVE") == "" {
		firstPool(t) // only needs ZFS to be present
	}
	if _, err := newClient(t).GetStatus(context.Background(), "gozfs_no_such_pool"); err == nil {
		t.Error("GetStatus of unknown pool succeeded, want error")
	}
}
