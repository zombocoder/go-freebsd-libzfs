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
// The Go runtime sends SIGURG to preempt goroutines and garbage collection
// preempts constantly, so listings under allocation and CPU pressure must
// never fail with "signal received".
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
	for i := 0; i < 2*runtime.GOMAXPROCS(0); i++ {
		wg.Add(1)
		go func() { // CPU spinner, preempted by signal
			defer wg.Done()
			x := 0
			for {
				select {
				case <-stop:
					return
				default:
					x++
				}
			}
		}()
	}
	wg.Add(1)
	go func() { // allocation churn plus forced collections
		defer wg.Done()
		var sink [][]byte
		for {
			select {
			case <-stop:
				return
			default:
				sink = append(sink[:0], make([]byte, 1<<20))
				runtime.GC()
			}
		}
	}()
	wg.Add(1)
	go func() { // process-directed signals, delivered to whichever thread accepts them
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
	defer func() { close(stop); wg.Wait() }()

	deadline := time.Now().Add(10 * time.Second)
	for i := 0; i < 3000 && time.Now().Before(deadline); i++ {
		if _, err := c.List(ctx, true); err != nil {
			t.Fatalf("List #%d: %v", i, err)
		}
		if _, err := c.FindByUserProperty(ctx, pool+"/jobs", propAgent, "A"); err != nil {
			t.Fatalf("FindByUserProperty #%d: %v", i, err)
		}
		if _, _, err := c.GetUserProperty(ctx, pool+"/jobs/d1", propAgent); err != nil {
			t.Fatalf("GetUserProperty #%d: %v", i, err)
		}
	}
}
