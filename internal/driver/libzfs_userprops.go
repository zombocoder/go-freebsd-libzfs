//go:build freebsd && !no_libzfs

package driver

/*
#cgo CFLAGS: -I/usr/src/sys/contrib/openzfs/lib/libspl/include -I/usr/src/sys/contrib/openzfs/lib/libspl/include/os/freebsd -I/usr/src/sys/contrib/openzfs/include
#cgo LDFLAGS: -lzfs -lnvpair
#include <libzfs.h>
#include <stdlib.h>
#include <stdint.h>

extern int go_userprops_iter_callback(zfs_handle_t *, void *);
extern void go_iter_given(void);
extern void go_iter_close_zfs(zfs_handle_t* zhp);

// userprops_walk iterates the filesystems and volumes below zhp; the handle
// is a runtime/cgo.Handle stored in C memory so no Go pointer crosses into C.
static int userprops_walk(zfs_handle_t *zhp, uintptr_t handle) {
	uintptr_t *box = malloc(sizeof(*box));
	if (box == NULL)
		return -1;
	*box = handle;
	int ret = zfs_iter_filesystems(zhp, go_userprops_iter_callback, box);
	free(box);
	return ret;
}

static int userprops_walk_box(zfs_handle_t *zhp, void *box) {
	return zfs_iter_filesystems(zhp, go_userprops_iter_callback, box);
}
*/
import "C"

import (
	"context"
	"fmt"
	"runtime/cgo"
	"strings"
	"unsafe"

	"github.com/zombocoder/go-freebsd-libzfs/errors"
)

// recvdSuffix marks the source of a received user property
// (ZPROP_SOURCE_VAL_RECVD).
const recvdSuffix = "$recvd"

// userPropSource maps the "source" string libzfs stores with a user property
// to a PropSource. libzfs stores the name of the dataset the value is set on;
// a value set on the dataset itself is local, on an ancestor inherited.
func userPropSource(source, dataset string) PropSource {
	switch {
	case source == "":
		return PropSourceNone
	case strings.HasSuffix(source, recvdSuffix):
		return PropSourceReceived
	case source == dataset:
		return PropSourceLocal
	default:
		return PropSourceInherited
	}
}

// readUserProps collects the user properties of an open dataset handle.
func readUserProps(zhp *C.zfs_handle_t) map[string]UserPropertyInfo {
	result := make(map[string]UserPropertyInfo)
	dataset := C.GoString(C.zfs_get_name(zhp))
	nv := C.zfs_get_user_props(zhp)
	if nv == nil {
		return result
	}

	cValue := C.CString("value")
	defer C.free(unsafe.Pointer(cValue))
	cSource := C.CString("source")
	defer C.free(unsafe.Pointer(cSource))

	for elem := C.nvlist_next_nvpair(nv, nil); elem != nil; elem = C.nvlist_next_nvpair(nv, elem) {
		name := C.GoString(C.nvpair_name(elem))
		if !strings.Contains(name, ":") {
			continue
		}
		var propNv *C.nvlist_t
		if C.nvpair_value_nvlist(elem, &propNv) != 0 {
			continue
		}
		var value, source *C.char
		if C.nvlist_lookup_string(propNv, cValue, &value) != 0 {
			continue
		}
		info := UserPropertyInfo{Value: C.GoString(value), Source: PropSourceNone}
		if C.nvlist_lookup_string(propNv, cSource, &source) == 0 {
			info.Source = userPropSource(C.GoString(source), dataset)
		}
		result[name] = info
	}
	return result
}

// GetUserProperties returns all user properties of a dataset, including
// values inherited from ancestors.
func (d *libzfsDriver) GetUserProperties(ctx context.Context, datasetName string) (map[string]UserPropertyInfo, error) {
	defer d.enter()()

	if d.h == nil {
		return nil, fmt.Errorf("driver closed")
	}

	zhp, err := d.openDatasetHandle(datasetName)
	if err != nil {
		return nil, err
	}
	defer C.zfs_close(zhp)

	return readUserProps(zhp), nil
}

type userPropsWalk struct {
	result []ChildUserProps
}

//export go_userprops_iter_callback
func go_userprops_iter_callback(zhp *C.zfs_handle_t, data unsafe.Pointer) C.int {
	w := cgo.Handle(*(*C.uintptr_t)(data)).Value().(*userPropsWalk)

	// zfs_iter_filesystems hands the handle to the callback, which must close
	// it (as zfs list does) once the nested walk is done.
	C.go_iter_given()
	defer C.go_iter_close_zfs(zhp)

	t := mapDatasetType(int(C.zfs_get_type(zhp)))
	if t == DatasetFilesystem || t == DatasetVolume {
		w.result = append(w.result, ChildUserProps{
			Name:  C.GoString(C.zfs_get_name(zhp)),
			Type:  t,
			Props: readUserProps(zhp),
		})
	}
	return C.userprops_walk_box(zhp, data)
}

// ListDescendantUserProps walks the filesystems and volumes below parent and
// reads each one's user properties from its own handle, under one lock.
func (d *libzfsDriver) ListDescendantUserProps(ctx context.Context, parent string) ([]ChildUserProps, error) {
	defer d.enter()()

	if d.h == nil {
		return nil, fmt.Errorf("driver closed")
	}

	cName := C.CString(parent)
	defer C.free(unsafe.Pointer(cName))
	zhp := C.zfs_open(d.h, cName, C.ZFS_TYPE_FILESYSTEM|C.ZFS_TYPE_VOLUME)
	if zhp == nil {
		errno := C.libzfs_errno(d.h)
		desc := C.GoString(C.libzfs_error_description(d.h))
		if errno == C.EZFS_NOENT {
			return nil, fmt.Errorf("dataset %s: %s: %w", parent, desc, errors.ErrDatasetNotFound)
		}
		return nil, fmt.Errorf("failed to open dataset %s (errno %d): %s", parent, errno, desc)
	}
	defer C.zfs_close(zhp)

	errBefore := C.libzfs_errno(d.h)
	for attempt := 1; ; attempt++ {
		attemptBefore := C.libzfs_errno(d.h)
		w := &userPropsWalk{}
		h := cgo.NewHandle(w)
		ret := C.userprops_walk(zhp, C.uintptr_t(h))
		h.Delete()
		if ret == 0 {
			return w.result, nil
		}
		if attempt >= maxIterAttempts || !interrupted(d.h, attemptBefore) {
			return nil, iterError("failed to iterate datasets below "+parent, d.h, errBefore)
		}
	}
}
