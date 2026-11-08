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

// Discover finds pools that are not imported but can be, like `zpool import`
// without arguments. Results are sorted by name.
func (c *Client) Discover(ctx context.Context, opts DiscoverOptions) ([]ImportablePool, error) {
	if c.d == nil {
		return nil, fmt.Errorf("client is closed")
	}

	found, err := c.d.DiscoverPools(ctx, driver.DiscoverOptions{
		SearchPaths:      opts.SearchPaths,
		IncludeDestroyed: opts.IncludeDestroyed,
		PoolName:         opts.PoolName,
	})
	if err != nil {
		return nil, err
	}

	pools := make([]ImportablePool, 0, len(found))
	for _, p := range found {
		pools = append(pools, ImportablePool{
			Name:        p.Name,
			GUID:        p.GUID,
			State:       p.State,
			Health:      p.Health,
			DevicePaths: p.DevicePaths,
		})
	}
	return pools, nil
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
	PropertySourceNone      PropertySource = "none" // read-only or statistic property
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

// Status represents detailed pool status information, as shown by
// `zpool status`.
type Status struct {
	Pool   Pool
	Config VDevTree   // Root of the data vdev tree (includes log/special vdevs)
	Spares []VDevTree // Hot spares
	Cache  []VDevTree // L2ARC cache devices
	Errors ErrorStats
	Scan   *ScanStatus // nil when the pool was never scrubbed or resilvered

	// Problem is the name of the most important pool status condition, e.g.
	// "OK", "OFFLINE_DEV", "FAILING_DEV", "RESILVERING" (ZPOOL_STATUS_* without
	// the prefix).
	Problem string
	// MsgID is the ZFS message identifier describing Problem (e.g.
	// "ZFS-8000-9P"), see https://openzfs.github.io/openzfs-docs/msg/. Empty
	// when there is no associated message.
	MsgID string
}

// VDevTree represents the virtual device tree structure
type VDevTree struct {
	Name     string // Display name as in `zpool status` (pool name for the root)
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

	// DataErrors is the number of persistent data errors (files with
	// unrecoverable damage) recorded in the pool's error log.
	DataErrors uint64
}

// ScanStatus represents scrub/resilver status
type ScanStatus struct {
	Function     ScanFunction // Scrub, Resilver, etc.
	State        ScanState    // Scanning, Finished, Canceled, etc.
	StartTime    time.Time
	EndTime      *time.Time
	Examined     uint64 // Bytes located by the scanner
	ToExamine    uint64 // Total bytes to scan
	Issued       uint64 // Bytes verified so far ("issued" in zpool status)
	Processed    uint64 // Bytes repaired (scrub) or resilvered
	Errors       uint64
	PassExamined uint64
	PassStart    time.Time
	Rate         uint64 // Bytes issued per second in the current pass; 0 unless scanning
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
	ScanStateSuspended // a paused scrub
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

	ds, err := c.d.GetPoolStatus(ctx, poolName)
	if err != nil {
		return nil, fmt.Errorf("failed to get status of pool %s: %w", poolName, err)
	}

	status := &Status{
		Pool:    *pool,
		Config:  convertVdev(ds.Root),
		Spares:  convertVdevs(ds.Spares),
		Cache:   convertVdevs(ds.Cache),
		Scan:    convertScan(ds.Scan, time.Now()),
		Problem: ds.Problem,
		MsgID:   ds.MsgID,
		Errors: ErrorStats{
			Read:       ds.Root.ReadErrors,
			Write:      ds.Root.WriteErrors,
			Cksum:      ds.Root.ChecksumErrors,
			DataErrors: ds.DataErrors,
		},
	}
	return status, nil
}

func convertVdevs(in []driver.VdevStatus) []VDevTree {
	if len(in) == 0 {
		return nil
	}
	out := make([]VDevTree, 0, len(in))
	for _, v := range in {
		out = append(out, convertVdev(v))
	}
	return out
}

func convertVdev(v driver.VdevStatus) VDevTree {
	return VDevTree{
		Name:  v.Name,
		Type:  v.Type,
		Path:  v.Path,
		GUID:  v.GUID,
		State: v.State,
		Stats: VDevStats{
			Alloc:          v.Alloc,
			Space:          v.Space,
			DSpace:         v.DSpace,
			ReadErrors:     v.ReadErrors,
			WriteErrors:    v.WriteErrors,
			ChecksumErrors: v.ChecksumErrors,
		},
		Children: convertVdevs(v.Children),
	}
}

// convertScan maps driver scan statistics; it returns nil for a pool that was
// never scanned.
func convertScan(s *driver.ScanStats, now time.Time) *ScanStatus {
	if s == nil || (s.Function == driver.PoolScanNone && s.State == driver.PoolScanStateNone) {
		return nil
	}

	scan := &ScanStatus{
		StartTime:    time.Unix(int64(s.StartTime), 0),
		Examined:     s.Examined,
		ToExamine:    s.ToExamine,
		Issued:       s.Issued,
		Processed:    s.Processed,
		Errors:       s.Errors,
		PassExamined: s.PassExamined,
		PassStart:    time.Unix(int64(s.PassStart), 0),
	}

	switch s.Function {
	case driver.PoolScanScrub:
		scan.Function = ScanFunctionScrub
	case driver.PoolScanResilver:
		scan.Function = ScanFunctionResilver
	}

	switch s.State {
	case driver.PoolScanStateScanning:
		scan.State = ScanStateScanning
		if s.PassPausedAt != 0 {
			scan.State = ScanStateSuspended
		}
	case driver.PoolScanStateFinished:
		scan.State = ScanStateFinished
	case driver.PoolScanStateCanceled:
		scan.State = ScanStateCanceled
	}

	if scan.State == ScanStateFinished || scan.State == ScanStateCanceled {
		end := time.Unix(int64(s.EndTime), 0)
		scan.EndTime = &end
	}

	// Same rate calculation as zpool status: issued bytes of this pass over
	// the time the pass was not paused.
	if scan.State == ScanStateScanning {
		elapsed := now.Unix() - int64(s.PassStart) - int64(s.PassSpentPaused)
		if elapsed < 1 {
			elapsed = 1
		}
		scan.Rate = s.PassIssued / uint64(elapsed)
	}

	return scan
}

// StartScrub starts a scrub on a pool, or resumes a paused one (`zpool scrub`).
func (c *Client) StartScrub(ctx context.Context, poolName string) error {
	if c.d == nil {
		return fmt.Errorf("client is closed")
	}

	return c.d.ScrubPool(ctx, poolName, driver.ScrubStart)
}

// PauseScrub pauses a running scrub. StartScrub resumes it.
func (c *Client) PauseScrub(ctx context.Context, poolName string) error {
	if c.d == nil {
		return fmt.Errorf("client is closed")
	}
	return c.d.ScrubPool(ctx, poolName, driver.ScrubPause)
}

// StopScrub cancels a running or paused scrub (`zpool scrub -s`).
func (c *Client) StopScrub(ctx context.Context, poolName string) error {
	if c.d == nil {
		return fmt.Errorf("client is closed")
	}

	return c.d.ScrubPool(ctx, poolName, driver.ScrubStop)
}

// ImportOptions represents options for pool import operations
type ImportOptions struct {
	NewName   string // Optional new name for the pool
	AltRoot   string // Alternative root directory
	Force     bool   // Force import even if pool appears active
	Destroyed bool   // Import destroyed pool

	// SearchPaths lists directories to scan for pool devices (like
	// `zpool import -d`). When empty the default device directories are scanned.
	SearchPaths []string
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
		NewName:     opts.NewName,
		AltRoot:     opts.AltRoot,
		Force:       opts.Force,
		Destroyed:   opts.Destroyed,
		SearchPaths: opts.SearchPaths,
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
