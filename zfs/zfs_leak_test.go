//go:build freebsd && !no_libzfs

package zfs

import (
	"context"
	"fmt"
	"testing"

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
