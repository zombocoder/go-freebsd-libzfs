//go:build freebsd

package zfs

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"testing"

	zerrors "github.com/zombocoder/go-freebsd-libzfs/errors"
)

const (
	propAgent = "microvm-runner:agent"
	propVM    = "microvm-runner:vm"
)

func TestValidateUserPropertyName(t *testing.T) {
	long := "a:" + strings.Repeat("x", 254) // 256 bytes: too long
	ok := "a:" + strings.Repeat("x", 253)   // 255 bytes: longest allowed
	tests := []struct {
		name string
		ok   bool
	}{
		{propAgent, true},
		{"a:b", true},
		{"com.example:key_1-x.y", true},
		{"a:b:c", true},
		{ok, true},
		{"", false},
		{"nocolon", false},
		{"compression", false},
		{"bad space:x", false},
		{"Upper:x", false},
		{"a:Upper", false},
		{"bad/slash:x", false},
		{"bad@at:x", false},
		{"ünï:x", false},
		{long, false},
	}
	for _, tc := range tests {
		err := validateUserPropertyName(tc.name)
		if (err == nil) != tc.ok {
			t.Errorf("validateUserPropertyName(%q) = %v, want ok=%v", tc.name, err, tc.ok)
		}
		if err != nil && !errors.Is(err, zerrors.ErrInvalidArgument) && !errors.Is(err, zerrors.ErrNameTooLong) {
			t.Errorf("validateUserPropertyName(%q) error %v is neither ErrInvalidArgument nor ErrNameTooLong", tc.name, err)
		}
	}
}

func TestValidateUserPropertyValue(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value string
		ok    bool
	}{
		{"empty", "", true},
		{"8191 bytes", strings.Repeat("v", 8191), true},
		{"8192 bytes", strings.Repeat("v", 8192), false},
		{"NUL byte", "a\x00b", false},
	} {
		err := validateUserProperty(propAgent, tc.value)
		if (err == nil) != tc.ok {
			t.Errorf("%s: err = %v, want ok=%v", tc.name, err, tc.ok)
		}
		if err != nil && !errors.Is(err, zerrors.ErrInvalidArgument) {
			t.Errorf("%s: err %v is not ErrInvalidArgument", tc.name, err)
		}
	}
}

func TestUserPropertyInvalidNamesRejected(t *testing.T) {
	c := newClient(t)
	ctx := context.Background()
	if _, _, err := c.GetUserProperty(ctx, "nosuchpool", "nocolon"); !errors.Is(err, zerrors.ErrInvalidArgument) {
		t.Errorf("GetUserProperty err = %v, want ErrInvalidArgument", err)
	}
	if err := c.SetProperty(ctx, "nosuchpool", "bad space:x", "v"); !errors.Is(err, zerrors.ErrInvalidArgument) {
		t.Errorf("SetProperty err = %v, want ErrInvalidArgument", err)
	}
	if err := c.CreateSnapshot(ctx, "nosuchpool/a@b", false, map[string]string{"bad space:x": "v"}); !errors.Is(err, zerrors.ErrInvalidArgument) {
		t.Errorf("CreateSnapshot err = %v, want ErrInvalidArgument", err)
	}
	if err := c.CreateClone(ctx, "nosuchpool/a@b", "nosuchpool/c", map[string]string{"bad space:x": "v"}); !errors.Is(err, zerrors.ErrInvalidArgument) {
		t.Errorf("CreateClone err = %v, want ErrInvalidArgument", err)
	}
}

func TestUserPropertySetGetRoundTrip(t *testing.T) {
	pool := newTestPool(t)
	c := newClient(t)
	ctx := context.Background()

	ds := pool + "/fs"
	run(t, "zfs", "create", ds)
	child := ds + "/child"
	run(t, "zfs", "create", child)

	// Unset property: ok=false, no error.
	if v, ok, err := c.GetUserProperty(ctx, ds, propAgent); err != nil || ok || v != "" {
		t.Fatalf("unset: got (%q, %v, %v), want (\"\", false, nil)", v, ok, err)
	}

	if err := c.SetProperty(ctx, ds, propAgent, "agent-1"); err != nil {
		t.Fatalf("SetProperty: %v", err)
	}
	if got := run(t, "zfs", "get", "-H", "-o", "value", propAgent, ds); got != "agent-1" {
		t.Errorf("zfs get = %q, want agent-1", got)
	}

	v, ok, err := c.GetUserProperty(ctx, ds, propAgent)
	if err != nil || !ok || v != "agent-1" {
		t.Fatalf("GetUserProperty = (%q, %v, %v)", v, ok, err)
	}

	props, err := c.UserProperties(ctx, ds)
	if err != nil {
		t.Fatalf("UserProperties: %v", err)
	}
	if got := props[propAgent]; got.Value != "agent-1" || got.Source != PropertySourceLocal {
		t.Errorf("UserProperties[agent] = %+v, want agent-1/local", got)
	}

	// Inherited by the child.
	cprops, err := c.UserProperties(ctx, child)
	if err != nil {
		t.Fatalf("UserProperties(child): %v", err)
	}
	if got := cprops[propAgent]; got.Value != "agent-1" || got.Source != PropertySourceInherited {
		t.Errorf("child UserProperties[agent] = %+v, want agent-1/inherited", got)
	}
	if got := cliSource(t, child, propAgent); got != PropertySourceInherited {
		t.Errorf("cli source = %q", got)
	}

	// GetLocalUserProperty only reports values set on the dataset itself.
	if v, ok, err := c.GetLocalUserProperty(ctx, ds, propAgent); err != nil || !ok || v != "agent-1" {
		t.Errorf("GetLocalUserProperty(ds) = (%q, %v, %v)", v, ok, err)
	}
	if v, ok, err := c.GetLocalUserProperty(ctx, child, propAgent); err != nil || ok || v != "" {
		t.Errorf("GetLocalUserProperty(child) = (%q, %v, %v), want not ok", v, ok, err)
	}

	// Long values: 8191 bytes is the maximum.
	long := strings.Repeat("v", 8191)
	if err := c.SetProperty(ctx, ds, propVM, long); err != nil {
		t.Fatalf("SetProperty(8191 bytes): %v", err)
	}
	if got, ok, err := c.GetUserProperty(ctx, ds, propVM); err != nil || !ok || got != long {
		t.Errorf("8191-byte value did not round-trip (len %d, ok %v, err %v)", len(got), ok, err)
	}
	if err := c.SetProperty(ctx, ds, propVM, strings.Repeat("v", 8192)); !errors.Is(err, zerrors.ErrInvalidArgument) {
		t.Errorf("SetProperty(8192 bytes) err = %v, want ErrInvalidArgument", err)
	}

	// Empty value and values with spaces/unicode round-trip.
	for _, val := range []string{"", "with spaces", "ünï=cødé"} {
		if err := c.SetProperty(ctx, ds, propVM, val); err != nil {
			t.Fatalf("SetProperty(%q): %v", val, err)
		}
		got, ok, err := c.GetUserProperty(ctx, ds, propVM)
		if err != nil || !ok || got != val {
			t.Errorf("round-trip %q: got (%q, %v, %v)", val, got, ok, err)
		}
	}

	if _, _, err := c.GetUserProperty(ctx, pool+"/nonexistent", propAgent); err == nil {
		t.Error("GetUserProperty on missing dataset: want error")
	}
}

func TestGetPropertiesUserAndUnknown(t *testing.T) {
	pool := newTestPool(t)
	c := newClient(t)
	ctx := context.Background()

	ds := pool + "/fs"
	run(t, "zfs", "create", "-o", propAgent+"=a1", ds)

	props, err := c.GetProperties(ctx, ds, PropertyNameUsed, propAgent, "other:unset")
	if err != nil {
		t.Fatalf("GetProperties: %v", err)
	}
	if p, ok := props[propAgent]; !ok || p.Value != "a1" || p.Source != PropertySourceLocal {
		t.Errorf("props[agent] = %+v (present=%v), want a1/local", p, ok)
	}
	if _, ok := props["other:unset"]; ok {
		t.Error("unset user property should be absent from the result")
	}
	if _, ok := props[PropertyNameUsed]; !ok {
		t.Error("native property used missing")
	}

	if _, err := c.GetProperties(ctx, ds, "no_such_property"); !errors.Is(err, zerrors.ErrInvalidArgument) {
		t.Errorf("unknown native property: err = %v, want ErrInvalidArgument", err)
	}
	// A real native property outside the legacy fixed set is now readable.
	if v, err := c.GetStringProperty(ctx, ds, "atime"); err != nil || v == "" {
		t.Errorf("atime = (%q, %v)", v, err)
	}
}

func TestCreateCloneWithUserPropsAndRefreservation(t *testing.T) {
	pool := newTestPool(t)
	c := newClient(t)
	ctx := context.Background()

	run(t, "zfs", "create", pool+"/jobs")
	run(t, "zfs", "create", "-V", "8m", pool+"/base")
	run(t, "zfs", "snapshot", pool+"/base@snap")

	clone := pool + "/jobs/c1"
	err := c.CreateClone(ctx, pool+"/base@snap", clone, map[string]string{
		propAgent:        "agent-9",
		propVM:           "vm-1",
		"refreservation": "4M",
	})
	if err != nil {
		t.Fatalf("CreateClone: %v", err)
	}
	if got := run(t, "zfs", "get", "-Hp", "-o", "value", "refreservation", clone); got != "4194304" {
		t.Errorf("refreservation = %q, want 4194304", got)
	}
	for p, want := range map[string]string{propAgent: "agent-9", propVM: "vm-1"} {
		if got := run(t, "zfs", "get", "-H", "-o", "value", p, clone); got != want {
			t.Errorf("%s = %q, want %q", p, got, want)
		}
		if v, ok, err := c.GetUserProperty(ctx, clone, p); err != nil || !ok || v != want {
			t.Errorf("GetUserProperty(%s) = (%q, %v, %v)", p, v, ok, err)
		}
	}

	// A second clone owned by someone else, and an unrelated dataset.
	if err := c.CreateClone(ctx, pool+"/base@snap", pool+"/jobs/c2", map[string]string{propAgent: "other"}); err != nil {
		t.Fatalf("CreateClone c2: %v", err)
	}
	run(t, "zfs", "create", pool+"/jobs/plain")

	owned, err := c.FindByUserProperty(ctx, pool+"/jobs", propAgent, "agent-9")
	if err != nil {
		t.Fatalf("FindByUserProperty: %v", err)
	}
	if len(owned) != 1 || owned[0].Name != clone || owned[0].Type != TypeVolume {
		t.Errorf("FindByUserProperty = %+v, want only %s", owned, clone)
	}
}

func names(ds []Dataset) []string {
	out := make([]string, 0, len(ds))
	for _, d := range ds {
		out = append(out, d.Name)
	}
	return out
}

func TestFindByUserPropertyOnlyLocalDescendants(t *testing.T) {
	pool := newTestPool(t)
	c := newClient(t)
	ctx := context.Background()

	run(t, "zfs", "create", "-o", propAgent+"=A", pool+"/jobs")
	run(t, "zfs", "create", pool+"/jobs/inherits")
	run(t, "zfs", "create", pool+"/jobs/inherits/deeper")
	run(t, "zfs", "create", "-o", propAgent+"=A", pool+"/jobs/local")
	run(t, "zfs", "create", "-o", propAgent+"=B", pool+"/jobs/other")
	run(t, "zfs", "create", "-o", propAgent+"=A", pool+"/jobs/local/nested")
	run(t, "zfs", "create", "-V", "1m", "-o", propAgent+"=A", pool+"/jobs/vol")
	// Outside the parent.
	run(t, "zfs", "create", "-o", propAgent+"=A", pool+"/elsewhere")
	run(t, "zfs", "create", "-o", propAgent+"=A", pool+"/jobs2")
	// Snapshot carrying the property.
	run(t, "zfs", "snapshot", "-o", propAgent+"=A", pool+"/jobs/local@s")

	got, err := c.FindByUserProperty(ctx, pool+"/jobs", propAgent, "A")
	if err != nil {
		t.Fatalf("FindByUserProperty: %v", err)
	}
	want := []string{pool + "/jobs/local", pool + "/jobs/local/nested", pool + "/jobs/vol"}
	if strings.Join(names(got), ",") != strings.Join(want, ",") {
		t.Errorf("FindByUserProperty = %v, want %v", names(got), want)
	}

	// The inherited child must not look owned.
	if _, ok, _ := c.GetLocalUserProperty(ctx, pool+"/jobs/inherits", propAgent); ok {
		t.Error("inherited value reported as local")
	}
}

func TestFindByUserPropertyMissingParent(t *testing.T) {
	pool := newTestPool(t)
	_, err := newClient(t).FindByUserProperty(context.Background(), pool+"/nope", propAgent, "A")
	if !errors.Is(err, zerrors.ErrDatasetNotFound) {
		t.Errorf("err = %v, want ErrDatasetNotFound", err)
	}
}

func TestFindByUserPropertyToleratesConcurrentDestroy(t *testing.T) {
	pool := newTestPool(t)
	c := newClient(t)
	ctx := context.Background()

	run(t, "zfs", "create", pool+"/jobs")
	const n = 40
	for i := 0; i < n; i++ {
		run(t, "zfs", "create", "-o", propAgent+"=A", fmt.Sprintf("%s/jobs/d%02d", pool, i))
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < n; i += 2 {
			_ = exec.Command("zfs", "destroy", fmt.Sprintf("%s/jobs/d%02d", pool, i)).Run()
		}
	}()
	for i := 0; i < 20; i++ {
		got, err := c.FindByUserProperty(ctx, pool+"/jobs", propAgent, "A")
		if err != nil {
			t.Fatalf("FindByUserProperty during destroy: %v", err)
		}
		if len(got) < n/2 {
			t.Fatalf("found %d, fewer than the %d datasets that are never destroyed", len(got), n/2)
		}
	}
	<-done
}
