//go:build freebsd && !no_libzfs

package zpool

import (
	"context"
	"testing"

	"github.com/zombocoder/go-freebsd-libzfs/internal/driver"
)

func TestListClosesPoolHandles(t *testing.T) {
	c := newClient(t)
	for i := 0; i < 200; i++ {
		if _, err := c.List(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if n := driver.OutstandingIterHandles(); n != 0 {
		t.Errorf("%d zpool handles were never closed", n)
	}
}
