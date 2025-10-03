//go:build freebsd && !no_libzfs

package driver

/*
#cgo CFLAGS: -I/usr/src/sys/contrib/openzfs/lib/libspl/include -I/usr/src/sys/contrib/openzfs/lib/libspl/include/os/freebsd -I/usr/src/sys/contrib/openzfs/include
#cgo LDFLAGS: -lzfs -lnvpair
#include <libzfs.h>
#include <sys/nvpair.h>
#include <stdlib.h>

// Forward declarations for our C helper functions
extern int go_libzfs_errno(libzfs_handle_t* hdl);
extern const char* go_libzfs_error_description(libzfs_handle_t* hdl);

// Dataset type constants
extern int go_get_zfs_type_filesystem();
extern int go_get_zfs_type_volume();
extern int go_get_zfs_type_snapshot();

// Nvlist operations
extern void* go_nvlist_alloc();
extern void go_nvlist_free(void* nvl);
extern int go_nvlist_add_string(void* nvl, char* name, char* val);
extern int go_nvlist_add_nvlist(void* nvl, char* name, void* val);
extern int go_nvlist_add_nvlist_array(void* nvl, char* name, void** val, unsigned int nelem);
extern void* go_create_vdev_nvlist(char* type, char* path);
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// Helper function to convert C strings safely
func safeGoString(cstr *C.char) string {
	if cstr == nil {
		return ""
	}
	return C.GoString(cstr)
}

// Helper function to convert integers safely
func safeGoUint64(val C.uint64_t) uint64 {
	return uint64(val)
}

// Helper function to convert Go bool to C boolean
func btoc(b bool) C.int {
	if b {
		return 1
	}
	return 0
}

// Helper function to convert Go properties map to nvlist
func (d *libzfsDriver) createPropsNvlist(props map[string]string) (unsafe.Pointer, error) {
	if len(props) == 0 {
		return nil, nil
	}

	nvl := C.go_nvlist_alloc()
	if nvl == nil {
		return nil, fmt.Errorf("failed to allocate nvlist")
	}

	for key, value := range props {
		cKey := C.CString(key)
		cValue := C.CString(value)

		ret := C.go_nvlist_add_string(nvl, cKey, cValue)

		C.free(unsafe.Pointer(cKey))
		C.free(unsafe.Pointer(cValue))

		if ret != 0 {
			C.go_nvlist_free(nvl)
			return nil, fmt.Errorf("failed to add property %s=%s to nvlist", key, value)
		}
	}

	return nvl, nil
}

// Helper function to free nvlist safely
func (d *libzfsDriver) freeNvlist(nvl unsafe.Pointer) {
	if nvl != nil {
		C.go_nvlist_free(nvl)
	}
}

// Helper function to create vdev nvlist for pool creation
func (d *libzfsDriver) createVdevNvlist(vdevType, path string) (unsafe.Pointer, error) {
	cVdevType := C.CString(vdevType)
	cVdevPath := C.CString(path)
	defer C.free(unsafe.Pointer(cVdevType))
	defer C.free(unsafe.Pointer(cVdevPath))

	vdev := C.go_create_vdev_nvlist(cVdevType, cVdevPath)
	if vdev == nil {
		return nil, fmt.Errorf("failed to create vdev nvlist for %s:%s", vdevType, path)
	}

	return vdev, nil
}

// Helper function to create vdev root nvlist with children
func (d *libzfsDriver) createVdevRoot(vdevs []string) (unsafe.Pointer, []unsafe.Pointer, error) {
	// Create vdev root nvlist
	nvroot := C.go_nvlist_alloc()
	if nvroot == nil {
		return nil, nil, fmt.Errorf("failed to allocate vdev root nvlist")
	}

	// Set root vdev type
	cType := C.CString("type")
	cRoot := C.CString(VdevTypeRoot)
	defer C.free(unsafe.Pointer(cType))
	defer C.free(unsafe.Pointer(cRoot))

	if C.go_nvlist_add_string(nvroot, cType, cRoot) != 0 {
		d.freeNvlist(nvroot)
		return nil, nil, fmt.Errorf("failed to set root vdev type")
	}

	// Create child vdevs array
	vdevList := make([]unsafe.Pointer, len(vdevs))
	for i, vdev := range vdevs {
		vdevNvlist, err := d.createVdevNvlist(VdevTypeDisk, vdev)
		if err != nil {
			// Clean up previously created vdevs
			for j := 0; j < i; j++ {
				d.freeNvlist(vdevList[j])
			}
			d.freeNvlist(nvroot)
			return nil, nil, err
		}
		vdevList[i] = vdevNvlist
	}

	// Add vdevs to root as children
	cChildren := C.CString("children")
	defer C.free(unsafe.Pointer(cChildren))

	ret := C.go_nvlist_add_nvlist_array(nvroot, cChildren, (*unsafe.Pointer)(unsafe.Pointer(&vdevList[0])), C.uint(len(vdevList)))
	if ret != 0 {
		// Clean up all nvlists
		for _, vdev := range vdevList {
			d.freeNvlist(vdev)
		}
		d.freeNvlist(nvroot)
		return nil, nil, fmt.Errorf("failed to add children to vdev root")
	}

	return nvroot, vdevList, nil
}

// Helper function to map driver pool state to string
func mapPoolState(state int) string {
	switch state {
	case PoolStateActive:
		return "ACTIVE"
	case PoolStateExported:
		return "EXPORTED"
	case PoolStateDestroyed:
		return "DESTROYED"
	case PoolStateSpare:
		return "SPARE"
	case PoolStateL2Cache:
		return "L2CACHE"
	case PoolStateUninitialized:
		return "UNINITIALIZED"
	case PoolStateUnavail:
		return "UNAVAIL"
	case PoolStatePotentiallyActive:
		return "POTENTIALLY_ACTIVE"
	default:
		return "UNKNOWN"
	}
}

// Helper function to map driver pool health to string
func mapPoolHealth(status int) string {
	switch status {
	case PoolStatusCorruptCache:
		return "DEGRADED"
	case PoolStatusMissingDevR:
		return "DEGRADED"
	case PoolStatusMissingDevNr:
		return "DEGRADED"
	case PoolStatusCorruptLabelR:
		return "DEGRADED"
	case PoolStatusCorruptLabelNr:
		return "DEGRADED"
	default:
		return "ONLINE" // Default to ONLINE for unknown status
	}
}

// Helper function to safely open a dataset handle with type detection
func (d *libzfsDriver) openDatasetHandle(datasetName string) (*C.zfs_handle_t, error) {
	cDatasetName := C.CString(datasetName)
	defer C.free(unsafe.Pointer(cDatasetName))

	// Try to open as filesystem first, then volume
	zhp := C.zfs_open(d.h, cDatasetName, C.int(C.go_get_zfs_type_filesystem()))
	if zhp == nil {
		zhp = C.zfs_open(d.h, cDatasetName, C.int(C.go_get_zfs_type_volume()))
		if zhp == nil {
			errno := C.go_libzfs_errno(d.h)
			desc := C.GoString(C.go_libzfs_error_description(d.h))
			return nil, fmt.Errorf("failed to open dataset %s (errno %d): %s", datasetName, errno, desc)
		}
	}

	return zhp, nil
}

// Helper function to safely open a pool handle
func (d *libzfsDriver) openPoolHandle(poolName string) (*C.zpool_handle_t, error) {
	cPoolName := C.CString(poolName)
	defer C.free(unsafe.Pointer(cPoolName))

	zhp := C.zpool_open(d.h, cPoolName)
	if zhp == nil {
		errno := C.go_libzfs_errno(d.h)
		desc := C.GoString(C.go_libzfs_error_description(d.h))
		return nil, fmt.Errorf("failed to open pool %s (errno %d): %s", poolName, errno, desc)
	}

	return zhp, nil
}

// Helper function to get libzfs error information
func (d *libzfsDriver) getLibzfsError() (int, string) {
	errno := C.go_libzfs_errno(d.h)
	desc := C.GoString(C.go_libzfs_error_description(d.h))
	return int(errno), desc
}
