//go:build freebsd

package version

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

func TestDetectReportsRealZFSVersion(t *testing.T) {
	out, err := exec.Command("zfs", "version").Output()
	if err != nil {
		t.Skipf("zfs CLI unavailable: %v", err)
	}
	// First line is the userland version, e.g. "zfs-2.2.7-FreeBSD_g...".
	wantZFS := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])

	info, err := Detect(context.Background())
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if info.ZFS != wantZFS {
		t.Errorf("ZFS = %q, want %q", info.ZFS, wantZFS)
	}
}

func TestDetectReportsKernelRelease(t *testing.T) {
	out, err := exec.Command("uname", "-r").Output()
	if err != nil {
		t.Fatalf("uname -r: %v", err)
	}
	release := strings.TrimSpace(string(out))

	info, err := Detect(context.Background())
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if !strings.Contains(info.Kernel, release) {
		t.Errorf("Kernel = %q, want it to contain %q", info.Kernel, release)
	}
}

func TestProbeCapabilitiesReportsCoreFeatures(t *testing.T) {
	caps, err := ProbeCapabilities(context.Background())
	if err != nil {
		t.Fatalf("ProbeCapabilities: %v", err)
	}
	for name, ok := range map[string]bool{
		"pools":     caps.Pools,
		"datasets":  caps.Datasets,
		"snapshots": caps.Snapshots,
		"clones":    caps.Clones,
	} {
		if !ok {
			t.Errorf("capability %q = false, want true", name)
		}
	}
}
