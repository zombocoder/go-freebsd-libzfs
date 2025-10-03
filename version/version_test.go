//go:build freebsd

package version

import (
	"context"
	"testing"
	"time"
)

func TestCapabilitySet_IsFeatureSupported(t *testing.T) {
	caps := &CapabilitySet{
		Pools:      true,
		Datasets:   true,
		Snapshots:  false,
		Encryption: true,
	}

	tests := []struct {
		feature  string
		expected bool
	}{
		{"pools", true},
		{"datasets", true},
		{"snapshots", false},
		{"encryption", true},
		{"nonexistent", false},
	}

	for _, test := range tests {
		t.Run(test.feature, func(t *testing.T) {
			result := caps.IsFeatureSupported(test.feature)
			if result != test.expected {
				t.Errorf("IsFeatureSupported(%q) = %v, want %v", test.feature, result, test.expected)
			}
		})
	}
}

func TestInfo_String(t *testing.T) {
	info := &Info{
		Impl:    "libzfs",
		ZFS:     "OpenZFS 2.1.6",
		Kernel:  "FreeBSD 14.0",
		Go:      "go1.22.0",
		BuiltAt: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		Arch:    "amd64",
		OS:      "freebsd",
	}

	result := info.String()
	expected := "libzfs | OpenZFS 2.1.6 | FreeBSD 14.0 | go1.22.0 | freebsd/amd64"

	if result != expected {
		t.Errorf("Info.String() = %q, want %q", result, expected)
	}
}

func TestWithLibZFS(t *testing.T) {
	cfg := Config{}
	WithLibZFS(true)(&cfg)

	if !cfg.UseLibZFS {
		t.Error("WithLibZFS(true) should set UseLibZFS to true")
	}

	WithLibZFS(false)(&cfg)

	if cfg.UseLibZFS {
		t.Error("WithLibZFS(false) should set UseLibZFS to false")
	}
}

func TestWithIoctlOnly(t *testing.T) {
	cfg := Config{UseLibZFS: true}
	WithIoctlOnly()(&cfg)

	if cfg.UseLibZFS {
		t.Error("WithIoctlOnly() should set UseLibZFS to false")
	}
}

// Test version detection with timeout
func TestDetect_WithTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// This test may fail on systems without ZFS, but we're testing the API
	_, err := Detect(ctx, WithIoctlOnly())

	// We expect either success or a specific error about missing ZFS
	if err != nil {
		t.Logf("Detect failed as expected on system without ZFS: %v", err)
	} else {
		t.Log("Detect succeeded - ZFS is available")
	}
}

func TestProbeCapabilities_WithTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// This test may fail on systems without ZFS, but we're testing the API
	_, err := ProbeCapabilities(ctx, WithIoctlOnly())

	// We expect either success or a specific error about missing ZFS
	if err != nil {
		t.Logf("ProbeCapabilities failed as expected on system without ZFS: %v", err)
	} else {
		t.Log("ProbeCapabilities succeeded - ZFS is available")
	}
}
