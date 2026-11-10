//go:build freebsd

package driver

import "context"

// PoolInfo represents basic pool information
type PoolInfo struct {
	Name   string
	GUID   uint64
	Health string
	State  string
}

// DatasetType represents the type of ZFS dataset
type DatasetType int

const (
	DatasetFilesystem DatasetType = iota
	DatasetVolume
	DatasetSnapshot
	DatasetBookmark
)

func (t DatasetType) String() string {
	switch t {
	case DatasetFilesystem:
		return "filesystem"
	case DatasetVolume:
		return "volume"
	case DatasetSnapshot:
		return "snapshot"
	case DatasetBookmark:
		return "bookmark"
	default:
		return "unknown"
	}
}

// DatasetInfo represents basic dataset information
type DatasetInfo struct {
	Name string
	Type DatasetType
	GUID uint64
}

// PropSource indicates where a property value comes from
type PropSource int

const (
	PropSourceLocal PropSource = iota
	PropSourceInherited
	PropSourceDefault
	PropSourceTemporary
	PropSourceReceived
	PropSourceNone // read-only or statistic property, no settable source
)

func (s PropSource) String() string {
	switch s {
	case PropSourceLocal:
		return "local"
	case PropSourceInherited:
		return "inherited"
	case PropSourceDefault:
		return "default"
	case PropSourceTemporary:
		return "temporary"
	case PropSourceReceived:
		return "received"
	case PropSourceNone:
		return "none"
	default:
		return "unknown"
	}
}

// PropertyInfo represents a property with its source and type information
type PropertyInfo struct {
	Name     string
	Value    any
	Source   PropSource
	Received bool

	// Numeric is the exact value of a numeric property (used, available,
	// refreservation, volsize, ...) in its base unit (bytes for sizes).
	// IsNumeric is false for string and index properties such as
	// mountpoint or compression.
	Numeric   uint64
	IsNumeric bool
}

// UserPropertyInfo is the value and source of a ZFS user property
// (a property whose name contains a colon, e.g. "org.example:key")
type UserPropertyInfo struct {
	Value  string
	Source PropSource
}

// ChildUserProps holds the user properties of one descendant dataset
type ChildUserProps struct {
	Name  string
	Type  DatasetType
	Props map[string]UserPropertyInfo
}

// ImportOptions represents options for pool import
type ImportOptions struct {
	NewName   string // Optional new name for the pool
	AltRoot   string // Alternative root directory
	Force     bool   // Force import even if pool appears active
	Destroyed bool   // Import destroyed pool

	// SearchPaths lists directories to scan for pool devices. When empty the
	// default device directories (e.g. /dev) are scanned.
	SearchPaths []string
}

// DiscoverOptions controls the search for importable pools
type DiscoverOptions struct {
	SearchPaths      []string // Directories to scan; default device directories when empty
	IncludeDestroyed bool     // Also report destroyed pools
	PoolName         string   // Only report pools with this name when set
}

// ImportablePool describes a pool found on disk that is not currently imported
type ImportablePool struct {
	Name        string
	GUID        uint64
	State       string   // e.g. "EXPORTED", "DESTROYED"
	Health      string   // e.g. "ONLINE", "DEGRADED"
	DevicePaths []string // Leaf devices (data, cache and spares)
}

// VdevStatus describes one vdev and its subtree, as in `zpool status`
type VdevStatus struct {
	Name           string // Display name (zpool_vdev_name)
	Type           string // e.g. "root", "mirror", "raidz", "disk", "file"
	Path           string // Device path for leaf vdevs
	GUID           uint64
	State          string // e.g. "ONLINE", "DEGRADED", "OFFLINE", "AVAIL"
	Alloc          uint64
	Space          uint64
	DSpace         uint64
	ReadErrors     uint64
	WriteErrors    uint64
	ChecksumErrors uint64
	Children       []VdevStatus
}

// ScanStats mirrors pool_scan_stat_t. Times are Unix seconds.
type ScanStats struct {
	Function        uint64 // PoolScan* constant
	State           uint64 // PoolScanState* constant
	StartTime       uint64
	EndTime         uint64
	ToExamine       uint64
	Examined        uint64
	Skipped         uint64
	Processed       uint64
	Errors          uint64
	PassExamined    uint64
	PassStart       uint64
	PassPausedAt    uint64 // non-zero while a scrub is paused
	PassSpentPaused uint64 // seconds the current pass spent paused
	PassIssued      uint64
	Issued          uint64
}

// PoolStatus is the detailed status of an imported pool
type PoolStatus struct {
	Root       VdevStatus
	Spares     []VdevStatus
	Cache      []VdevStatus
	DataErrors uint64
	Scan       *ScanStats // nil when the pool has no scan statistics
	Problem    string     // zpool_status_t name without the ZPOOL_STATUS_ prefix
	MsgID      string     // ZFS-8000-* message id, empty if none
}

// ScrubCommand selects the scrub action for ScrubPool
type ScrubCommand int

const (
	ScrubStart ScrubCommand = iota // start, or resume a paused scrub
	ScrubPause
	ScrubStop
)

// ExportOptions represents options for pool export
type ExportOptions struct {
	Force   bool   // Force export even if pool is busy
	Message string // Optional message for export
}

// CreateOptions represents options for pool creation
type CreateOptions struct {
	Properties   map[string]string // Pool properties
	FsProperties map[string]string // Root filesystem properties
	AltRoot      string            // Alternative root directory
	Force        bool              // Force creation
}

// VdevSpec represents a vdev specification for pool operations
type VdevSpec struct {
	Type    string   // Type of vdev (mirror, raidz, etc.) - use VdevType* constants
	Devices []string // Device paths
}

// VdevOnlineFlags represents flags for vdev online operations
type VdevOnlineFlags int

const (
	VdevOnlineCheckRemove VdevOnlineFlags = 1 << iota
	VdevOnlineUnspare
	VdevOnlineForceFault
	VdevOnlineExpand
)

// CloneInfo represents information about a ZFS clone
type CloneInfo struct {
	Name       string   // Clone dataset name
	Origin     string   // Origin snapshot name
	IsClone    bool     // Whether this dataset is a clone
	CloneCount int      // Number of clones (if this is a snapshot)
	Dependents []string // List of datasets that depend on this clone
}

// Driver is the abstraction layer for ZFS operations
// Implementations can use libzfs (CGO) or direct ioctls
type Driver interface {
	// Close releases any resources held by the driver
	Close() error

	// RuntimeInfo returns implementation details and version information
	RuntimeInfo(ctx context.Context) (impl string, zfsVer string, kernel string, err error)

	// Pool operations
	ListPools(ctx context.Context) ([]PoolInfo, error)
	GetPoolProps(ctx context.Context, poolName string, propNames []string) (map[string]PropertyInfo, error)
	DiscoverPools(ctx context.Context, opts DiscoverOptions) ([]ImportablePool, error)
	GetPoolStatus(ctx context.Context, poolName string) (*PoolStatus, error)
	ScrubPool(ctx context.Context, poolName string, cmd ScrubCommand) error
	ImportPool(ctx context.Context, poolName string, opts ImportOptions) error
	ExportPool(ctx context.Context, poolName string, opts ExportOptions) error
	CreatePool(ctx context.Context, poolName string, vdevs []string, opts CreateOptions) error
	DestroyPool(ctx context.Context, poolName string) error

	// Dataset operations
	ListDatasets(ctx context.Context, recursive bool) ([]DatasetInfo, error)
	ListDatasetsInPool(ctx context.Context, poolName string, recursive bool) ([]DatasetInfo, error)
	ListDatasetsByType(ctx context.Context, dsType *DatasetType, recursive bool) ([]DatasetInfo, error)
	GetDatasetProps(ctx context.Context, datasetName string, propNames []string) (map[string]PropertyInfo, error)
	GetUserProperties(ctx context.Context, datasetName string) (map[string]UserPropertyInfo, error)
	// ListDescendantUserProps returns the user properties of every filesystem
	// and volume below parent (not parent itself, no snapshots), read in one
	// pass. It fails if parent does not exist; descendants destroyed during
	// the walk are skipped.
	ListDescendantUserProps(ctx context.Context, parent string) ([]ChildUserProps, error)
	SetDatasetProp(ctx context.Context, datasetName, propName, propValue string) error
	CreateDataset(ctx context.Context, datasetName string, dsType DatasetType, props map[string]string) error
	DestroyDataset(ctx context.Context, datasetName string, recursive bool) error

	// Snapshot operations
	CreateSnapshot(ctx context.Context, snapshotName string, recursive bool, props map[string]string) error
	DestroySnapshot(ctx context.Context, snapshotName string) error
	RollbackToSnapshot(ctx context.Context, datasetName, snapshotName string, force bool) error

	// Clone operations
	CreateClone(ctx context.Context, snapshotName, cloneName string, props map[string]string) error
	PromoteClone(ctx context.Context, cloneName string) error
	GetCloneInfo(ctx context.Context, datasetName string) (*CloneInfo, error)
	ListClones(ctx context.Context, snapshotName string) ([]string, error)
	DestroyClone(ctx context.Context, cloneName string, force bool) error

	// Vdev management operations
	AddVdev(ctx context.Context, poolName string, vdevSpec VdevSpec) error
	AttachVdev(ctx context.Context, poolName, oldDevice, newDevice string, replacing bool) error
	DetachVdev(ctx context.Context, poolName, device string) error
	ReplaceVdev(ctx context.Context, poolName, oldDevice, newDevice string) error
	RemoveVdev(ctx context.Context, poolName, device string) error
	OnlineVdev(ctx context.Context, poolName, device string, flags VdevOnlineFlags) error
	OfflineVdev(ctx context.Context, poolName, device string, temporary bool) error
	ClearVdev(ctx context.Context, poolName, device string) error

	// Feature and capability detection
	SupportsFeature(ctx context.Context, feature string) (bool, error)
	GetAvailableFeatures(ctx context.Context) ([]string, error)
	GetSupportedCompressionAlgorithms(ctx context.Context) ([]string, error)
	GetZFSVersion(ctx context.Context) (string, error)
}

// Common property names as constants for type safety
const (
	PropNameMountpoint    = "mountpoint"
	PropNameCompression   = "compression"
	PropNameUsed          = "used"
	PropNameAvail         = "avail"
	PropNameRefer         = "refer"
	PropNameCompressratio = "compressratio"
	PropNameQuota         = "quota"
	PropNameReservation   = "reservation"
	PropNameRecordsize    = "recordsize"
	PropNameAtime         = "atime"
	PropNameDevices       = "devices"
	PropNameExec          = "exec"
	PropNameReadonly      = "readonly"
	PropNameSetuid        = "setuid"
	PropNameZoned         = "zoned"
	PropNameSnapdir       = "snapdir"
	PropNameAclmode       = "aclmode"
	PropNameCanmount      = "canmount"
	PropNameXattr         = "xattr"
	PropNameCopies        = "copies"
	PropNameVersion       = "version"
	PropNameUtf8only      = "utf8only"
	PropNameNormalize     = "normalize"
	PropNameCase          = "casesensitivity"
	PropNameVscan         = "vscan"
	PropNameNbmand        = "nbmand"
	PropNameSharenfs      = "sharenfs"
	PropNameSharesmb      = "sharesmb"
	PropNameRefquota      = "refquota"
	PropNameRefreserv     = "refreservation"
	PropNameGuid          = "guid"
	PropNamePrimcache     = "primarycache"
	PropNameSeccache      = "secondarycache"
	PropNameUsedsnap      = "usedbysnapshots"
	PropNameUsedds        = "usedbydataset"
	PropNameUsedchild     = "usedbychildren"
	PropNameUsedrefreserv = "usedbyrefreservation"
	PropNameDefer         = "defer_destroy"
	PropNameUserrefs      = "userrefs"
	PropNameLogbias       = "logbias"
	PropNameUnique        = "unique"
	PropNameWritten       = "written"
	PropNameClones        = "clones"
	PropNameLogicalused   = "logicalused"
	PropNameLogicalavail  = "logicalavail"
	PropNameSync          = "sync"
	PropNameDnodesize     = "dnodesize"
	PropNameRefcomprat    = "refcompressratio"
	PropNameEncryption    = "encryption"
	PropNameKeylocation   = "keylocation"
	PropNameKeyformat     = "keyformat"
	PropNamePbkdf2iters   = "pbkdf2iters"
	PropNameEncroot       = "encryptionroot"
	PropNameKeystatus     = "keystatus"

	// Pool-specific property names
	PropNameSize      = "size"
	PropNameCapacity  = "capacity"
	PropNameFree      = "free"
	PropNameAllocated = "allocated"
	PropNameHealth    = "health"
)
