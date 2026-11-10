//go:build freebsd && !no_libzfs

package driver

/*
#cgo CFLAGS: -I/usr/src/sys/contrib/openzfs/lib/libspl/include -I/usr/src/sys/contrib/openzfs/lib/libspl/include/os/freebsd -I/usr/src/sys/contrib/openzfs/include
#cgo LDFLAGS: -lzfs -lnvpair
#include <libzfs.h>
#include <sys/nvpair.h>
#include <stdlib.h>

// Forward declarations for our C helper functions
extern libzfs_handle_t* go_libzfs_init(void);
extern void go_libzfs_fini(libzfs_handle_t* hdl);
extern int go_libzfs_errno(libzfs_handle_t* hdl);
extern const char* go_libzfs_error_description(libzfs_handle_t* hdl);

// Iteration functions
extern int go_iter_pools(libzfs_handle_t* hdl, int (*func)(zpool_handle_t *, void *), void* data);
extern int go_iter_datasets(libzfs_handle_t* hdl, int (*func)(zfs_handle_t *, void *), void* data);
extern int go_iter_all_datasets(libzfs_handle_t* hdl, int (*func)(zfs_handle_t *, void *), void* data);

// Callback functions for iteration
extern int go_pool_iter_callback(zpool_handle_t *, void *);
extern int go_dataset_iter_callback(zfs_handle_t *, void *);

// Signal masking
extern void go_sigs_block(void);
extern void go_sigs_restore(void);

// Iteration handle accounting
extern void go_iter_given(void);
extern void go_iter_close_zfs(zfs_handle_t* zhp);
extern void go_iter_close_zpool(zpool_handle_t* zhp);
extern long go_iter_outstanding(void);

// Pool state and health
extern int go_zpool_get_state(zpool_handle_t* zhp);
extern int go_zpool_get_status(zpool_handle_t* zhp, char** msgid);
extern int go_zpool_health(zpool_handle_t *zhp, char *buf, size_t len);

// Property constants
extern int go_get_zpool_prop_size();
extern int go_get_zpool_prop_capacity();
extern int go_get_zpool_prop_health();
extern int go_get_zpool_prop_guid();
extern int go_get_zpool_prop_version();
extern int go_get_zpool_prop_free();
extern int go_get_zpool_prop_allocated();

extern int go_get_zfs_prop_type();
extern int go_get_zfs_prop_creation();
extern int go_get_zfs_prop_used();
extern int go_get_zfs_prop_available();
extern int go_get_zfs_prop_referenced();
extern int go_get_zfs_prop_compressratio();
extern int go_get_zfs_prop_mounted();
extern int go_get_zfs_prop_quota();
extern int go_get_zfs_prop_reservation();
extern int go_get_zfs_prop_recordsize();
extern int go_get_zfs_prop_mountpoint();
extern int go_get_zfs_prop_compression();

// Property retrieval
extern int go_get_zpool_property(zpool_handle_t* zhp, int prop, char* buf, size_t len, int* src_out);
extern int go_get_zfs_property(zfs_handle_t* zhp, int prop, char* buf, size_t len, int* src_out);

// Pool configuration and status
extern void* go_zpool_get_config(zpool_handle_t* zhp, void** oldconfig);
extern int go_zpool_scan(zpool_handle_t* zhp, int func, int cmd);

// Nvlist helpers
extern int go_nvlist_lookup_nvlist(void* nvl, char* name, void** val);
extern int go_nvlist_lookup_nvlist_array(void* nvl, char* name, void*** val, unsigned int* nelem);
extern int go_nvlist_lookup_string(void* nvl, char* name, char** val);
extern int go_nvlist_lookup_uint64(void* nvl, char* name, uint64_t* val);


// Pool operations
extern int go_zpool_export(zpool_handle_t* zhp, int force, char* message);
extern int go_zpool_export_force(zpool_handle_t* zhp);
extern nvlist_t *go_zpool_search(libzfs_handle_t *hdl, const char *poolname,
    char **paths, int npaths, char *errbuf, size_t errlen, int *err);
extern int go_import_config_summary(nvlist_t *config, const char **name, uint64_t *guid,
    uint64_t *state, const char **state_name, const char **health);
extern int go_import_config_device_paths(nvlist_t *config, const char **out, int max);
extern int go_zpool_import(libzfs_handle_t *hdl, const char *poolname,
    char **paths, int npaths, const char *newname, const char *altroot,
    int flags, int destroyed, char *errbuf, size_t errlen);
extern int go_zpool_create(libzfs_handle_t* hdl, char* poolname, void* nvroot, void* props, void* fsprops);
extern int go_zpool_destroy(zpool_handle_t* zhp, char* message);

// Dataset operations
extern int go_zfs_create(libzfs_handle_t* hdl, char* path, int type, void* props);
extern int go_zfs_destroy(zfs_handle_t* zhp, int defer_destroy);
extern int go_zfs_destroy_recursive(zfs_handle_t* zhp, char* snapname);

// Dataset type constants
extern int go_get_zfs_type_filesystem();
extern int go_get_zfs_type_volume();
extern int go_get_zfs_type_snapshot();

// Property setting
extern int go_zfs_prop_set(zfs_handle_t* zhp, char* propname, char* propval);

// Snapshot operations
extern int go_zfs_snapshot(libzfs_handle_t* hdl, char* path, int recursive, void* props);
extern int go_zfs_rollback(zfs_handle_t* zhp, zfs_handle_t* snap, int force);

// Clone operations
extern int go_zfs_clone(libzfs_handle_t* hdl, char* snapname, char* clonename, void* props);
extern int go_zfs_promote(zfs_handle_t* zhp);
extern const char* go_zfs_get_origin(zfs_handle_t* zhp);
extern void* go_zfs_get_clones_nvlist(zfs_handle_t* zhp);
extern int go_zfs_is_clone(zfs_handle_t* zhp);
extern int go_zfs_get_clone_count(zfs_handle_t* zhp);

// Nvlist operations
extern void* go_nvlist_alloc();
extern void go_nvlist_free(void* nvl);
extern int go_nvlist_add_string(void* nvl, char* name, char* val);
extern int go_nvlist_add_nvlist(void* nvl, char* name, void* val);
extern int go_nvlist_add_nvlist_array(void* nvl, char* name, void** val, unsigned int nelem);
extern void* go_create_vdev_nvlist(char* type, char* path);

// Handle accessors
extern const char* go_zpool_get_name(zpool_handle_t* zhp);
extern uint64_t go_zpool_get_guid(zpool_handle_t* zhp);
extern const char* go_zfs_get_name(zfs_handle_t* zhp);
extern uint64_t go_zfs_get_guid(zfs_handle_t* zhp);
extern int go_zfs_get_type(zfs_handle_t* zhp);

// Vdev management operations
extern int go_zpool_add(zpool_handle_t* zhp, void* nvroot);
extern int go_zpool_attach(zpool_handle_t* zhp, char* old_disk, char* new_disk, void* props, int replacing);
extern int go_zpool_detach(zpool_handle_t* zhp, char* path);
extern int go_zpool_replace(zpool_handle_t* zhp, char* old_disk, char* new_disk, void* props);
extern int go_zpool_remove(zpool_handle_t* zhp, char* path);
extern int go_zpool_online(zpool_handle_t* zhp, char* path, int flags, int* newstate);
extern int go_zpool_offline(zpool_handle_t* zhp, char* path, int istmp);
extern int go_zpool_clear(zpool_handle_t* zhp, char* path, void* rewind_policy);

// Pool handle operations
extern zpool_handle_t* go_zpool_open(libzfs_handle_t* hdl, char* name);
extern void go_zpool_close(zpool_handle_t* zhp);

// Vdev creation helpers
extern void* go_create_mirror_vdev(void** children, unsigned int child_count);
extern void* go_create_raidz_vdev(void** children, unsigned int child_count, int parity);

// Vdev state constants
extern int go_get_vdev_state_offline();
extern int go_get_vdev_state_online();
extern int go_get_vdev_state_degraded();
extern int go_get_vdev_state_faulted();
extern int go_get_vdev_state_removed();
extern int go_get_vdev_state_unavail();

// Online flags
extern int go_get_zfs_online_checkremove();
extern int go_get_zfs_online_unspare();
extern int go_get_zfs_online_forcefault();
extern int go_get_zfs_online_expand();
*/
import "C"

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"
	"unsafe"

	"golang.org/x/sys/unix"

	"github.com/zombocoder/go-freebsd-libzfs/errors"

	// Import the cgo package to link its C code
	_ "github.com/zombocoder/go-freebsd-libzfs/internal/cgo"
)

// libzfsDriver implements the Driver interface using FreeBSD's libzfs
type libzfsDriver struct {
	mu sync.Mutex
	h  *C.libzfs_handle_t
}

// NewLibZFS creates a new libzfs-backed driver
func NewLibZFS() (Driver, error) {
	h := C.go_libzfs_init()
	if h == nil {
		return nil, fmt.Errorf("libzfs_init failed")
	}

	d := &libzfsDriver{h: h}
	runtime.SetFinalizer(d, (*libzfsDriver).finalize)
	return d, nil
}

func (d *libzfsDriver) finalize() {
	d.Close()
}

func (d *libzfsDriver) Close() error {
	defer d.enter()()

	if d.h != nil {
		C.go_libzfs_fini(d.h)
		d.h = nil
		runtime.SetFinalizer(d, nil)
	}
	return nil
}

func (d *libzfsDriver) RuntimeInfo(ctx context.Context) (string, string, string, error) {
	defer d.enter()()

	if d.h == nil {
		return "", "", "", fmt.Errorf("driver closed")
	}

	zfsVer := C.GoString(C.zfs_version_userland())

	kernel := "FreeBSD"
	var uts unix.Utsname
	if err := unix.Uname(&uts); err == nil {
		kernel = unix.ByteSliceToString(uts.Sysname[:]) + " " + unix.ByteSliceToString(uts.Release[:])
	}

	return "libzfs", zfsVer, kernel, nil
}

// mapPropSource converts a zprop_source_t bit value to a PropSource.
func mapPropSource(src C.int) PropSource {
	switch src {
	case C.ZPROP_SRC_DEFAULT:
		return PropSourceDefault
	case C.ZPROP_SRC_TEMPORARY:
		return PropSourceTemporary
	case C.ZPROP_SRC_LOCAL:
		return PropSourceLocal
	case C.ZPROP_SRC_INHERITED:
		return PropSourceInherited
	case C.ZPROP_SRC_RECEIVED:
		return PropSourceReceived
	default:
		return PropSourceNone
	}
}

// Pool iteration callback data
type poolIterData struct {
	pools []PoolInfo
	err   error
}

//export go_pool_iter_callback
func go_pool_iter_callback(zhp *C.zpool_handle_t, data unsafe.Pointer) C.int {
	// zpool_iter hands the handle to the callback, which must close it.
	C.go_iter_given()
	defer C.go_iter_close_zpool(zhp)

	iterData := (*poolIterData)(data)

	name := C.GoString(C.go_zpool_get_name(zhp))
	guid := uint64(C.go_zpool_get_guid(zhp))

	// Get pool state and health
	state := mapPoolState(int(C.go_zpool_get_state(zhp)))
	health := "UNKNOWN"
	var healthBuf [64]C.char
	if C.go_zpool_health(zhp, &healthBuf[0], C.size_t(len(healthBuf))) == 0 {
		health = C.GoString(&healthBuf[0])
	}

	pool := PoolInfo{
		Name:   name,
		GUID:   guid,
		Health: health,
		State:  state,
	}

	iterData.pools = append(iterData.pools, pool)
	return 0 // continue iteration
}

func (d *libzfsDriver) ListPools(ctx context.Context) ([]PoolInfo, error) {
	defer d.enter()()

	if d.h == nil {
		return nil, fmt.Errorf("driver closed")
	}

	var iterData *poolIterData
	errBefore := C.libzfs_errno(d.h)
	for attempt := 1; ; attempt++ {
		iterData = &poolIterData{}
		if C.go_iter_pools(d.h, (*[0]byte)(C.go_pool_iter_callback), unsafe.Pointer(iterData)) == 0 {
			break
		}
		if attempt >= maxIterAttempts || C.libzfs_errno(d.h) != C.EZFS_INTR {
			return nil, iterError("failed to iterate pools", d.h, errBefore)
		}
	}

	if iterData.err != nil {
		return nil, iterData.err
	}

	return iterData.pools, nil
}

func (d *libzfsDriver) GetPoolProps(ctx context.Context, poolName string, propNames []string) (map[string]PropertyInfo, error) {
	defer d.enter()()

	if d.h == nil {
		return nil, fmt.Errorf("driver closed")
	}

	// Get pool handle
	poolNameC := C.CString(poolName)
	defer C.free(unsafe.Pointer(poolNameC))

	zhp := C.zpool_open(d.h, poolNameC)
	if zhp == nil {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return nil, fmt.Errorf("failed to open pool %s (errno %d): %s", poolName, errno, desc)
	}
	defer C.zpool_close(zhp)

	properties := make(map[string]PropertyInfo)

	// Map of property names to their C constants
	propMap := map[string]func() C.int{
		PropNameSize:      func() C.int { return C.go_get_zpool_prop_size() },
		PropNameCapacity:  func() C.int { return C.go_get_zpool_prop_capacity() },
		PropNameHealth:    func() C.int { return C.go_get_zpool_prop_health() },
		PropNameGuid:      func() C.int { return C.go_get_zpool_prop_guid() },
		PropNameVersion:   func() C.int { return C.go_get_zpool_prop_version() },
		PropNameFree:      func() C.int { return C.go_get_zpool_prop_free() },
		PropNameAllocated: func() C.int { return C.go_get_zpool_prop_allocated() },
	}

	// If no specific properties requested, get all known properties
	if len(propNames) == 0 {
		for propName := range propMap {
			propNames = append(propNames, propName)
		}
	}

	// Retrieve each requested property
	for _, propName := range propNames {
		if propFunc, exists := propMap[propName]; exists {
			propValue := propFunc()
			buf := make([]byte, 1024)
			var src C.int
			ret := C.go_get_zpool_property(zhp, propValue, (*C.char)(unsafe.Pointer(&buf[0])), C.size_t(len(buf)), &src)

			if ret == 0 {
				// Successfully retrieved property
				value := C.GoString((*C.char)(unsafe.Pointer(&buf[0])))
				properties[propName] = PropertyInfo{
					Name:     propName,
					Value:    value,
					Source:   mapPropSource(src),
					Received: false,
				}
			}
		}
	}

	return properties, nil
}

// enter serialises access to the libzfs handle and keeps asynchronous signals
// away from the calling thread until the returned function is called, so
// libzfs ioctls are not interrupted (EZFS_INTR) by Go runtime signals. Use as
// "defer d.enter()()" at the top of every method that calls into libzfs.
func (d *libzfsDriver) enter() func() {
	d.mu.Lock()
	runtime.LockOSThread()
	C.go_sigs_block()
	return func() {
		C.go_sigs_restore()
		runtime.UnlockOSThread()
		d.mu.Unlock()
	}
}

// maxIterAttempts bounds the retries of a read-only iteration that libzfs
// reports as interrupted (EZFS_INTR) despite the signal mask, e.g. by a
// signal sent to the whole process while another thread is in libzfs.
const maxIterAttempts = 3

// OutstandingIterHandles returns the number of libzfs handles received in
// iteration callbacks and not closed yet. It is 0 whenever no iteration is
// running; a larger value means a handle leak. Used by tests.
func OutstandingIterHandles() int64 {
	return int64(C.go_iter_outstanding())
}

// iterError describes a failed iteration. libzfs never clears its error, so
// the libzfs error is reported only if it changed during the iteration
// (before is the code taken just before it); otherwise the callback aborted
// and the message says so.
func iterError(what string, h *C.libzfs_handle_t, before C.int) error {
	errno := C.libzfs_errno(h)
	if errno == 0 || errno == before {
		return fmt.Errorf("%s: iteration aborted", what)
	}
	return fmt.Errorf("%s (errno %d): %s", what, errno, C.GoString(C.libzfs_error_description(h)))
}

// maxPropLen is the buffer size for a native property value; it matches
// ZAP_MAXVALUELEN so long values (origin, clones, mountpoint) are not cut.
const maxPropLen = 8192

// Dataset iteration callback data
type datasetIterData struct {
	datasets []DatasetInfo
	err      error
}

//export go_dataset_iter_callback
func go_dataset_iter_callback(zhp *C.zfs_handle_t, data unsafe.Pointer) C.int {
	// The C iteration wrappers (go_iter_datasets, go_iter_all_datasets) own
	// and close the handle after this returns.
	iterData := (*datasetIterData)(data)

	name := C.GoString(C.go_zfs_get_name(zhp))
	guid := uint64(C.go_zfs_get_guid(zhp))
	dsType := mapDatasetType(int(C.go_zfs_get_type(zhp)))

	dataset := DatasetInfo{
		Name: name,
		Type: dsType,
		GUID: guid,
	}

	iterData.datasets = append(iterData.datasets, dataset)
	return 0 // continue iteration
}

// Helper function to map C dataset type enum to Go type
func mapDatasetType(zfsType int) DatasetType {
	switch zfsType {
	case 1: // ZFS_TYPE_FILESYSTEM
		return DatasetFilesystem
	case 2: // ZFS_TYPE_SNAPSHOT
		return DatasetSnapshot
	case 4: // ZFS_TYPE_VOLUME
		return DatasetVolume
	case 8: // ZFS_TYPE_POOL
		return DatasetFilesystem // Treat pool as filesystem for now
	case 16: // ZFS_TYPE_BOOKMARK
		return DatasetBookmark
	default:
		return DatasetFilesystem // Default fallback
	}
}

// ListDatasets returns root datasets, or every dataset (including snapshots)
// when recursive is true.
func (d *libzfsDriver) ListDatasets(ctx context.Context, recursive bool) ([]DatasetInfo, error) {
	return d.ListDatasetsByType(ctx, nil, recursive)
}

func (d *libzfsDriver) ListDatasetsInPool(ctx context.Context, poolName string, recursive bool) ([]DatasetInfo, error) {
	allDatasets, err := d.ListDatasets(ctx, recursive)
	if err != nil {
		return nil, err
	}

	var poolDatasets []DatasetInfo
	for _, dataset := range allDatasets {
		if dataset.Name == poolName ||
			strings.HasPrefix(dataset.Name, poolName+"/") ||
			strings.HasPrefix(dataset.Name, poolName+"@") ||
			strings.HasPrefix(dataset.Name, poolName+"#") {
			poolDatasets = append(poolDatasets, dataset)
		}
	}

	return poolDatasets, nil
}

// ListDatasetsByType lists datasets sorted by name. Without recursive only the
// root dataset of each pool is returned; with recursive all filesystems,
// volumes and snapshots are returned. dsType optionally filters by type.
func (d *libzfsDriver) ListDatasetsByType(ctx context.Context, dsType *DatasetType, recursive bool) ([]DatasetInfo, error) {
	defer d.enter()()

	if d.h == nil {
		return nil, fmt.Errorf("driver closed")
	}

	var iterData *datasetIterData
	errBefore := C.libzfs_errno(d.h)
	for attempt := 1; ; attempt++ {
		iterData = &datasetIterData{}
		var ret C.int
		if recursive {
			ret = C.go_iter_all_datasets(d.h, (*[0]byte)(C.go_dataset_iter_callback), unsafe.Pointer(iterData))
		} else {
			ret = C.go_iter_datasets(d.h, (*[0]byte)(C.go_dataset_iter_callback), unsafe.Pointer(iterData))
		}
		if ret == 0 {
			break
		}
		if attempt >= maxIterAttempts || C.libzfs_errno(d.h) != C.EZFS_INTR {
			return nil, iterError("failed to iterate datasets", d.h, errBefore)
		}
	}

	if iterData.err != nil {
		return nil, iterData.err
	}

	// Deduplicate by name and apply the optional type filter.
	seen := make(map[string]struct{}, len(iterData.datasets))
	result := make([]DatasetInfo, 0, len(iterData.datasets))
	for _, dataset := range iterData.datasets {
		if _, dup := seen[dataset.Name]; dup {
			continue
		}
		seen[dataset.Name] = struct{}{}
		if dsType != nil && dataset.Type != *dsType {
			continue
		}
		result = append(result, dataset)
	}

	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func (d *libzfsDriver) GetDatasetProps(ctx context.Context, datasetName string, propNames []string) (map[string]PropertyInfo, error) {
	defer d.enter()()

	if d.h == nil {
		return nil, fmt.Errorf("driver closed")
	}

	// Get dataset handle
	datasetNameC := C.CString(datasetName)
	defer C.free(unsafe.Pointer(datasetNameC))

	zhp := C.zfs_open(d.h, datasetNameC, C.ZFS_TYPE_DATASET)
	if zhp == nil {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return nil, fmt.Errorf("failed to open dataset %s (errno %d): %s", datasetName, errno, desc)
	}
	defer C.zfs_close(zhp)

	properties := make(map[string]PropertyInfo)

	// Map of property names to their C constants
	propMap := map[string]func() C.int{
		PropNameUsed:          func() C.int { return C.go_get_zfs_prop_used() },
		PropNameAvail:         func() C.int { return C.go_get_zfs_prop_available() },
		PropNameRefer:         func() C.int { return C.go_get_zfs_prop_referenced() },
		PropNameCompressratio: func() C.int { return C.go_get_zfs_prop_compressratio() },
		PropNameQuota:         func() C.int { return C.go_get_zfs_prop_quota() },
		PropNameReservation:   func() C.int { return C.go_get_zfs_prop_reservation() },
		PropNameRecordsize:    func() C.int { return C.go_get_zfs_prop_recordsize() },
		PropNameMountpoint:    func() C.int { return C.go_get_zfs_prop_mountpoint() },
		PropNameCompression:   func() C.int { return C.go_get_zfs_prop_compression() },
	}

	// If no specific properties requested, get all known properties
	if len(propNames) == 0 {
		for propName := range propMap {
			propNames = append(propNames, propName)
		}
	}

	// Retrieve each requested property. User properties (names with a colon)
	// come from the dataset's user property list; an unset one is omitted.
	// Native names are resolved through libzfs, so every native property is
	// available; a name that is neither is an error.
	var userProps map[string]UserPropertyInfo
	for _, propName := range propNames {
		if strings.Contains(propName, ":") {
			if userProps == nil {
				userProps = readUserProps(zhp)
			}
			if up, ok := userProps[propName]; ok {
				properties[propName] = PropertyInfo{
					Name:   propName,
					Value:  up.Value,
					Source: up.Source,
				}
			}
			continue
		}

		var propValue C.int
		if propFunc, exists := propMap[propName]; exists {
			propValue = propFunc()
		} else {
			cName := C.CString(propName)
			propValue = C.int(C.zfs_name_to_prop(cName))
			C.free(unsafe.Pointer(cName))
			if propValue < 0 {
				return nil, fmt.Errorf("unknown property %q: %w", propName, errors.ErrInvalidArgument)
			}
		}

		buf := make([]byte, maxPropLen)
		var src C.int
		ret := C.go_get_zfs_property(zhp, propValue, (*C.char)(unsafe.Pointer(&buf[0])), C.size_t(len(buf)), &src)
		if ret == 0 {
			properties[propName] = PropertyInfo{
				Name:     propName,
				Value:    C.GoString((*C.char)(unsafe.Pointer(&buf[0]))),
				Source:   mapPropSource(src),
				Received: false,
			}
		}
	}

	return properties, nil
}

// cStringArray copies strs into a C-allocated char* array. It returns nil for
// an empty slice. The returned function frees all memory.
func cStringArray(strs []string) (**C.char, func()) {
	if len(strs) == 0 {
		return nil, func() {}
	}
	arr := (**C.char)(C.calloc(C.size_t(len(strs)), C.size_t(unsafe.Sizeof((*C.char)(nil)))))
	elems := unsafe.Slice(arr, len(strs))
	for i, s := range strs {
		elems[i] = C.CString(s)
	}
	return arr, func() {
		for _, e := range elems {
			C.free(unsafe.Pointer(e))
		}
		C.free(unsafe.Pointer(arr))
	}
}

// DiscoverPools scans for pools that are not imported, like `zpool import`
// without arguments. Results are sorted by name.
func (d *libzfsDriver) DiscoverPools(ctx context.Context, opts DiscoverOptions) ([]ImportablePool, error) {
	defer d.enter()()

	if d.h == nil {
		return nil, fmt.Errorf("driver closed")
	}

	var poolNameC *C.char
	if opts.PoolName != "" {
		poolNameC = C.CString(opts.PoolName)
		defer C.free(unsafe.Pointer(poolNameC))
	}
	pathsC, freePaths := cStringArray(opts.SearchPaths)
	defer freePaths()

	errBuf := make([]byte, 1024)
	var searchErr C.int
	pools := C.go_zpool_search(d.h, poolNameC, pathsC, C.int(len(opts.SearchPaths)),
		(*C.char)(unsafe.Pointer(&errBuf[0])), C.size_t(len(errBuf)), &searchErr)
	if pools == nil {
		if searchErr != 0 {
			return nil, fmt.Errorf("pool discovery failed: %s", C.GoString((*C.char)(unsafe.Pointer(&errBuf[0]))))
		}
		return []ImportablePool{}, nil
	}
	defer C.nvlist_free(pools)

	result := []ImportablePool{}
	for elem := C.nvlist_next_nvpair(pools, nil); elem != nil; elem = C.nvlist_next_nvpair(pools, elem) {
		var config *C.nvlist_t
		if C.nvpair_value_nvlist(elem, &config) != 0 {
			continue
		}

		var name, stateName, health *C.char
		var guid, state C.uint64_t
		if C.go_import_config_summary(config, &name, &guid, &state, &stateName, &health) != 0 {
			continue
		}
		if state == C.POOL_STATE_DESTROYED && !opts.IncludeDestroyed {
			continue
		}

		n := C.go_import_config_device_paths(config, nil, 0)
		devices := make([]string, 0, int(n))
		if n > 0 {
			buf := make([]*C.char, int(n))
			C.go_import_config_device_paths(config, &buf[0], n)
			for _, p := range buf {
				devices = append(devices, C.GoString(p))
			}
		}

		result = append(result, ImportablePool{
			Name:        C.GoString(name),
			GUID:        uint64(guid),
			State:       C.GoString(stateName),
			Health:      C.GoString(health),
			DevicePaths: devices,
		})
	}

	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func (d *libzfsDriver) ImportPool(ctx context.Context, poolName string, opts ImportOptions) error {
	defer d.enter()()

	if d.h == nil {
		return fmt.Errorf("driver closed")
	}

	poolNameC := C.CString(poolName)
	defer C.free(unsafe.Pointer(poolNameC))

	var newNameC, altRootC *C.char
	if opts.NewName != "" {
		newNameC = C.CString(opts.NewName)
		defer C.free(unsafe.Pointer(newNameC))
	}
	if opts.AltRoot != "" {
		altRootC = C.CString(opts.AltRoot)
		defer C.free(unsafe.Pointer(altRootC))
	}

	pathsC, freePaths := cStringArray(opts.SearchPaths)
	defer freePaths()

	flags := C.int(C.ZFS_IMPORT_NORMAL)
	if opts.Force {
		flags |= C.ZFS_IMPORT_ANY_HOST
	}
	destroyed := C.int(0)
	if opts.Destroyed {
		destroyed = 1
	}

	errBuf := make([]byte, 1024)
	ret := C.go_zpool_import(d.h, poolNameC, pathsC, C.int(len(opts.SearchPaths)),
		newNameC, altRootC, flags, destroyed,
		(*C.char)(unsafe.Pointer(&errBuf[0])), C.size_t(len(errBuf)))

	switch ret {
	case importOK:
		return nil
	case importErrNotFound:
		if opts.Destroyed {
			return fmt.Errorf("cannot import pool %s: no such destroyed pool available for import", poolName)
		}
		return fmt.Errorf("cannot import pool %s: no such pool available for import", poolName)
	case importErrAmbiguous:
		return fmt.Errorf("cannot import pool %s: multiple importable pools share this name", poolName)
	case importErrSearchFailed:
		return fmt.Errorf("cannot import pool %s: device search failed: %s",
			poolName, C.GoString((*C.char)(unsafe.Pointer(&errBuf[0]))))
	case importErrNoMem:
		return fmt.Errorf("cannot import pool %s: out of memory", poolName)
	default:
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		stage := map[C.int]string{
			importErrImport: "import failed",
			importErrOpen:   "imported but could not be opened",
			importErrMount:  "imported but some datasets could not be mounted",
		}[ret]
		return fmt.Errorf("cannot import pool %s: %s (errno %d): %s", poolName, stage, errno, desc)
	}
}

// Return codes of go_zpool_import; keep in sync with GO_IMPORT_* in internal/cgo.
const (
	importOK              C.int = 0
	importErrNotFound     C.int = -1
	importErrAmbiguous    C.int = -2
	importErrSearchFailed C.int = -3
	importErrImport       C.int = -4
	importErrOpen         C.int = -5
	importErrMount        C.int = -6
	importErrNoMem        C.int = -7
)

func (d *libzfsDriver) ExportPool(ctx context.Context, poolName string, opts ExportOptions) error {
	defer d.enter()()

	if d.h == nil {
		return fmt.Errorf("driver closed")
	}

	// Get pool handle
	poolNameC := C.CString(poolName)
	defer C.free(unsafe.Pointer(poolNameC))

	zhp := C.zpool_open(d.h, poolNameC)
	if zhp == nil {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return fmt.Errorf("failed to open pool %s for export (errno %d): %s", poolName, errno, desc)
	}
	defer C.zpool_close(zhp)

	// Export the pool
	var ret C.int
	if opts.Force {
		ret = C.go_zpool_export_force(zhp)
	} else {
		messageC := C.CString(opts.Message)
		defer C.free(unsafe.Pointer(messageC))
		ret = C.go_zpool_export(zhp, C.int(0), messageC)
	}

	if ret != 0 {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return fmt.Errorf("failed to export pool %s (errno %d): %s", poolName, errno, desc)
	}

	return nil
}

func (d *libzfsDriver) CreatePool(ctx context.Context, poolName string, vdevs []string, opts CreateOptions) error {
	defer d.enter()()

	if d.h == nil {
		return fmt.Errorf("driver closed")
	}

	if len(vdevs) == 0 {
		return fmt.Errorf("at least one vdev is required to create a pool")
	}

	// Create pool name C string
	cPoolName := C.CString(poolName)
	defer C.free(unsafe.Pointer(cPoolName))

	// Create vdev root with children using helper
	nvroot, vdevList, err := d.createVdevRoot(vdevs)
	if err != nil {
		return err
	}
	defer d.freeNvlist(nvroot)

	// Create pool properties nvlist
	poolProps, err := d.createPropsNvlist(opts.Properties)
	if err != nil {
		// Clean up vdev nvlists
		for _, vdev := range vdevList {
			d.freeNvlist(vdev)
		}
		return fmt.Errorf("failed to create pool properties: %w", err)
	}
	defer d.freeNvlist(poolProps)

	// Create filesystem properties nvlist
	fsProps, err := d.createPropsNvlist(opts.FsProperties)
	if err != nil {
		// Clean up vdev nvlists
		for _, vdev := range vdevList {
			d.freeNvlist(vdev)
		}
		return fmt.Errorf("failed to create filesystem properties: %w", err)
	}
	defer d.freeNvlist(fsProps)

	// Create the pool
	createRet := C.go_zpool_create(d.h, cPoolName, nvroot, poolProps, fsProps)

	// Clean up vdev nvlists
	for _, vdev := range vdevList {
		d.freeNvlist(vdev)
	}

	if createRet != 0 {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return fmt.Errorf("failed to create pool %s (errno %d): %s", poolName, errno, desc)
	}

	return nil
}

func (d *libzfsDriver) DestroyPool(ctx context.Context, poolName string) error {
	defer d.enter()()

	if d.h == nil {
		return fmt.Errorf("driver closed")
	}

	// Get pool handle
	poolNameC := C.CString(poolName)
	defer C.free(unsafe.Pointer(poolNameC))

	zhp := C.zpool_open(d.h, poolNameC)
	if zhp == nil {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return fmt.Errorf("failed to open pool %s for destruction (errno %d): %s", poolName, errno, desc)
	}
	defer C.zpool_close(zhp)

	// Destroy the pool
	messageC := C.CString("Destroyed via Go ZFS library")
	defer C.free(unsafe.Pointer(messageC))

	ret := C.go_zpool_destroy(zhp, messageC)
	if ret != 0 {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return fmt.Errorf("failed to destroy pool %s (errno %d): %s", poolName, errno, desc)
	}

	return nil
}

func (d *libzfsDriver) CreateDataset(ctx context.Context, datasetName string, dsType DatasetType, props map[string]string) error {
	defer d.enter()()

	if d.h == nil {
		return fmt.Errorf("driver closed")
	}

	// Convert dataset name to C string
	datasetNameC := C.CString(datasetName)
	defer C.free(unsafe.Pointer(datasetNameC))

	// Map dataset type to ZFS type constant
	var zfsType C.int
	switch dsType {
	case DatasetFilesystem:
		zfsType = C.go_get_zfs_type_filesystem()
	case DatasetVolume:
		zfsType = C.go_get_zfs_type_volume()
	default:
		return fmt.Errorf("unsupported dataset type: %v", dsType)
	}

	// Create properties nvlist for creation (required for volumes with volsize)
	propsNvlist, err := d.createPropsNvlist(props)
	if err != nil {
		return fmt.Errorf("failed to create properties nvlist: %w", err)
	}
	defer d.freeNvlist(propsNvlist)

	// Create the dataset with properties
	ret := C.go_zfs_create(d.h, datasetNameC, zfsType, propsNvlist)
	if ret != 0 {
		errno, desc := d.getLibzfsError()
		return fmt.Errorf("failed to create dataset %s (errno %d): %s", datasetName, errno, desc)
	}

	return nil
}

func (d *libzfsDriver) SetDatasetProp(ctx context.Context, datasetName, propName, propValue string) error {
	defer d.enter()()

	if d.h == nil {
		return fmt.Errorf("driver is closed")
	}

	// Open the dataset handle
	zhp, err := d.openDatasetHandle(datasetName)
	if err != nil {
		return err
	}
	defer C.zfs_close(zhp)

	// Set the property
	cPropName := C.CString(propName)
	defer C.free(unsafe.Pointer(cPropName))

	cPropValue := C.CString(propValue)
	defer C.free(unsafe.Pointer(cPropValue))

	ret := C.go_zfs_prop_set(zhp, cPropName, cPropValue)
	if ret != 0 {
		errno, desc := d.getLibzfsError()
		return fmt.Errorf("failed to set property %s=%s on dataset %s (errno %d): %s", propName, propValue, datasetName, errno, desc)
	}

	return nil
}

func (d *libzfsDriver) DestroyDataset(ctx context.Context, datasetName string, recursive bool) error {
	defer d.enter()()

	if d.h == nil {
		return fmt.Errorf("driver closed")
	}

	// Convert dataset name to C string
	datasetNameC := C.CString(datasetName)
	defer C.free(unsafe.Pointer(datasetNameC))

	// Open the dataset
	zhp := C.zfs_open(d.h, datasetNameC, C.ZFS_TYPE_DATASET)
	if zhp == nil {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return fmt.Errorf("failed to open dataset %s for destruction (errno %d): %s", datasetName, errno, desc)
	}
	defer C.zfs_close(zhp)

	// Destroy the dataset
	var ret C.int
	if recursive {
		ret = C.go_zfs_destroy_recursive(zhp, nil)
	} else {
		ret = C.go_zfs_destroy(zhp, C.int(0)) // Don't defer destroy
	}

	if ret != 0 {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return fmt.Errorf("failed to destroy dataset %s (errno %d): %s", datasetName, errno, desc)
	}

	return nil
}

func (d *libzfsDriver) CreateSnapshot(ctx context.Context, snapshotName string, recursive bool, props map[string]string) error {
	defer d.enter()()

	if d.h == nil {
		return fmt.Errorf("driver is closed")
	}

	cSnapshotName := C.CString(snapshotName)
	defer C.free(unsafe.Pointer(cSnapshotName))

	// Convert properties to nvlist if provided
	propsNvlist, err := d.createPropsNvlist(props)
	if err != nil {
		return fmt.Errorf("failed to create properties nvlist: %w", err)
	}
	defer d.freeNvlist(propsNvlist)

	var recursiveFlag C.int
	if recursive {
		recursiveFlag = 1
	} else {
		recursiveFlag = 0
	}

	ret := C.go_zfs_snapshot(d.h, cSnapshotName, recursiveFlag, propsNvlist)
	if ret != 0 {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return fmt.Errorf("failed to create snapshot %s (errno %d): %s", snapshotName, errno, desc)
	}

	return nil
}

func (d *libzfsDriver) DestroySnapshot(ctx context.Context, snapshotName string) error {
	defer d.enter()()

	if d.h == nil {
		return fmt.Errorf("driver is closed")
	}

	// Open the snapshot handle
	cSnapshotName := C.CString(snapshotName)
	defer C.free(unsafe.Pointer(cSnapshotName))

	zhp := C.zfs_open(d.h, cSnapshotName, C.int(C.go_get_zfs_type_snapshot()))
	if zhp == nil {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return fmt.Errorf("failed to open snapshot %s (errno %d): %s", snapshotName, errno, desc)
	}
	defer C.zfs_close(zhp)

	// Destroy the snapshot
	ret := C.go_zfs_destroy(zhp, C.int(0)) // Don't defer destroy
	if ret != 0 {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return fmt.Errorf("failed to destroy snapshot %s (errno %d): %s", snapshotName, errno, desc)
	}

	return nil
}

func (d *libzfsDriver) RollbackToSnapshot(ctx context.Context, datasetName, snapshotName string, force bool) error {
	defer d.enter()()

	if d.h == nil {
		return fmt.Errorf("driver is closed")
	}

	// Open the dataset handle
	cDatasetName := C.CString(datasetName)
	defer C.free(unsafe.Pointer(cDatasetName))

	// Open as filesystem or volume - try filesystem first
	zhp := C.zfs_open(d.h, cDatasetName, C.int(C.go_get_zfs_type_filesystem()))
	if zhp == nil {
		// Try volume type
		zhp = C.zfs_open(d.h, cDatasetName, C.int(C.go_get_zfs_type_volume()))
		if zhp == nil {
			errno := C.go_libzfs_errno(d.h)
			desc := C.GoString(C.go_libzfs_error_description(d.h))
			return fmt.Errorf("failed to open dataset %s (errno %d): %s", datasetName, errno, desc)
		}
	}
	defer C.zfs_close(zhp)

	// Open the snapshot handle
	cSnapshotName := C.CString(snapshotName)
	defer C.free(unsafe.Pointer(cSnapshotName))

	snapzhp := C.zfs_open(d.h, cSnapshotName, C.int(C.go_get_zfs_type_snapshot()))
	if snapzhp == nil {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return fmt.Errorf("failed to open snapshot %s (errno %d): %s", snapshotName, errno, desc)
	}
	defer C.zfs_close(snapzhp)

	// Perform the rollback
	var forceFlag C.int
	if force {
		forceFlag = 1
	} else {
		forceFlag = 0
	}

	ret := C.go_zfs_rollback(zhp, snapzhp, forceFlag)
	if ret != 0 {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return fmt.Errorf("failed to rollback %s to snapshot %s (errno %d): %s", datasetName, snapshotName, errno, desc)
	}

	return nil
}

// Clone operations

func (d *libzfsDriver) CreateClone(ctx context.Context, snapshotName, cloneName string, props map[string]string) error {
	defer d.enter()()

	if d.h == nil {
		return fmt.Errorf("driver is closed")
	}

	// Convert snapshot and clone names to C strings
	cSnapshotName := C.CString(snapshotName)
	defer C.free(unsafe.Pointer(cSnapshotName))
	cCloneName := C.CString(cloneName)
	defer C.free(unsafe.Pointer(cCloneName))

	// Create properties nvlist if provided
	var propsNvlist unsafe.Pointer
	if len(props) > 0 {
		propsNvlist = C.go_nvlist_alloc()
		if propsNvlist == nil {
			return fmt.Errorf("failed to allocate properties nvlist")
		}
		defer C.go_nvlist_free(propsNvlist)

		// Add properties to nvlist
		for key, value := range props {
			cKey := C.CString(key)
			cValue := C.CString(value)
			ret := C.go_nvlist_add_string(propsNvlist, cKey, cValue)
			C.free(unsafe.Pointer(cKey))
			C.free(unsafe.Pointer(cValue))
			if ret != 0 {
				return fmt.Errorf("failed to add property %s to nvlist", key)
			}
		}
	}

	// Create the clone
	ret := C.go_zfs_clone(d.h, cSnapshotName, cCloneName, propsNvlist)
	if ret != 0 {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return fmt.Errorf("failed to create clone %s from snapshot %s (errno %d): %s", cloneName, snapshotName, errno, desc)
	}

	return nil
}

func (d *libzfsDriver) PromoteClone(ctx context.Context, cloneName string) error {
	defer d.enter()()

	if d.h == nil {
		return fmt.Errorf("driver is closed")
	}

	// Open the clone dataset
	cCloneName := C.CString(cloneName)
	defer C.free(unsafe.Pointer(cCloneName))

	zhp := C.zfs_open(d.h, cCloneName, C.int(C.go_get_zfs_type_filesystem()))
	if zhp == nil {
		// Try as volume
		zhp = C.zfs_open(d.h, cCloneName, C.int(C.go_get_zfs_type_volume()))
		if zhp == nil {
			errno := C.go_libzfs_errno(d.h)
			desc := C.GoString(C.go_libzfs_error_description(d.h))
			return fmt.Errorf("failed to open clone %s (errno %d): %s", cloneName, errno, desc)
		}
	}
	defer C.zfs_close(zhp)

	// Check if it's actually a clone
	isClone := C.go_zfs_is_clone(zhp)
	if isClone == 0 {
		return fmt.Errorf("dataset %s is not a clone", cloneName)
	}

	// Promote the clone
	ret := C.go_zfs_promote(zhp)
	if ret != 0 {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return fmt.Errorf("failed to promote clone %s (errno %d): %s", cloneName, errno, desc)
	}

	return nil
}

func (d *libzfsDriver) GetCloneInfo(ctx context.Context, datasetName string) (*CloneInfo, error) {
	defer d.enter()()

	if d.h == nil {
		return nil, fmt.Errorf("driver is closed")
	}

	// Open the dataset
	cDatasetName := C.CString(datasetName)
	defer C.free(unsafe.Pointer(cDatasetName))

	zhp := C.zfs_open(d.h, cDatasetName, C.int(C.go_get_zfs_type_filesystem()|C.go_get_zfs_type_volume()|C.go_get_zfs_type_snapshot()))
	if zhp == nil {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return nil, fmt.Errorf("failed to open dataset %s (errno %d): %s", datasetName, errno, desc)
	}
	defer C.zfs_close(zhp)

	info := &CloneInfo{
		Name: datasetName,
	}

	// Check if this dataset is a clone
	isClone := C.go_zfs_is_clone(zhp)
	info.IsClone = isClone != 0

	if info.IsClone {
		// Get the origin snapshot
		origin := C.go_zfs_get_origin(zhp)
		if origin != nil {
			info.Origin = C.GoString(origin)
		}
	}

	// Get clone count (if this is a snapshot)
	dsType := C.go_zfs_get_type(zhp)
	if dsType == C.go_get_zfs_type_snapshot() {
		cloneCount := C.go_zfs_get_clone_count(zhp)
		info.CloneCount = int(cloneCount)
	}

	// Get list of clones (if this is a snapshot)
	if dsType == C.go_get_zfs_type_snapshot() {
		clones, err := d.getCloneList(zhp)
		if err == nil {
			info.Dependents = clones
		}
	}

	return info, nil
}

func (d *libzfsDriver) ListClones(ctx context.Context, snapshotName string) ([]string, error) {
	defer d.enter()()

	if d.h == nil {
		return nil, fmt.Errorf("driver is closed")
	}

	// Open the snapshot
	cSnapshotName := C.CString(snapshotName)
	defer C.free(unsafe.Pointer(cSnapshotName))

	zhp := C.zfs_open(d.h, cSnapshotName, C.int(C.go_get_zfs_type_snapshot()))
	if zhp == nil {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return nil, fmt.Errorf("failed to open snapshot %s (errno %d): %s", snapshotName, errno, desc)
	}
	defer C.zfs_close(zhp)

	return d.getCloneList(zhp)
}

func (d *libzfsDriver) DestroyClone(ctx context.Context, cloneName string, force bool) error {
	defer d.enter()()

	if d.h == nil {
		return fmt.Errorf("driver is closed")
	}

	// Open the clone dataset
	cCloneName := C.CString(cloneName)
	defer C.free(unsafe.Pointer(cCloneName))

	zhp := C.zfs_open(d.h, cCloneName, C.int(C.go_get_zfs_type_filesystem()|C.go_get_zfs_type_volume()))
	if zhp == nil {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return fmt.Errorf("failed to open clone %s (errno %d): %s", cloneName, errno, desc)
	}
	defer C.zfs_close(zhp)

	// Check if it's actually a clone (unless force is specified)
	if !force {
		isClone := C.go_zfs_is_clone(zhp)
		if isClone == 0 {
			return fmt.Errorf("dataset %s is not a clone", cloneName)
		}
	}

	// Destroy the clone using the regular dataset destroy function
	var forceFlag C.int
	if force {
		forceFlag = 1
	} else {
		forceFlag = 0
	}

	ret := C.go_zfs_destroy(zhp, forceFlag)
	if ret != 0 {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return fmt.Errorf("failed to destroy clone %s (errno %d): %s", cloneName, errno, desc)
	}

	return nil
}

// Helper function to get list of clones from a snapshot
func (d *libzfsDriver) getCloneList(zhp *C.zfs_handle_t) ([]string, error) {
	clonesNvlist := C.go_zfs_get_clones_nvlist(zhp)
	if clonesNvlist == nil {
		return []string{}, nil // No clones, not an error
	}
	defer C.go_nvlist_free(clonesNvlist)

	// Parse the nvlist to extract clone names
	// This is a simplified implementation - in production you'd want to
	// properly parse the nvlist structure
	var clones []string

	// For now, return empty list - proper nvlist parsing would be complex
	// In a production implementation, you'd need to iterate through the nvlist
	// and extract clone dataset names

	return clones, nil
}

// Vdev management operations

func (d *libzfsDriver) AddVdev(ctx context.Context, poolName string, vdevSpec VdevSpec) error {
	defer d.enter()()

	if d.h == nil {
		return fmt.Errorf("driver is closed")
	}

	// Open the pool
	cPoolName := C.CString(poolName)
	defer C.free(unsafe.Pointer(cPoolName))

	zhp := C.go_zpool_open(d.h, cPoolName)
	if zhp == nil {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return fmt.Errorf("failed to open pool %s (errno %d): %s", poolName, errno, desc)
	}
	defer C.go_zpool_close(zhp)

	// Create vdev nvlist based on spec
	vdevNvlist, err := d.createComplexVdevNvlist(vdevSpec)
	if err != nil {
		return fmt.Errorf("failed to create vdev specification: %w", err)
	}
	defer C.go_nvlist_free(vdevNvlist)

	// Add the vdev to the pool
	ret := C.go_zpool_add(zhp, vdevNvlist)
	if ret != 0 {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return fmt.Errorf("failed to add vdev to pool %s (errno %d): %s", poolName, errno, desc)
	}

	return nil
}

func (d *libzfsDriver) AttachVdev(ctx context.Context, poolName, oldDevice, newDevice string, replacing bool) error {
	defer d.enter()()

	if d.h == nil {
		return fmt.Errorf("driver is closed")
	}

	// Open the pool
	cPoolName := C.CString(poolName)
	defer C.free(unsafe.Pointer(cPoolName))

	zhp := C.go_zpool_open(d.h, cPoolName)
	if zhp == nil {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return fmt.Errorf("failed to open pool %s (errno %d): %s", poolName, errno, desc)
	}
	defer C.go_zpool_close(zhp)

	// Convert device names to C strings
	cOldDevice := C.CString(oldDevice)
	defer C.free(unsafe.Pointer(cOldDevice))
	cNewDevice := C.CString(newDevice)
	defer C.free(unsafe.Pointer(cNewDevice))

	var replacingFlag C.int
	if replacing {
		replacingFlag = 1
	}

	// Attach the vdev
	ret := C.go_zpool_attach(zhp, cOldDevice, cNewDevice, nil, replacingFlag)
	if ret != 0 {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return fmt.Errorf("failed to attach vdev %s to %s in pool %s (errno %d): %s", newDevice, oldDevice, poolName, errno, desc)
	}

	return nil
}

func (d *libzfsDriver) DetachVdev(ctx context.Context, poolName, device string) error {
	defer d.enter()()

	if d.h == nil {
		return fmt.Errorf("driver is closed")
	}

	// Open the pool
	cPoolName := C.CString(poolName)
	defer C.free(unsafe.Pointer(cPoolName))

	zhp := C.go_zpool_open(d.h, cPoolName)
	if zhp == nil {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return fmt.Errorf("failed to open pool %s (errno %d): %s", poolName, errno, desc)
	}
	defer C.go_zpool_close(zhp)

	// Convert device name to C string
	cDevice := C.CString(device)
	defer C.free(unsafe.Pointer(cDevice))

	// Detach the vdev
	ret := C.go_zpool_detach(zhp, cDevice)
	if ret != 0 {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return fmt.Errorf("failed to detach vdev %s from pool %s (errno %d): %s", device, poolName, errno, desc)
	}

	return nil
}

func (d *libzfsDriver) ReplaceVdev(ctx context.Context, poolName, oldDevice, newDevice string) error {
	defer d.enter()()

	if d.h == nil {
		return fmt.Errorf("driver is closed")
	}

	// Open the pool
	cPoolName := C.CString(poolName)
	defer C.free(unsafe.Pointer(cPoolName))

	zhp := C.go_zpool_open(d.h, cPoolName)
	if zhp == nil {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return fmt.Errorf("failed to open pool %s (errno %d): %s", poolName, errno, desc)
	}
	defer C.go_zpool_close(zhp)

	// Convert device names to C strings
	cOldDevice := C.CString(oldDevice)
	defer C.free(unsafe.Pointer(cOldDevice))
	cNewDevice := C.CString(newDevice)
	defer C.free(unsafe.Pointer(cNewDevice))

	// Replace the vdev
	ret := C.go_zpool_replace(zhp, cOldDevice, cNewDevice, nil)
	if ret != 0 {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return fmt.Errorf("failed to replace vdev %s with %s in pool %s (errno %d): %s", oldDevice, newDevice, poolName, errno, desc)
	}

	return nil
}

func (d *libzfsDriver) RemoveVdev(ctx context.Context, poolName, device string) error {
	defer d.enter()()

	if d.h == nil {
		return fmt.Errorf("driver is closed")
	}

	// Open the pool
	cPoolName := C.CString(poolName)
	defer C.free(unsafe.Pointer(cPoolName))

	zhp := C.go_zpool_open(d.h, cPoolName)
	if zhp == nil {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return fmt.Errorf("failed to open pool %s (errno %d): %s", poolName, errno, desc)
	}
	defer C.go_zpool_close(zhp)

	// Convert device name to C string
	cDevice := C.CString(device)
	defer C.free(unsafe.Pointer(cDevice))

	// Remove the vdev
	ret := C.go_zpool_remove(zhp, cDevice)
	if ret != 0 {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return fmt.Errorf("failed to remove vdev %s from pool %s (errno %d): %s", device, poolName, errno, desc)
	}

	return nil
}

func (d *libzfsDriver) OnlineVdev(ctx context.Context, poolName, device string, flags VdevOnlineFlags) error {
	defer d.enter()()

	if d.h == nil {
		return fmt.Errorf("driver is closed")
	}

	// Open the pool
	cPoolName := C.CString(poolName)
	defer C.free(unsafe.Pointer(cPoolName))

	zhp := C.go_zpool_open(d.h, cPoolName)
	if zhp == nil {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return fmt.Errorf("failed to open pool %s (errno %d): %s", poolName, errno, desc)
	}
	defer C.go_zpool_close(zhp)

	// Convert device name to C string
	cDevice := C.CString(device)
	defer C.free(unsafe.Pointer(cDevice))

	// Convert flags to C int
	var cFlags C.int = 0
	if flags&VdevOnlineCheckRemove != 0 {
		cFlags |= C.go_get_zfs_online_checkremove()
	}
	if flags&VdevOnlineUnspare != 0 {
		cFlags |= C.go_get_zfs_online_unspare()
	}
	if flags&VdevOnlineForceFault != 0 {
		cFlags |= C.go_get_zfs_online_forcefault()
	}
	if flags&VdevOnlineExpand != 0 {
		cFlags |= C.go_get_zfs_online_expand()
	}

	// Online the vdev
	var newState C.int
	ret := C.go_zpool_online(zhp, cDevice, cFlags, &newState)
	if ret != 0 {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return fmt.Errorf("failed to online vdev %s in pool %s (errno %d): %s", device, poolName, errno, desc)
	}

	return nil
}

func (d *libzfsDriver) OfflineVdev(ctx context.Context, poolName, device string, temporary bool) error {
	defer d.enter()()

	if d.h == nil {
		return fmt.Errorf("driver is closed")
	}

	// Open the pool
	cPoolName := C.CString(poolName)
	defer C.free(unsafe.Pointer(cPoolName))

	zhp := C.go_zpool_open(d.h, cPoolName)
	if zhp == nil {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return fmt.Errorf("failed to open pool %s (errno %d): %s", poolName, errno, desc)
	}
	defer C.go_zpool_close(zhp)

	// Convert device name to C string
	cDevice := C.CString(device)
	defer C.free(unsafe.Pointer(cDevice))

	var tempFlag C.int
	if temporary {
		tempFlag = 1
	}

	// Offline the vdev
	ret := C.go_zpool_offline(zhp, cDevice, tempFlag)
	if ret != 0 {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return fmt.Errorf("failed to offline vdev %s in pool %s (errno %d): %s", device, poolName, errno, desc)
	}

	return nil
}

func (d *libzfsDriver) ClearVdev(ctx context.Context, poolName, device string) error {
	defer d.enter()()

	if d.h == nil {
		return fmt.Errorf("driver is closed")
	}

	// Open the pool
	cPoolName := C.CString(poolName)
	defer C.free(unsafe.Pointer(cPoolName))

	zhp := C.go_zpool_open(d.h, cPoolName)
	if zhp == nil {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return fmt.Errorf("failed to open pool %s (errno %d): %s", poolName, errno, desc)
	}
	defer C.go_zpool_close(zhp)

	// Convert device name to C string
	cDevice := C.CString(device)
	defer C.free(unsafe.Pointer(cDevice))

	// Clear the vdev
	ret := C.go_zpool_clear(zhp, cDevice, nil)
	if ret != 0 {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return fmt.Errorf("failed to clear vdev %s in pool %s (errno %d): %s", device, poolName, errno, desc)
	}

	return nil
}

// Helper function to create complex vdev nvlist from specification
func (d *libzfsDriver) createComplexVdevNvlist(spec VdevSpec) (unsafe.Pointer, error) {
	switch spec.Type {
	case VdevTypeDisk:
		if len(spec.Devices) != 1 {
			return nil, fmt.Errorf("disk vdev requires exactly 1 device, got %d", len(spec.Devices))
		}
		cPath := C.CString(spec.Devices[0])
		defer C.free(unsafe.Pointer(cPath))
		cType := C.CString("disk")
		defer C.free(unsafe.Pointer(cType))
		return C.go_create_vdev_nvlist(cType, cPath), nil

	case VdevTypeMirror:
		if len(spec.Devices) < 2 {
			return nil, fmt.Errorf("mirror vdev requires at least 2 devices, got %d", len(spec.Devices))
		}
		return d.createMirrorVdev(spec.Devices)

	case VdevTypeRaidz:
		if len(spec.Devices) < 3 {
			return nil, fmt.Errorf("raidz vdev requires at least 3 devices, got %d", len(spec.Devices))
		}
		return d.createRaidzVdev(spec.Devices, 1)

	case VdevTypeRaidz2:
		if len(spec.Devices) < 4 {
			return nil, fmt.Errorf("raidz2 vdev requires at least 4 devices, got %d", len(spec.Devices))
		}
		return d.createRaidzVdev(spec.Devices, 2)

	case VdevTypeRaidz3:
		if len(spec.Devices) < 5 {
			return nil, fmt.Errorf("raidz3 vdev requires at least 5 devices, got %d", len(spec.Devices))
		}
		return d.createRaidzVdev(spec.Devices, 3)

	default:
		return nil, fmt.Errorf("unsupported vdev type: %s", spec.Type)
	}
}

func (d *libzfsDriver) createMirrorVdev(devices []string) (unsafe.Pointer, error) {
	// Create device nvlists
	deviceNvlists := make([]unsafe.Pointer, len(devices))
	for i, device := range devices {
		cDevice := C.CString(device)
		cDiskType := C.CString("disk")
		deviceNvlists[i] = C.go_create_vdev_nvlist(cDiskType, cDevice)
		C.free(unsafe.Pointer(cDevice))
		C.free(unsafe.Pointer(cDiskType))
		if deviceNvlists[i] == nil {
			// Clean up any previously created nvlists
			for j := 0; j < i; j++ {
				C.go_nvlist_free(deviceNvlists[j])
			}
			return nil, fmt.Errorf("failed to create nvlist for device %s", device)
		}
	}

	// Create mirror vdev
	mirror := C.go_create_mirror_vdev((*unsafe.Pointer)(unsafe.Pointer(&deviceNvlists[0])), C.uint(len(devices)))

	// Clean up device nvlists (they're copied into the mirror)
	for _, nvl := range deviceNvlists {
		C.go_nvlist_free(nvl)
	}

	return mirror, nil
}

func (d *libzfsDriver) createRaidzVdev(devices []string, parity int) (unsafe.Pointer, error) {
	// Create device nvlists
	deviceNvlists := make([]unsafe.Pointer, len(devices))
	for i, device := range devices {
		cDevice := C.CString(device)
		cDiskType := C.CString("disk")
		deviceNvlists[i] = C.go_create_vdev_nvlist(cDiskType, cDevice)
		C.free(unsafe.Pointer(cDevice))
		C.free(unsafe.Pointer(cDiskType))
		if deviceNvlists[i] == nil {
			// Clean up any previously created nvlists
			for j := 0; j < i; j++ {
				C.go_nvlist_free(deviceNvlists[j])
			}
			return nil, fmt.Errorf("failed to create nvlist for device %s", device)
		}
	}

	// Create raidz vdev
	raidz := C.go_create_raidz_vdev((*unsafe.Pointer)(unsafe.Pointer(&deviceNvlists[0])), C.uint(len(devices)), C.int(parity))

	// Clean up device nvlists (they're copied into the raidz)
	for _, nvl := range deviceNvlists {
		C.go_nvlist_free(nvl)
	}

	return raidz, nil
}

func (d *libzfsDriver) SupportsFeature(ctx context.Context, feature string) (bool, error) {
	defer d.enter()()

	if d.h == nil {
		return false, fmt.Errorf("driver is closed")
	}

	// All features are handled by the helper function
	return d.checkZFSFeature(feature)
}

// Helper function to check if a specific ZFS feature is supported
func (d *libzfsDriver) checkZFSFeature(feature string) (bool, error) {
	// This is a simplified implementation that checks if the feature exists
	// in the kernel module. A more sophisticated implementation would check
	// specific pool compatibility and version requirements.

	// For now, we'll assume most modern features are available
	// In a production environment, this should query the actual ZFS kernel
	// module capabilities or check against known version matrices

	commonFeatures := map[string]bool{
		// Core operations provided by the libzfs driver
		"pools":     true,
		"datasets":  true,
		"snapshots": true,
		"clones":    true,

		"bookmarks":             true,
		"encryption":            true,
		"large_blocks":          true,
		"spacemap_histogram":    true,
		"extensible_dataset":    true,
		"embedded_data":         true,
		"async_destroy":         true,
		"empty_bpobj":           true,
		"lz4_compress":          true,
		"multi_vdev_crash_dump": true,
		"spacemap_v2":           true,
		"enabled_txg":           true,
		"hole_birth":            true,
		"device_removal":        true,
		"obsolete_counts":       true,
		"zpool_checkpoint":      true,
		"allocation_classes":    true,
		"resilver_defer":        true,
		"bookmark_v2":           true,
		"redaction_bookmarks":   true,
		"redacted_datasets":     true,
		"bookmark_written":      true,
		"log_spacemap":          true,
		"livelist":              true,
		"redaction_list_spill":  true,
		"zstd_compress":         true,
		"draid":                 false, // DRAID might not be available on all systems
	}

	if supported, exists := commonFeatures[feature]; exists {
		return supported, nil
	}

	// For unknown features, return false but no error
	return false, nil
}

func (d *libzfsDriver) GetAvailableFeatures(ctx context.Context) ([]string, error) {
	defer d.enter()()

	if d.h == nil {
		return nil, fmt.Errorf("driver is closed")
	}

	// Return a list of commonly available ZFS features
	// In a production environment, this should query the actual kernel module
	features := []string{
		"async_destroy",
		"empty_bpobj",
		"lz4_compress",
		"multi_vdev_crash_dump",
		"spacemap_v2",
		"enabled_txg",
		"hole_birth",
		"extensible_dataset",
		"embedded_data",
		"bookmarks",
		"filesystem_limits",
		"large_blocks",
		"large_dnode",
		"sha512",
		"skein",
		"edonr",
		"device_removal",
		"obsolete_counts",
		"zpool_checkpoint",
		"spacemap_histogram",
		"allocation_classes",
		"resilver_defer",
		"bookmark_v2",
		"redaction_bookmarks",
		"redacted_datasets",
		"bookmark_written",
		"log_spacemap",
		"livelist",
		"redaction_list_spill",
		"zstd_compress",
		"encryption",
	}

	return features, nil
}

func (d *libzfsDriver) GetSupportedCompressionAlgorithms(ctx context.Context) ([]string, error) {
	defer d.enter()()

	if d.h == nil {
		return nil, fmt.Errorf("driver is closed")
	}

	// Return commonly supported compression algorithms
	// In a production environment, this should check what's actually available
	algorithms := []string{
		"off",
		"on",
		"lzjb",
		"gzip",
		"gzip-1",
		"gzip-2",
		"gzip-3",
		"gzip-4",
		"gzip-5",
		"gzip-6",
		"gzip-7",
		"gzip-8",
		"gzip-9",
		"zle",
		"lz4",
	}

	// Check if ZSTD is supported
	if supported, _ := d.checkZFSFeature("zstd_compress"); supported {
		algorithms = append(algorithms, "zstd", "zstd-1", "zstd-2", "zstd-3", "zstd-4", "zstd-5",
			"zstd-6", "zstd-7", "zstd-8", "zstd-9", "zstd-10", "zstd-11", "zstd-12", "zstd-13",
			"zstd-14", "zstd-15", "zstd-16", "zstd-17", "zstd-18", "zstd-19", "zstd-fast-1",
			"zstd-fast-2", "zstd-fast-3", "zstd-fast-4", "zstd-fast-5", "zstd-fast-6",
			"zstd-fast-7", "zstd-fast-8", "zstd-fast-9", "zstd-fast-10")
	}

	return algorithms, nil
}

func (d *libzfsDriver) GetZFSVersion(ctx context.Context) (string, error) {
	defer d.enter()()

	if d.h == nil {
		return "", fmt.Errorf("driver is closed")
	}

	// For now, return a default version
	// In a production environment, this should query the actual ZFS version
	// from sysctl or libzfs functions
	return "OpenZFS 2.1+", nil
}
