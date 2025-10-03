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
}

// ImportOptions represents options for pool import
type ImportOptions struct {
	NewName   string // Optional new name for the pool
	AltRoot   string // Alternative root directory
	Force     bool   // Force import even if pool appears active
	Destroyed bool   // Import destroyed pool
}

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
	ImportPool(ctx context.Context, poolName string, opts ImportOptions) error
	ExportPool(ctx context.Context, poolName string, opts ExportOptions) error
	CreatePool(ctx context.Context, poolName string, vdevs []string, opts CreateOptions) error
	DestroyPool(ctx context.Context, poolName string) error

	// Dataset operations
	ListDatasets(ctx context.Context, recursive bool) ([]DatasetInfo, error)
	ListDatasetsInPool(ctx context.Context, poolName string, recursive bool) ([]DatasetInfo, error)
	ListDatasetsByType(ctx context.Context, dsType *DatasetType, recursive bool) ([]DatasetInfo, error)
	GetDatasetProps(ctx context.Context, datasetName string, propNames []string) (map[string]PropertyInfo, error)
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
