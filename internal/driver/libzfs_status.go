//go:build freebsd && !no_libzfs

package driver

/*
#cgo CFLAGS: -I/usr/src/sys/contrib/openzfs/lib/libspl/include -I/usr/src/sys/contrib/openzfs/lib/libspl/include/os/freebsd -I/usr/src/sys/contrib/openzfs/include
#cgo LDFLAGS: -lzfs -lnvpair
#include <libzfs.h>
#include <stdlib.h>

// Implemented in internal/cgo.
extern const char *go_zpool_status_name(int status);
extern nvlist_t *go_config_vdev_tree(nvlist_t *config);
extern uint64_t go_pool_errcount(nvlist_t *config);
extern int go_vdev_group(nvlist_t *nv, int group, nvlist_t ***child, uint_t *n);
extern void go_vdev_get_info(nvlist_t *nv, int is_spare, const char **type,
    const char **path, const char **state, uint64_t *nums);
extern int go_pool_scan_stats(nvlist_t *nvroot, pool_scan_stat_t *out);
extern int go_zpool_get_status(zpool_handle_t* zhp, char** msgid);
extern int go_zpool_scan(zpool_handle_t* zhp, int func, int cmd);
*/
import "C"

import (
	"context"
	"fmt"
	"unsafe"
)

// vdev child groups understood by go_vdev_group
const (
	vdevGroupChildren = 0
	vdevGroupSpares   = 1
	vdevGroupL2Cache  = 2
)

func (d *libzfsDriver) GetPoolStatus(ctx context.Context, poolName string) (*PoolStatus, error) {
	defer d.enter()()

	if d.h == nil {
		return nil, fmt.Errorf("driver closed")
	}

	zhp, err := d.openPoolHandle(poolName)
	if err != nil {
		return nil, err
	}
	defer C.zpool_close(zhp)

	var missing C.boolean_t
	if C.zpool_refresh_stats(zhp, &missing) != 0 {
		errno, desc := d.getLibzfsError()
		return nil, fmt.Errorf("failed to refresh status of pool %s (errno %d): %s", poolName, errno, desc)
	}

	config := C.zpool_get_config(zhp, nil)
	if config == nil {
		return nil, fmt.Errorf("pool %s has no configuration", poolName)
	}
	nvroot := C.go_config_vdev_tree(config)
	if nvroot == nil {
		return nil, fmt.Errorf("pool %s has no vdev tree", poolName)
	}

	st := &PoolStatus{
		Root:       d.vdevStatus(zhp, nvroot, false),
		Spares:     d.vdevGroup(zhp, nvroot, vdevGroupSpares),
		Cache:      d.vdevGroup(zhp, nvroot, vdevGroupL2Cache),
		DataErrors: uint64(C.go_pool_errcount(config)),
	}
	st.Root.Name = poolName

	var pss C.pool_scan_stat_t
	if C.go_pool_scan_stats(nvroot, &pss) == 0 {
		st.Scan = &ScanStats{
			Function:        uint64(pss.pss_func),
			State:           uint64(pss.pss_state),
			StartTime:       uint64(pss.pss_start_time),
			EndTime:         uint64(pss.pss_end_time),
			ToExamine:       uint64(pss.pss_to_examine),
			Examined:        uint64(pss.pss_examined),
			Skipped:         uint64(pss.pss_skipped),
			Processed:       uint64(pss.pss_processed),
			Errors:          uint64(pss.pss_errors),
			PassExamined:    uint64(pss.pss_pass_exam),
			PassStart:       uint64(pss.pss_pass_start),
			PassPausedAt:    uint64(pss.pss_pass_scrub_pause),
			PassSpentPaused: uint64(pss.pss_pass_scrub_spent_paused),
			PassIssued:      uint64(pss.pss_pass_issued),
			Issued:          uint64(pss.pss_issued),
		}
	}

	var msgid *C.char
	code := C.go_zpool_get_status(zhp, &msgid)
	st.Problem = C.GoString(C.go_zpool_status_name(code))
	if msgid != nil {
		st.MsgID = C.GoString(msgid)
	}

	return st, nil
}

// vdevGroup converts the spares or cache devices of the root vdev.
func (d *libzfsDriver) vdevGroup(zhp *C.zpool_handle_t, nvroot *C.nvlist_t, group int) []VdevStatus {
	var child **C.nvlist_t
	var n C.uint_t
	if C.go_vdev_group(nvroot, C.int(group), &child, &n) == 0 {
		return nil
	}
	out := make([]VdevStatus, 0, int(n))
	for _, nv := range unsafe.Slice(child, int(n)) {
		out = append(out, d.vdevStatus(zhp, nv, group == vdevGroupSpares))
	}
	return out
}

// vdevStatus converts a vdev nvlist and its regular children recursively.
func (d *libzfsDriver) vdevStatus(zhp *C.zpool_handle_t, nv *C.nvlist_t, spare bool) VdevStatus {
	var typ, path, state *C.char
	var nums [7]C.uint64_t
	isSpare := C.int(0)
	if spare {
		isSpare = 1
	}
	C.go_vdev_get_info(nv, isSpare, &typ, &path, &state, &nums[0])

	v := VdevStatus{
		Type:           C.GoString(typ),
		Path:           C.GoString(path),
		State:          C.GoString(state),
		GUID:           uint64(nums[0]),
		Alloc:          uint64(nums[1]),
		Space:          uint64(nums[2]),
		DSpace:         uint64(nums[3]),
		ReadErrors:     uint64(nums[4]),
		WriteErrors:    uint64(nums[5]),
		ChecksumErrors: uint64(nums[6]),
	}
	// VDEV_NAME_TYPE_ID gives "mirror-0" style names, as zpool status prints them.
	if name := C.zpool_vdev_name(d.h, zhp, nv, C.VDEV_NAME_TYPE_ID); name != nil {
		v.Name = C.GoString(name)
		C.free(unsafe.Pointer(name))
	}

	var child **C.nvlist_t
	var n C.uint_t
	if C.go_vdev_group(nv, vdevGroupChildren, &child, &n) > 0 {
		for _, c := range unsafe.Slice(child, int(n)) {
			v.Children = append(v.Children, d.vdevStatus(zhp, c, false))
		}
	}
	return v
}

// ScrubPool starts (or resumes), pauses or stops a scrub, like `zpool scrub`,
// `zpool scrub -p` and `zpool scrub -s`.
func (d *libzfsDriver) ScrubPool(ctx context.Context, poolName string, cmd ScrubCommand) error {
	defer d.enter()()

	if d.h == nil {
		return fmt.Errorf("driver closed")
	}

	fn, scrubCmd := C.int(C.POOL_SCAN_SCRUB), C.int(C.POOL_SCRUB_NORMAL)
	action := "start scrub on"
	switch cmd {
	case ScrubStart:
	case ScrubPause:
		scrubCmd = C.POOL_SCRUB_PAUSE
		action = "pause scrub on"
	case ScrubStop:
		fn = C.POOL_SCAN_NONE
		action = "stop scrub on"
	default:
		return fmt.Errorf("unknown scrub command %d", cmd)
	}

	zhp, err := d.openPoolHandle(poolName)
	if err != nil {
		return err
	}
	defer C.zpool_close(zhp)

	if C.go_zpool_scan(zhp, fn, scrubCmd) != 0 {
		errno, desc := d.getLibzfsError()
		return fmt.Errorf("failed to %s pool %s (errno %d): %s", action, poolName, errno, desc)
	}
	return nil
}
