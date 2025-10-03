//go:build freebsd

package driver

// Pool state constants
const (
	PoolStateActive            = 0
	PoolStateExported          = 1
	PoolStateDestroyed         = 2
	PoolStateSpare             = 3
	PoolStateL2Cache           = 4
	PoolStateUninitialized     = 5
	PoolStateUnavail           = 6
	PoolStatePotentiallyActive = 7
)

// Pool health status constants
const (
	PoolStatusCorruptCache   = 0
	PoolStatusMissingDevR    = 1
	PoolStatusMissingDevNr   = 2
	PoolStatusCorruptLabelR  = 3
	PoolStatusCorruptLabelNr = 4
)

// Property buffer size for C operations
const (
	PropertyBufferSize = 1024
	PropertyNameMaxLen = 256
)

// Common ZFS property names
const (
	// Pool properties
	PropPoolSize      = "size"
	PropPoolCapacity  = "capacity"
	PropPoolFree      = "free"
	PropPoolAllocated = "allocated"
	PropPoolHealth    = "health"
	PropPoolGuid      = "guid"
	PropPoolVersion   = "version"

	// Dataset properties
	PropDatasetType          = "type"
	PropDatasetCreation      = "creation"
	PropDatasetUsed          = "used"
	PropDatasetAvail         = "avail"
	PropDatasetReferenced    = "referenced"
	PropDatasetCompressratio = "compressratio"
	PropDatasetMounted       = "mounted"
	PropDatasetQuota         = "quota"
	PropDatasetReservation   = "reservation"
	PropDatasetRecordsize    = "recordsize"
	PropDatasetMountpoint    = "mountpoint"
	PropDatasetCompression   = "compression"
	PropDatasetAtime         = "atime"
	PropDatasetDevices       = "devices"
	PropDatasetExec          = "exec"
	PropDatasetReadonly      = "readonly"
	PropDatasetSetuid        = "setuid"
	PropDatasetZoned         = "zoned"
	PropDatasetSnapdir       = "snapdir"
	PropDatasetAclmode       = "aclmode"
	PropDatasetCanmount      = "canmount"
	PropDatasetXattr         = "xattr"
	PropDatasetCopies        = "copies"
	PropDatasetVersion       = "version"
	PropDatasetUtf8only      = "utf8only"
	PropDatasetNormalize     = "normalize"
	PropDatasetCase          = "casesensitivity"
	PropDatasetVscan         = "vscan"
	PropDatasetNbmand        = "nbmand"
	PropDatasetSharenfs      = "sharenfs"
	PropDatasetSharesmb      = "sharesmb"
	PropDatasetRefquota      = "refquota"
	PropDatasetRefreserv     = "refreservation"
	PropDatasetPrimcache     = "primarycache"
	PropDatasetSeccache      = "secondarycache"
	PropDatasetUsedsnap      = "usedbysnapshots"
	PropDatasetUsedds        = "usedbydataset"
	PropDatasetUsedchild     = "usedbychildren"
	PropDatasetUsedrefreserv = "usedbyrefreservation"
	PropDatasetLogbias       = "logbias"
	PropDatasetSync          = "sync"
	PropDatasetDedup         = "dedup"
	PropDatasetMlslabel      = "mlslabel"
	PropDatasetRelAtime      = "relatime"
	PropDatasetRedundant     = "redundant_metadata"
	PropDatasetOverlay       = "overlay"
	PropDatasetEncryption    = "encryption"
	PropDatasetKeylocation   = "keylocation"
	PropDatasetKeyformat     = "keyformat"
	PropDatasetPbkdf2iters   = "pbkdf2iters"
	PropDatasetEncroot       = "encryptionroot"
	PropDatasetKeystatus     = "keystatus"
)

// Vdev types
const (
	VdevTypeRoot   = "root"
	VdevTypeDisk   = "disk"
	VdevTypeMirror = "mirror"
	VdevTypeRaidz  = "raidz"
	VdevTypeRaidz2 = "raidz2"
	VdevTypeRaidz3 = "raidz3"
	VdevTypeSpare  = "spare"
	VdevTypeLog    = "log"
	VdevTypeCache  = "cache"
)

// Pool scan function types
const (
	PoolScanNone     = 0
	PoolScanScrub    = 1
	PoolScanResilver = 2
)

// Pool scan states
const (
	PoolScanStateNone     = 0
	PoolScanStateScanning = 1
	PoolScanStateFinished = 2
	PoolScanStateCanceled = 3
)
