//go:build freebsd && !no_libzfs

package zfs

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/zombocoder/go-freebsd-libzfs/internal/driver"
)

// libzfs hands every handle given to an iteration callback over to the
// callback, which must close it. The driver counts handles received and not
// yet closed; repeated listings must leave that count at zero.
func TestIterationClosesHandles(t *testing.T) {
	pool := newTestPool(t)
	c := newClient(t)
	ctx := context.Background()

	run(t, "zfs", "create", pool+"/jobs")
	for i := 0; i < 5; i++ {
		run(t, "zfs", "create", "-o", propAgent+"=A", fmt.Sprintf("%s/jobs/d%d", pool, i))
	}
	run(t, "zfs", "snapshot", pool+"/jobs/d0@s")

	if n := driver.OutstandingIterHandles(); n != 0 {
		t.Fatalf("handles outstanding before the test: %d", n)
	}
	for i := 0; i < 2000; i++ {
		if _, err := c.FindByUserProperty(ctx, pool+"/jobs", propAgent, "A"); err != nil {
			t.Fatal(err)
		}
		if i%10 == 0 {
			if _, err := c.List(ctx, true); err != nil {
				t.Fatal(err)
			}
			if _, err := c.List(ctx, false); err != nil {
				t.Fatal(err)
			}
		}
	}
	if n := driver.OutstandingIterHandles(); n != 0 {
		t.Errorf("%d libzfs handles were never closed", n)
	}
}

// libzfs does not retry an ioctl that fails with EINTR (EZFS_INTR), so the
// driver must keep signals away from the thread while it is inside libzfs.
// Process-directed signals are delivered to any thread that accepts them,
// including one blocked in a libzfs ioctl. The SIGURG flood below is what
// reproduces the failure ("signal received", errno 2035) on the unmasked
// driver; the Go runtime's own preemption and GC signals alone did not
// reproduce it on a 16-CPU VM, so a light CPU/GC load is kept only as extra
// realism.
func TestListingSurvivesRuntimeSignals(t *testing.T) {
	pool := newTestPool(t)
	c := newClient(t)
	ctx := context.Background()

	run(t, "zfs", "create", pool+"/jobs")
	for i := 0; i < 10; i++ {
		run(t, "zfs", "create", "-o", propAgent+"=A", fmt.Sprintf("%s/jobs/d%d", pool, i))
	}

	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { // the essential part: flood the process with SIGURG
		defer wg.Done()
		pid := os.Getpid()
		for {
			select {
			case <-stop:
				return
			default:
				_ = syscall.Kill(pid, syscall.SIGURG)
			}
		}
	}()
	go func() { // light GC pressure
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				_ = make([]byte, 1<<20)
				runtime.GC()
			}
		}
	}()
	defer func() { close(stop); wg.Wait() }()

	const minIterations = 100
	deadline := time.Now().Add(10 * time.Second)
	done := 0
	for done < 3000 && time.Now().Before(deadline) {
		if _, err := c.List(ctx, true); err != nil {
			t.Fatalf("List #%d: %v", done, err)
		}
		if _, err := c.FindByUserProperty(ctx, pool+"/jobs", propAgent, "A"); err != nil {
			t.Fatalf("FindByUserProperty #%d: %v", done, err)
		}
		if _, _, err := c.GetUserProperty(ctx, pool+"/jobs/d1", propAgent); err != nil {
			t.Fatalf("GetUserProperty #%d: %v", done, err)
		}
		done++
	}
	t.Logf("%d iterations under the SIGURG flood", done)
	if done < minIterations {
		t.Fatalf("only %d iterations ran before the deadline, want at least %d", done, minIterations)
	}
}
