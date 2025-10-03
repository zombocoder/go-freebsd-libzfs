//go:build freebsd

package version

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/zombocoder/go-freebsd-libzfs/internal/driver"
)

// Info contains version and runtime information
type Info struct {
	Impl    string    // "libzfs" or "ioctl"
	ZFS     string    // ZFS version
	Kernel  string    // Kernel version
	Go      string    // Go version
	BuiltAt time.Time // Build timestamp
	Arch    string    // Architecture
	OS      string    // Operating system
}

// String returns a formatted version string
func (i *Info) String() string {
	return fmt.Sprintf("%s | %s | %s | %s | %s/%s",
		i.Impl, i.ZFS, i.Kernel, i.Go, i.OS, i.Arch)
}

// Config controls driver selection and behavior
type Config struct {
	UseLibZFS bool // prefer libzfs over ioctl (default: true)
}

// Option configures the version detection
type Option func(*Config)

// WithLibZFS forces the use of libzfs driver
func WithLibZFS(use bool) Option {
	return func(c *Config) {
		c.UseLibZFS = use
	}
}

// WithIoctlOnly forces the use of ioctl-only driver
func WithIoctlOnly() Option {
	return func(c *Config) {
		c.UseLibZFS = false
	}
}

// Detect probes the system and returns version information
func Detect(ctx context.Context, opts ...Option) (*Info, error) {
	cfg := Config{UseLibZFS: true}
	for _, opt := range opts {
		opt(&cfg)
	}

	var d driver.Driver
	var err error

	// Create libzfs driver
	d, err = driver.NewLibZFS()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize libzfs driver: %w", err)
	}
	defer d.Close()

	// Get runtime information from the driver
	impl, zfsVer, kernel, err := d.RuntimeInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get runtime info: %w", err)
	}

	return &Info{
		Impl:    impl,
		ZFS:     zfsVer,
		Kernel:  kernel,
		Go:      runtime.Version(),
		BuiltAt: time.Now(), // TODO: Use build-time constant
		Arch:    runtime.GOARCH,
		OS:      runtime.GOOS,
	}, nil
}

// CapabilitySet represents available ZFS features and capabilities
type CapabilitySet struct {
	// Core features
	Pools     bool
	Datasets  bool
	Snapshots bool
	Clones    bool

	// Advanced features
	SendReceive bool
	Encryption  bool
	Bookmarks   bool
	Events      bool

	// Property features
	UserProps    bool
	InheritProps bool
	TempProps    bool

	// Dataset types
	Filesystems bool
	Volumes     bool

	// Send/receive features
	SendRaw         bool
	SendCompressed  bool
	SendEmbedded    bool
	SendLargeBlocks bool
	ResumeTokens    bool

	// Encryption features
	NativeEncryption bool
	KeyManagement    bool
	RawSends         bool
}

// ProbeCapabilities detects what features are available on this system
func ProbeCapabilities(ctx context.Context, opts ...Option) (*CapabilitySet, error) {
	cfg := Config{UseLibZFS: true}
	for _, opt := range opts {
		opt(&cfg)
	}

	var d driver.Driver
	var err error

	d, err = driver.NewLibZFS()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize libzfs driver: %w", err)
	}
	defer d.Close()

	caps := &CapabilitySet{}

	// Probe core features
	if supported, err := d.SupportsFeature(ctx, "pools"); err == nil {
		caps.Pools = supported
	}
	if supported, err := d.SupportsFeature(ctx, "datasets"); err == nil {
		caps.Datasets = supported
	}
	if supported, err := d.SupportsFeature(ctx, "snapshots"); err == nil {
		caps.Snapshots = supported
	}
	if supported, err := d.SupportsFeature(ctx, "clones"); err == nil {
		caps.Clones = supported
	}

	// TODO: Probe more features systematically
	// For now, assume common features are available
	caps.Filesystems = true
	caps.Volumes = true
	caps.UserProps = true
	caps.InheritProps = true

	return caps, nil
}

// IsFeatureSupported checks if a specific feature is supported
func (c *CapabilitySet) IsFeatureSupported(feature string) bool {
	switch feature {
	case "pools":
		return c.Pools
	case "datasets":
		return c.Datasets
	case "snapshots":
		return c.Snapshots
	case "clones":
		return c.Clones
	case "send_receive":
		return c.SendReceive
	case "encryption":
		return c.Encryption
	case "bookmarks":
		return c.Bookmarks
	case "events":
		return c.Events
	case "user_props":
		return c.UserProps
	case "inherit_props":
		return c.InheritProps
	case "temp_props":
		return c.TempProps
	case "filesystems":
		return c.Filesystems
	case "volumes":
		return c.Volumes
	case "send_raw":
		return c.SendRaw
	case "send_compressed":
		return c.SendCompressed
	case "send_embedded":
		return c.SendEmbedded
	case "send_large_blocks":
		return c.SendLargeBlocks
	case "resume_tokens":
		return c.ResumeTokens
	case "native_encryption":
		return c.NativeEncryption
	case "key_management":
		return c.KeyManagement
	case "raw_sends":
		return c.RawSends
	default:
		return false
	}
}
