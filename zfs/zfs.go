//go:build freebsd

package zfs

import (
	"context"
	"fmt"
	"strings"

	"github.com/zombocoder/go-freebsd-libzfs/internal/driver"
)

// Client provides access to ZFS dataset operations
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

// New creates a new ZFS dataset client
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

// Dataset represents a ZFS dataset (filesystem, volume, snapshot, or bookmark)
type Dataset struct {
	Name string
	Type DatasetType
	GUID uint64
}

// DatasetType represents the type of ZFS dataset
type DatasetType string

const (
	TypeFilesystem DatasetType = "filesystem"
	TypeVolume     DatasetType = "volume"
	TypeSnapshot   DatasetType = "snapshot"
	TypeBookmark   DatasetType = "bookmark"
)

// IsSnapshot returns true if this dataset is a snapshot
func (d Dataset) IsSnapshot() bool {
	return d.Type == TypeSnapshot
}

// IsFilesystem returns true if this dataset is a filesystem
func (d Dataset) IsFilesystem() bool {
	return d.Type == TypeFilesystem
}

// IsVolume returns true if this dataset is a volume
func (d Dataset) IsVolume() bool {
	return d.Type == TypeVolume
}

// IsBookmark returns true if this dataset is a bookmark
func (d Dataset) IsBookmark() bool {
	return d.Type == TypeBookmark
}

// Pool returns the pool name this dataset belongs to
func (d Dataset) Pool() string {
	parts := strings.SplitN(d.Name, "/", 2)
	return parts[0]
}

// List returns all datasets, optionally recursively
func (c *Client) List(ctx context.Context, recursive bool) ([]Dataset, error) {
	if c.d == nil {
		return nil, fmt.Errorf("client is closed")
	}

	// Use comprehensive listing to get all dataset types including snapshots
	datasetInfos, err := c.d.ListDatasetsByType(ctx, nil, recursive)
	if err != nil {
		return nil, fmt.Errorf("failed to list datasets: %w", err)
	}

	datasets := make([]Dataset, 0, len(datasetInfos))
	for _, info := range datasetInfos {
		dataset := Dataset{
			Name: info.Name,
			Type: mapDriverTypeToPublic(info.Type),
			GUID: info.GUID,
		}
		datasets = append(datasets, dataset)
	}

	return datasets, nil
}

// ListInPool returns all datasets in a specific pool
func (c *Client) ListInPool(ctx context.Context, poolName string, recursive bool) ([]Dataset, error) {
	if c.d == nil {
		return nil, fmt.Errorf("client is closed")
	}

	datasetInfos, err := c.d.ListDatasetsInPool(ctx, poolName, recursive)
	if err != nil {
		return nil, fmt.Errorf("failed to list datasets in pool %q: %w", poolName, err)
	}

	datasets := make([]Dataset, 0, len(datasetInfos))
	for _, info := range datasetInfos {
		dataset := Dataset{
			Name: info.Name,
			Type: mapDriverTypeToPublic(info.Type),
			GUID: info.GUID,
		}
		datasets = append(datasets, dataset)
	}

	return datasets, nil
}

// Get retrieves a specific dataset by name
func (c *Client) Get(ctx context.Context, name string) (*Dataset, error) {
	datasets, err := c.List(ctx, true)
	if err != nil {
		return nil, err
	}

	for _, dataset := range datasets {
		if dataset.Name == name {
			return &dataset, nil
		}
	}

	return nil, fmt.Errorf("dataset %q not found", name)
}

// Property represents a dataset property with its value and metadata
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

// Common dataset property names
const (
	PropertyNameType          = "type"
	PropertyNameCreation      = "creation"
	PropertyNameUsed          = "used"
	PropertyNameAvail         = "avail"
	PropertyNameReferenced    = "referenced"
	PropertyNameCompressratio = "compressratio"
	PropertyNameMounted       = "mounted"
	PropertyNameQuota         = "quota"
	PropertyNameReservation   = "reservation"
	PropertyNameRecordsize    = "recordsize"
	PropertyNameMountpoint    = "mountpoint"
	PropertyNameSharenfs      = "sharenfs"
	PropertyNameChecksum      = "checksum"
	PropertyNameCompression   = "compression"
	PropertyNameAtime         = "atime"
	PropertyNameDevices       = "devices"
	PropertyNameExec          = "exec"
	PropertyNameSetuid        = "setuid"
	PropertyNameReadonly      = "readonly"
	PropertyNameZoned         = "zoned"
	PropertyNameSnapdir       = "snapdir"
	PropertyNameAclmode       = "aclmode"
	PropertyNameCanmount      = "canmount"
	PropertyNameXattr         = "xattr"
	PropertyNameCopies        = "copies"
	PropertyNameVersion       = "version"
	PropertyNameUtf8only      = "utf8only"
	PropertyNameNormalize     = "normalize"
	PropertyNameCase          = "casesensitivity"
	PropertyNameVscan         = "vscan"
	PropertyNameNbmand        = "nbmand"
	PropertyNameSharesmb      = "sharesmb"
	PropertyNameRefquota      = "refquota"
	PropertyNameRefreserv     = "refreservation"
	PropertyNameGuid          = "guid"
	PropertyNamePrimcache     = "primarycache"
	PropertyNameSeccache      = "secondarycache"
	PropertyNameUsedsnap      = "usedbysnapshots"
	PropertyNameUsedds        = "usedbydataset"
	PropertyNameUsedchild     = "usedbychildren"
	PropertyNameUsedrefreserv = "usedbyrefreservation"
	PropertyNameLogbias       = "logbias"
	PropertyNameSync          = "sync"
	PropertyNameDedup         = "dedup"
	PropertyNameMlslabel      = "mlslabel"
	PropertyNameRelAtime      = "relatime"
	PropertyNameRedundant     = "redundant_metadata"
	PropertyNameOverlay       = "overlay"
	PropertyNameEncryption    = "encryption"
	PropertyNameKeylocation   = "keylocation"
	PropertyNameKeyformat     = "keyformat"
	PropertyNamePbkdf2iters   = "pbkdf2iters"
	PropertyNameEncroot       = "encryptionroot"
	PropertyNameKeystatus     = "keystatus"
)

// GetProperties retrieves multiple properties for a dataset
func (c *Client) GetProperties(ctx context.Context, datasetName string, propNames ...string) (map[string]Property[any], error) {
	if c.d == nil {
		return nil, fmt.Errorf("client is closed")
	}

	propInfos, err := c.d.GetDatasetProps(ctx, datasetName, propNames)
	if err != nil {
		return nil, fmt.Errorf("failed to get dataset properties: %w", err)
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
func (c *Client) GetStringProperty(ctx context.Context, datasetName, propName string) (string, error) {
	props, err := c.GetProperties(ctx, datasetName, propName)
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
func (c *Client) GetUint64Property(ctx context.Context, datasetName, propName string) (uint64, error) {
	props, err := c.GetProperties(ctx, datasetName, propName)
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

// GetBoolProperty retrieves a boolean property value
func (c *Client) GetBoolProperty(ctx context.Context, datasetName, propName string) (bool, error) {
	str, err := c.GetStringProperty(ctx, datasetName, propName)
	if err != nil {
		return false, err
	}

	switch strings.ToLower(str) {
	case "on", "true", "yes", "1":
		return true, nil
	case "off", "false", "no", "0":
		return false, nil
	default:
		return false, fmt.Errorf("property %q value %q is not a boolean", propName, str)
	}
}

// SetProperty sets a property on a dataset
func (c *Client) SetProperty(ctx context.Context, datasetName, propName, propValue string) error {
	if c.d == nil {
		return fmt.Errorf("client is closed")
	}

	return c.d.SetDatasetProp(ctx, datasetName, propName, propValue)
}

// SetStringProperty sets a string property on a dataset
func (c *Client) SetStringProperty(ctx context.Context, datasetName, propName, propValue string) error {
	return c.SetProperty(ctx, datasetName, propName, propValue)
}

// SetBoolProperty sets a boolean property on a dataset
func (c *Client) SetBoolProperty(ctx context.Context, datasetName, propName string, propValue bool) error {
	value := "off"
	if propValue {
		value = "on"
	}
	return c.SetProperty(ctx, datasetName, propName, value)
}

// Snapshot represents a ZFS snapshot
type Snapshot struct {
	Dataset
	Parent string // Parent dataset name
}

// ListSnapshots returns all snapshots, optionally filtering by parent dataset
func (c *Client) ListSnapshots(ctx context.Context, parent string) ([]Snapshot, error) {
	if c.d == nil {
		return nil, fmt.Errorf("client is closed")
	}

	// Get only snapshots using the new filtered API
	snapshotType := driver.DatasetSnapshot
	datasetInfos, err := c.d.ListDatasetsByType(ctx, &snapshotType, true)
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	var snapshots []Snapshot
	for _, info := range datasetInfos {
		// Extract parent from snapshot name (format: parent@snapname)
		atIndex := strings.LastIndex(info.Name, "@")
		if atIndex == -1 {
			continue // Invalid snapshot name
		}

		snapParent := info.Name[:atIndex]

		// Filter by parent if specified
		if parent != "" {
			// Support both exact match and hierarchical match
			if snapParent != parent && !strings.HasPrefix(snapParent, parent+"/") {
				continue
			}
		}

		dataset := Dataset{
			Name: info.Name,
			Type: mapDriverTypeToPublic(info.Type),
			GUID: info.GUID,
		}

		snapshot := Snapshot{
			Dataset: dataset,
			Parent:  snapParent,
		}
		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}

// CreateOptions represents options for dataset creation
type CreateOptions struct {
	Properties map[string]string // Dataset properties to set
}

// Create creates a new ZFS dataset
func (c *Client) Create(ctx context.Context, datasetName string, dsType DatasetType, opts CreateOptions) error {
	if c.d == nil {
		return fmt.Errorf("client is closed")
	}

	// Map public dataset type to driver type
	var driverType driver.DatasetType
	switch dsType {
	case TypeFilesystem:
		driverType = driver.DatasetFilesystem
	case TypeVolume:
		driverType = driver.DatasetVolume
	default:
		return fmt.Errorf("unsupported dataset type for creation: %s", dsType)
	}

	return c.d.CreateDataset(ctx, datasetName, driverType, opts.Properties)
}

// CreateFilesystem creates a new ZFS filesystem
func (c *Client) CreateFilesystem(ctx context.Context, datasetName string, properties map[string]string) error {
	return c.Create(ctx, datasetName, TypeFilesystem, CreateOptions{Properties: properties})
}

// CreateVolume creates a new ZFS volume (zvol)
func (c *Client) CreateVolume(ctx context.Context, datasetName string, size string, properties map[string]string) error {
	if properties == nil {
		properties = make(map[string]string)
	}
	properties["volsize"] = size

	return c.Create(ctx, datasetName, TypeVolume, CreateOptions{Properties: properties})
}

// Destroy destroys a ZFS dataset
func (c *Client) Destroy(ctx context.Context, datasetName string, recursive bool) error {
	if c.d == nil {
		return fmt.Errorf("client is closed")
	}

	return c.d.DestroyDataset(ctx, datasetName, recursive)
}

// CreateSnapshot creates a snapshot of a dataset
func (c *Client) CreateSnapshot(ctx context.Context, snapshotName string, recursive bool, properties map[string]string) error {
	if c.d == nil {
		return fmt.Errorf("client is closed")
	}

	return c.d.CreateSnapshot(ctx, snapshotName, recursive, properties)
}

// DestroySnapshot destroys a snapshot
func (c *Client) DestroySnapshot(ctx context.Context, snapshotName string) error {
	if c.d == nil {
		return fmt.Errorf("client is closed")
	}

	return c.d.DestroySnapshot(ctx, snapshotName)
}

// RollbackToSnapshot rolls back a dataset to a specific snapshot
func (c *Client) RollbackToSnapshot(ctx context.Context, datasetName, snapshotName string, force bool) error {
	if c.d == nil {
		return fmt.Errorf("client is closed")
	}

	return c.d.RollbackToSnapshot(ctx, datasetName, snapshotName, force)
}

// RuntimeInfo returns information about the underlying ZFS implementation
func (c *Client) RuntimeInfo(ctx context.Context) (impl, zfsVersion, kernelVersion string, err error) {
	if c.d == nil {
		return "", "", "", fmt.Errorf("client is closed")
	}
	return c.d.RuntimeInfo(ctx)
}

// Clone represents a ZFS clone with its relationship information
type Clone struct {
	Dataset
	Origin     string   // Origin snapshot name
	IsClone    bool     // Whether this dataset is a clone
	CloneCount int      // Number of clones (if this is a snapshot)
	Dependents []string // List of datasets that depend on this clone
}

// CreateClone creates a clone from a snapshot
func (c *Client) CreateClone(ctx context.Context, snapshotName, cloneName string, properties map[string]string) error {
	if c.d == nil {
		return fmt.Errorf("client is closed")
	}

	return c.d.CreateClone(ctx, snapshotName, cloneName, properties)
}

// PromoteClone promotes a clone to become independent from its origin
func (c *Client) PromoteClone(ctx context.Context, cloneName string) error {
	if c.d == nil {
		return fmt.Errorf("client is closed")
	}

	return c.d.PromoteClone(ctx, cloneName)
}

// GetCloneInfo retrieves detailed information about a clone or snapshot
func (c *Client) GetCloneInfo(ctx context.Context, datasetName string) (*Clone, error) {
	if c.d == nil {
		return nil, fmt.Errorf("client is closed")
	}

	info, err := c.d.GetCloneInfo(ctx, datasetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get clone info: %w", err)
	}

	// Get basic dataset info
	dataset, err := c.Get(ctx, datasetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get dataset info: %w", err)
	}

	clone := &Clone{
		Dataset:    *dataset,
		Origin:     info.Origin,
		IsClone:    info.IsClone,
		CloneCount: info.CloneCount,
		Dependents: info.Dependents,
	}

	return clone, nil
}

// ListClones returns all clones of a snapshot
func (c *Client) ListClones(ctx context.Context, snapshotName string) ([]string, error) {
	if c.d == nil {
		return nil, fmt.Errorf("client is closed")
	}

	return c.d.ListClones(ctx, snapshotName)
}

// DestroyClone destroys a clone dataset
func (c *Client) DestroyClone(ctx context.Context, cloneName string, force bool) error {
	if c.d == nil {
		return fmt.Errorf("client is closed")
	}

	return c.d.DestroyClone(ctx, cloneName, force)
}

// IsClone checks if a dataset is a clone
func (c *Client) IsClone(ctx context.Context, datasetName string) (bool, error) {
	info, err := c.GetCloneInfo(ctx, datasetName)
	if err != nil {
		return false, err
	}
	return info.IsClone, nil
}

// GetCloneOrigin returns the origin snapshot of a clone
func (c *Client) GetCloneOrigin(ctx context.Context, cloneName string) (string, error) {
	info, err := c.GetCloneInfo(ctx, cloneName)
	if err != nil {
		return "", err
	}
	if !info.IsClone {
		return "", fmt.Errorf("dataset %s is not a clone", cloneName)
	}
	return info.Origin, nil
}

// Helper function to map driver types to public types
func mapDriverTypeToPublic(driverType driver.DatasetType) DatasetType {
	switch driverType {
	case driver.DatasetFilesystem:
		return TypeFilesystem
	case driver.DatasetVolume:
		return TypeVolume
	case driver.DatasetSnapshot:
		return TypeSnapshot
	case driver.DatasetBookmark:
		return TypeBookmark
	default:
		return TypeFilesystem // default fallback
	}
}
