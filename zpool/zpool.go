//go:build freebsd

package zpool

import (
	"context"
	"fmt"
	"time"

	"github.com/zombocoder/go-freebsd-libzfs/internal/driver"
)

// Client provides access to ZFS pool operations
type Client struct {
	d driver.Driver
}

// Option configures Client creation
type Option func(*config)

type config struct {
	useLibzfs bool
}

// WithLibZFS forces the use of libzfs driver (default)
func WithLibZFS() Option {
	return func(c *config) {
		c.useLibzfs = true
	}
}

// WithIoctlOnly forces the use of ioctl-only driver
func WithIoctlOnly() Option {
	return func(c *config) {
		c.useLibzfs = false
	}
}

// New creates a new ZFS pool client
func New(ctx context.Context, opts ...Option) (*Client, error) {
	cfg := config{useLibzfs: true}
	for _, opt := range opts {
		opt(&cfg)
	}

	var d driver.Driver
	var err error

	d, err = driver.NewLibZFS()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize libzfs driver: %w", err)
	}

	return &Client{d: d}, nil
}

// Close releases resources held by the client
func (c *Client) Close() error {
	if c.d != nil {
		return c.d.Close()
	}
	return nil
}

// Pool represents a ZFS storage pool
type Pool struct {
	Name   string
	GUID   uint64
	Health Health
	State  State

	// TODO: Add more pool fields as needed
	// Size, Allocated, Free, etc.
}

// Health represents pool health status
type Health string

const (
	HealthOnline   Health = "ONLINE"
	HealthDegraded Health = "DEGRADED"
	HealthFaulted  Health = "FAULTED"
	HealthOffline  Health = "OFFLINE"
	HealthRemoved  Health = "REMOVED"
	HealthUnavail  Health = "UNAVAIL"
)

// State represents pool state
type State string

const (
	StateActive            State = "ACTIVE"
	StateExported          State = "EXPORTED"
	StateDestroyed         State = "DESTROYED"
	StateSpare             State = "SPARE"
	StateL2Cache           State = "L2CACHE"
	StateUninitialized     State = "UNINITIALIZED"
	StateUnavail           State = "UNAVAIL"
	StatePotentiallyActive State = "POTENTIALLY_ACTIVE"
)

// List returns all available pools
func (c *Client) List(ctx context.Context) ([]Pool, error) {
	if c.d == nil {
		return nil, fmt.Errorf("client is closed")
	}

	poolInfos, err := c.d.ListPools(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list pools: %w", err)
	}

	pools := make([]Pool, 0, len(poolInfos))
	for _, info := range poolInfos {
		pool := Pool{
			Name:   info.Name,
			GUID:   info.GUID,
			Health: Health(info.Health),
			State:  State(info.State),
		}
		pools = append(pools, pool)
	}

	return pools, nil
}

// Get retrieves a specific pool by name
func (c *Client) Get(ctx context.Context, name string) (*Pool, error) {
	pools, err := c.List(ctx)
	if err != nil {
		return nil, err
	}

	for _, pool := range pools {
		if pool.Name == name {
			return &pool, nil
		}
	}

	return nil, fmt.Errorf("pool %q not found", name)
}

// DiscoverOptions represents options for pool discovery
type DiscoverOptions struct {
	SearchPaths      []string // Directories to search for pool devices
	IncludeDestroyed bool     // Include destroyed pools in results
	PoolName         string   // Search for specific pool name only
}

// ImportablePool represents a pool that can be imported
type ImportablePool struct {
	Name        string
	GUID        uint64
	State       string
	Health      string
	DevicePaths []string // Devices that make up this pool
}

// Discover finds pools that can be imported
func (c *Client) Discover(ctx context.Context, opts DiscoverOptions) ([]ImportablePool, error) {
	if c.d == nil {
		return nil, fmt.Errorf("client is closed")
	}

	// TODO: Implement pool discovery via driver
	// This would call zpool_find_import through the driver interface
	// For now, return empty list since it requires additional driver method
	return []ImportablePool{}, nil
}

// CreateOptions represents options for pool creation
type CreateOptions struct {
	Properties   map[string]string // Pool properties to set
	FsProperties map[string]string // Root filesystem properties to set
	AltRoot      string            // Alternative root directory
	Force        bool              // Force creation even if devices are in use
}

// Create creates a new ZFS pool
func (c *Client) Create(ctx context.Context, poolName string, vdevs []string, opts CreateOptions) error {
	if c.d == nil {
		return fmt.Errorf("client is closed")
	}

	if len(vdevs) == 0 {
		return fmt.Errorf("at least one vdev is required")
	}

	driverOpts := driver.CreateOptions{
		Properties:   opts.Properties,
		FsProperties: opts.FsProperties,
		AltRoot:      opts.AltRoot,
		Force:        opts.Force,
	}

	return c.d.CreatePool(ctx, poolName, vdevs, driverOpts)
}

// Property represents a pool property with its value and metadata
type Property[T any] struct {
	Name     string
	Value    T
	Source   PropertySource
	Received bool
}

// PropertySource indicates where a property value originates
type PropertySource string

const (
	PropertySourceLocal     PropertySource = "local"
	PropertySourceInherited PropertySource = "inherited"
	PropertySourceDefault   PropertySource = "default"
	PropertySourceTemporary PropertySource = "temporary"
	PropertySourceReceived  PropertySource = "received"
)

// Common pool property names
const (
	PropertyNameSize          = "size"
	PropertyNameCapacity      = "capacity"
	PropertyNameAltroot       = "altroot"
	PropertyNameHealth        = "health"
	PropertyNameGUID          = "guid"
	PropertyNameVersion       = "version"
	PropertyNameBootfs        = "bootfs"
	PropertyNameDelegation    = "delegation"
	PropertyNameAutoreplace   = "autoreplace"
	PropertyNameCachefile     = "cachefile"
	PropertyNameFailmode      = "failmode"
	PropertyNameListsnaps     = "listsnapshots"
	PropertyNameAutoexpand    = "autoexpand"
	PropertyNameDedupditto    = "dedupditto"
	PropertyNameDedupratio    = "dedupratio"
	PropertyNameFree          = "free"
	PropertyNameAllocated     = "allocated"
	PropertyNameReadonly      = "readonly"
	PropertyNameComment       = "comment"
	PropertyNameExpandsz      = "expandsize"
	PropertyNameFreeing       = "freeing"
	PropertyNameFragmentation = "fragmentation"
)

// GetProperties retrieves multiple properties for a pool
func (c *Client) GetProperties(ctx context.Context, poolName string, propNames ...string) (map[string]Property[any], error) {
	if c.d == nil {
		return nil, fmt.Errorf("client is closed")
	}

	propInfos, err := c.d.GetPoolProps(ctx, poolName, propNames)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool properties: %w", err)
	}

	properties := make(map[string]Property[any])
	for name, info := range propInfos {
		prop := Property[any]{
			Name:     info.Name,
			Value:    info.Value,
			Source:   PropertySource(info.Source.String()),
			Received: info.Received,
		}
		properties[name] = prop
	}

	return properties, nil
}

// GetStringProperty retrieves a string property value
func (c *Client) GetStringProperty(ctx context.Context, poolName, propName string) (string, error) {
	props, err := c.GetProperties(ctx, poolName, propName)
	if err != nil {
		return "", err
	}

	prop, exists := props[propName]
	if !exists {
		return "", fmt.Errorf("property %q not found", propName)
	}

	if str, ok := prop.Value.(string); ok {
		return str, nil
	}

	return fmt.Sprintf("%v", prop.Value), nil
}

// GetUint64Property retrieves a uint64 property value
func (c *Client) GetUint64Property(ctx context.Context, poolName, propName string) (uint64, error) {
	props, err := c.GetProperties(ctx, poolName, propName)
	if err != nil {
		return 0, err
	}

	prop, exists := props[propName]
	if !exists {
		return 0, fmt.Errorf("property %q not found", propName)
	}

	if val, ok := prop.Value.(uint64); ok {
		return val, nil
	}

	return 0, fmt.Errorf("property %q is not a uint64", propName)
}

// Status represents detailed pool status information
type Status struct {
	Pool   Pool
	Config VDevTree
	Errors ErrorStats
	Scan   *ScanStatus
}

// VDevTree represents the virtual device tree structure
type VDevTree struct {
	Type     string
	Path     string
	GUID     uint64
	State    string
	Stats    VDevStats
	Children []VDevTree
}

// VDevStats represents vdev statistics
type VDevStats struct {
	Alloc          uint64
	Space          uint64
	DSpace         uint64
	ReadErrors     uint64
	WriteErrors    uint64
	ChecksumErrors uint64
}

// ErrorStats represents pool error statistics
type ErrorStats struct {
	Read  uint64
	Write uint64
	Cksum uint64
}

// ScanStatus represents scrub/resilver status
type ScanStatus struct {
	Function     ScanFunction // Scrub, Resilver, etc.
	State        ScanState    // Scanning, Finished, Canceled, etc.
	StartTime    time.Time
	EndTime      *time.Time
	Examined     uint64
	ToExamine    uint64
	Processed    uint64
	ToProcess    uint64
	Errors       uint64
	PassExamined uint64
	PassStart    time.Time
	Rate         uint64 // bytes per second (calculated)
}

// ScanFunction represents the type of scan operation
type ScanFunction int

const (
	ScanFunctionNone ScanFunction = iota
	ScanFunctionScrub
	ScanFunctionResilver
)

func (f ScanFunction) String() string {
	switch f {
	case ScanFunctionScrub:
		return "scrub"
	case ScanFunctionResilver:
		return "resilver"
	default:
		return "none"
	}
}

// ScanState represents the state of a scan operation
type ScanState int

const (
	ScanStateNone ScanState = iota
	ScanStateScanning
	ScanStateFinished
	ScanStateCanceled
	ScanStateSuspended
)

func (s ScanState) String() string {
	switch s {
	case ScanStateScanning:
		return "scanning"
	case ScanStateFinished:
		return "finished"
	case ScanStateCanceled:
		return "canceled"
	case ScanStateSuspended:
		return "suspended"
	default:
		return "none"
	}
}

// GetStatus retrieves detailed status for a pool
func (c *Client) GetStatus(ctx context.Context, poolName string) (*Status, error) {
	if c.d == nil {
		return nil, fmt.Errorf("client is closed")
	}

	// Get basic pool information
	pool, err := c.Get(ctx, poolName)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool %s: %w", poolName, err)
	}

	status := &Status{
		Pool: *pool,
	}

	// TODO: Implement vdev tree parsing and scan status
	// This requires complex nvlist parsing that we'll implement next

	return status, nil
}

// StartScrub starts a scrub operation on a pool
func (c *Client) StartScrub(ctx context.Context, poolName string) error {
	if c.d == nil {
		return fmt.Errorf("client is closed")
	}

	// TODO: Implement scrub start via driver
	return fmt.Errorf("StartScrub not implemented yet")
}

// StopScrub stops a running scrub operation on a pool
func (c *Client) StopScrub(ctx context.Context, poolName string) error {
	if c.d == nil {
		return fmt.Errorf("client is closed")
	}

	// TODO: Implement scrub stop via driver
	return fmt.Errorf("StopScrub not implemented yet")
}

// ImportOptions represents options for pool import operations
type ImportOptions struct {
	NewName   string // Optional new name for the pool
	AltRoot   string // Alternative root directory
	Force     bool   // Force import even if pool appears active
	Destroyed bool   // Import destroyed pool
}

// ExportOptions represents options for pool export operations
type ExportOptions struct {
	Force   bool   // Force export even if pool is busy
	Message string // Optional message for export
}

// Import imports a pool that was previously exported or is available for import
func (c *Client) Import(ctx context.Context, poolName string, opts ImportOptions) error {
	if c.d == nil {
		return fmt.Errorf("client is closed")
	}

	return c.d.ImportPool(ctx, poolName, driver.ImportOptions{
		NewName:   opts.NewName,
		AltRoot:   opts.AltRoot,
		Force:     opts.Force,
		Destroyed: opts.Destroyed,
	})
}

// Export exports a pool, making it available for import on other systems
func (c *Client) Export(ctx context.Context, poolName string, opts ExportOptions) error {
	if c.d == nil {
		return fmt.Errorf("client is closed")
	}

	return c.d.ExportPool(ctx, poolName, driver.ExportOptions{
		Force:   opts.Force,
		Message: opts.Message,
	})
}

// Destroy permanently destroys a pool and all its data
func (c *Client) Destroy(ctx context.Context, poolName string) error {
	if c.d == nil {
		return fmt.Errorf("client is closed")
	}

	return c.d.DestroyPool(ctx, poolName)
}

// RuntimeInfo returns information about the underlying ZFS implementation
func (c *Client) RuntimeInfo(ctx context.Context) (impl, zfsVersion, kernelVersion string, err error) {
	if c.d == nil {
		return "", "", "", fmt.Errorf("client is closed")
	}
	return c.d.RuntimeInfo(ctx)
}
