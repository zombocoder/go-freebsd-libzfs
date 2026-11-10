//go:build freebsd && !no_libzfs

package cgo

/*
#cgo CFLAGS: -I/usr/src/sys/contrib/openzfs/lib/libspl/include -I/usr/src/sys/contrib/openzfs/lib/libspl/include/os/freebsd -I/usr/src/sys/contrib/openzfs/include
#cgo LDFLAGS: -lzfs -lnvpair -lzutil
#include <libzfs.h>
#include <libzutil.h>
#include <sys/nvpair.h>
#include <stdlib.h>
#include <string.h>
#include <signal.h>
#include <pthread.h>

// libzfs handle management
libzfs_handle_t* go_libzfs_init(void) {
    return libzfs_init();
}

void go_libzfs_fini(libzfs_handle_t* hdl) {
    if (hdl != NULL) {
        libzfs_fini(hdl);
    }
}

int go_libzfs_errno(libzfs_handle_t* hdl) {
    return libzfs_errno(hdl);
}

const char* go_libzfs_error_description(libzfs_handle_t* hdl) {
    return libzfs_error_description(hdl);
}

// Pool handle accessors
const char* go_zpool_get_name(zpool_handle_t* zhp) {
    return zpool_get_name(zhp);
}

uint64_t go_zpool_get_guid(zpool_handle_t* zhp) {
    return zpool_get_prop_int(zhp, ZPOOL_PROP_GUID, NULL);
}

// Dataset handle accessors
const char* go_zfs_get_name(zfs_handle_t* zhp) {
    return zfs_get_name(zhp);
}

uint64_t go_zfs_get_guid(zfs_handle_t* zhp) {
    return zfs_prop_get_int(zhp, ZFS_PROP_GUID);
}

int go_zfs_get_type(zfs_handle_t* zhp) {
    return zfs_get_type(zhp);
}

// Signal masking. libzfs does not retry ioctls that fail with EINTR, and the
// Go runtime signals threads (SIGURG for preemption, SIGPROF, ...) at any
// time, including while a thread is inside libzfs. go_sigs_block blocks every
// asynchronous signal on the calling thread and go_sigs_restore puts the
// previous mask back. The caller must stay on the same OS thread in between
// (runtime.LockOSThread). Synchronous fault signals stay unblocked so a
// crash is still reported by the runtime instead of killing the process.
static __thread sigset_t go_saved_sigmask;

void go_sigs_block(void) {
    sigset_t all;
    sigfillset(&all);
    sigdelset(&all, SIGSEGV);
    sigdelset(&all, SIGBUS);
    sigdelset(&all, SIGFPE);
    sigdelset(&all, SIGILL);
    sigdelset(&all, SIGTRAP);
    sigdelset(&all, SIGABRT);
    sigdelset(&all, SIGSYS);
    pthread_sigmask(SIG_BLOCK, &all, &go_saved_sigmask);
}

void go_sigs_restore(void) {
    pthread_sigmask(SIG_SETMASK, &go_saved_sigmask, NULL);
}

// Iteration handle accounting. libzfs hands every handle passed to an
// iteration callback over to the callback, which must close it (zpool_iter,
// zfs_iter_root, zfs_iter_children, zfs_iter_filesystems, zfs_iter_snapshots).
// go_iter_given records a handle received, go_iter_close_* closes it and
// records that; go_iter_outstanding is the number not closed yet (0 when
// nothing leaked).
static long iter_handles_out = 0;

void go_iter_given(void) {
    __atomic_add_fetch(&iter_handles_out, 1, __ATOMIC_SEQ_CST);
}

void go_iter_close_zfs(zfs_handle_t* zhp) {
    zfs_close(zhp);
    __atomic_sub_fetch(&iter_handles_out, 1, __ATOMIC_SEQ_CST);
}

void go_iter_close_zpool(zpool_handle_t* zhp) {
    zpool_close(zhp);
    __atomic_sub_fetch(&iter_handles_out, 1, __ATOMIC_SEQ_CST);
}

long go_iter_outstanding(void) {
    return __atomic_load_n(&iter_handles_out, __ATOMIC_SEQ_CST);
}

// Pool iteration helpers. The callback owns (and must close) the handle.
int go_iter_pools(libzfs_handle_t* hdl, int (*func)(zpool_handle_t *, void *), void* data) {
    return zpool_iter(hdl, func, data);
}

// Context for iteration with a Go callback
typedef struct {
    int (*callback)(zfs_handle_t *, void *);
    void* user_data;
} comprehensive_iter_ctx_t;

// Calls the callback, then closes the handle: for iterations where the
// callback does not need the handle afterwards.
static int closing_iter_func(zfs_handle_t* zhp, void* ctx_ptr) {
    comprehensive_iter_ctx_t* ctx = (comprehensive_iter_ctx_t*)ctx_ptr;
    go_iter_given();
    int ret = ctx->callback(zhp, ctx->user_data);
    go_iter_close_zfs(zhp);
    return ret;
}

// Dataset iteration helpers - only iterate datasets, not snapshots.
// The callback only reads the handle; it is closed here.
int go_iter_datasets(libzfs_handle_t* hdl, int (*func)(zfs_handle_t *, void *), void* data) {
    comprehensive_iter_ctx_t ctx = { .callback = func, .user_data = data };
    return zfs_iter_root(hdl, closing_iter_func, &ctx);
}

// Forward declaration
static int comprehensive_iter_func(zfs_handle_t* zhp, void* ctx_ptr);

// Comprehensive iteration that includes all datasets and snapshots
int go_iter_all_datasets(libzfs_handle_t* hdl, int (*func)(zfs_handle_t *, void *), void* data) {
    comprehensive_iter_ctx_t ctx = { .callback = func, .user_data = data };
    return zfs_iter_root(hdl, comprehensive_iter_func, &ctx);
}

// Recursive function that iterates datasets, children, and snapshots. It owns
// zhp and closes it on every path.
static int comprehensive_iter_func(zfs_handle_t* zhp, void* ctx_ptr) {
    comprehensive_iter_ctx_t* ctx = (comprehensive_iter_ctx_t*)ctx_ptr;
    int ret;
    int zfs_type = zfs_get_type(zhp);

    go_iter_given();

    // Process this dataset first
    ret = ctx->callback(zhp, ctx->user_data);

    // Only process children and snapshots for non-snapshot datasets
    if (ret == 0 && zfs_type != ZFS_TYPE_SNAPSHOT) {
        // Process all child datasets (filesystems and volumes) recursively
        ret = zfs_iter_children(zhp, comprehensive_iter_func, ctx);

        // Process snapshots of this dataset
        if (ret == 0) {
            ret = zfs_iter_snapshots(zhp, B_FALSE, closing_iter_func, ctx, 0, 0);
        }
    }

    go_iter_close_zfs(zhp);
    return ret;
}

// Pool state and health helpers
int go_zpool_get_state(zpool_handle_t* zhp) {
    return zpool_get_state(zhp);
}

zpool_status_t go_zpool_get_status(zpool_handle_t* zhp, char** msgid) {
    const char* msg = NULL;
    zpool_status_t result = zpool_get_status(zhp, &msg, NULL);
    if (msgid && msg) {
        *msgid = (char*)msg;
    }
    return result;
}

// Property retrieval helpers
int go_zpool_get_prop(zpool_handle_t* zhp, zpool_prop_t prop, char* buf, size_t len,
                      zprop_source_t* src, boolean_t literal) {
    return zpool_get_prop(zhp, prop, buf, len, src, literal);
}

int go_zfs_get_prop(zfs_handle_t* zhp, zfs_prop_t prop, char* buf, size_t len,
                    zprop_source_t* src, char* statbuf, size_t statlen, boolean_t literal) {
    return zfs_prop_get(zhp, prop, buf, len, src, statbuf, statlen, literal);
}

// Property constants helper functions
int go_get_zpool_prop_size() { return ZPOOL_PROP_SIZE; }
int go_get_zpool_prop_capacity() { return ZPOOL_PROP_CAPACITY; }
int go_get_zpool_prop_health() { return ZPOOL_PROP_HEALTH; }
int go_get_zpool_prop_guid() { return ZPOOL_PROP_GUID; }
int go_get_zpool_prop_version() { return ZPOOL_PROP_VERSION; }
int go_get_zpool_prop_free() { return ZPOOL_PROP_FREE; }
int go_get_zpool_prop_allocated() { return ZPOOL_PROP_ALLOCATED; }

int go_get_zfs_prop_type() { return ZFS_PROP_TYPE; }
int go_get_zfs_prop_creation() { return ZFS_PROP_CREATION; }
int go_get_zfs_prop_used() { return ZFS_PROP_USED; }
int go_get_zfs_prop_available() { return ZFS_PROP_AVAILABLE; }
int go_get_zfs_prop_referenced() { return ZFS_PROP_REFERENCED; }
int go_get_zfs_prop_compressratio() { return ZFS_PROP_COMPRESSRATIO; }
int go_get_zfs_prop_mounted() { return ZFS_PROP_MOUNTED; }
int go_get_zfs_prop_quota() { return ZFS_PROP_QUOTA; }
int go_get_zfs_prop_reservation() { return ZFS_PROP_RESERVATION; }
int go_get_zfs_prop_recordsize() { return ZFS_PROP_RECORDSIZE; }
int go_get_zfs_prop_mountpoint() { return ZFS_PROP_MOUNTPOINT; }
int go_get_zfs_prop_compression() { return ZFS_PROP_COMPRESSION; }

// Property retrieval with proper error handling
// src_out receives the zprop_source_t of the value (ZPROP_SRC_*).
int go_get_zpool_property(zpool_handle_t* zhp, int prop, char* buf, size_t len, int* src_out) {
    zprop_source_t src = ZPROP_SRC_NONE;
    int ret = zpool_get_prop(zhp, prop, buf, len, &src, B_FALSE);
    *src_out = (int)src;
    return ret;
}

int go_get_zfs_property(zfs_handle_t* zhp, int prop, char* buf, size_t len, int* src_out) {
    zprop_source_t src = ZPROP_SRC_NONE;
    char statbuf[256] = {0};
    int ret = zfs_prop_get(zhp, prop, buf, len, &src, statbuf, sizeof(statbuf), B_FALSE);
    *src_out = (int)src;
    return ret;
}

// Pool configuration and status helpers
nvlist_t* go_zpool_get_config(zpool_handle_t* zhp, nvlist_t** oldconfig) {
    return zpool_get_config(zhp, oldconfig);
}

int go_zpool_scan(zpool_handle_t* zhp, pool_scan_func_t func, pool_scrub_cmd_t cmd) {
    return zpool_scan(zhp, func, cmd);
}

// Vdev tree navigation helpers
int go_nvlist_lookup_nvlist(nvlist_t* nvl, const char* name, nvlist_t** val) {
    return nvlist_lookup_nvlist(nvl, name, val);
}

int go_nvlist_lookup_nvlist_array(nvlist_t* nvl, const char* name, nvlist_t*** val, uint_t* nelem) {
    return nvlist_lookup_nvlist_array(nvl, name, val, nelem);
}

int go_nvlist_lookup_string(nvlist_t* nvl, const char* name, char** val) {
    const char* temp_val;
    int ret = nvlist_lookup_string(nvl, name, &temp_val);
    if (ret == 0) {
        *val = (char*)temp_val;
    }
    return ret;
}

int go_nvlist_lookup_uint64(nvlist_t* nvl, const char* name, uint64_t* val) {
    return nvlist_lookup_uint64(nvl, name, val);
}

int go_nvlist_lookup_uint64_array(nvlist_t* nvl, const char* name, uint64_t** val, uint_t* nelem) {
    return nvlist_lookup_uint64_array(nvl, name, val, nelem);
}

// ---- Pool status helpers ----

// go_zpool_status_name returns the name of a zpool_status_t without the
// ZPOOL_STATUS_ prefix.
const char *go_zpool_status_name(int status) {
    switch ((zpool_status_t)status) {
    case ZPOOL_STATUS_CORRUPT_CACHE: return "CORRUPT_CACHE";
    case ZPOOL_STATUS_MISSING_DEV_R: return "MISSING_DEV_R";
    case ZPOOL_STATUS_MISSING_DEV_NR: return "MISSING_DEV_NR";
    case ZPOOL_STATUS_CORRUPT_LABEL_R: return "CORRUPT_LABEL_R";
    case ZPOOL_STATUS_CORRUPT_LABEL_NR: return "CORRUPT_LABEL_NR";
    case ZPOOL_STATUS_BAD_GUID_SUM: return "BAD_GUID_SUM";
    case ZPOOL_STATUS_CORRUPT_POOL: return "CORRUPT_POOL";
    case ZPOOL_STATUS_CORRUPT_DATA: return "CORRUPT_DATA";
    case ZPOOL_STATUS_FAILING_DEV: return "FAILING_DEV";
    case ZPOOL_STATUS_VERSION_NEWER: return "VERSION_NEWER";
    case ZPOOL_STATUS_HOSTID_MISMATCH: return "HOSTID_MISMATCH";
    case ZPOOL_STATUS_HOSTID_ACTIVE: return "HOSTID_ACTIVE";
    case ZPOOL_STATUS_HOSTID_REQUIRED: return "HOSTID_REQUIRED";
    case ZPOOL_STATUS_IO_FAILURE_WAIT: return "IO_FAILURE_WAIT";
    case ZPOOL_STATUS_IO_FAILURE_CONTINUE: return "IO_FAILURE_CONTINUE";
    case ZPOOL_STATUS_IO_FAILURE_MMP: return "IO_FAILURE_MMP";
    case ZPOOL_STATUS_BAD_LOG: return "BAD_LOG";
    case ZPOOL_STATUS_ERRATA: return "ERRATA";
    case ZPOOL_STATUS_UNSUP_FEAT_READ: return "UNSUP_FEAT_READ";
    case ZPOOL_STATUS_UNSUP_FEAT_WRITE: return "UNSUP_FEAT_WRITE";
    case ZPOOL_STATUS_FAULTED_DEV_R: return "FAULTED_DEV_R";
    case ZPOOL_STATUS_FAULTED_DEV_NR: return "FAULTED_DEV_NR";
    case ZPOOL_STATUS_VERSION_OLDER: return "VERSION_OLDER";
    case ZPOOL_STATUS_FEAT_DISABLED: return "FEAT_DISABLED";
    case ZPOOL_STATUS_RESILVERING: return "RESILVERING";
    case ZPOOL_STATUS_OFFLINE_DEV: return "OFFLINE_DEV";
    case ZPOOL_STATUS_REMOVED_DEV: return "REMOVED_DEV";
    case ZPOOL_STATUS_REBUILDING: return "REBUILDING";
    case ZPOOL_STATUS_REBUILD_SCRUB: return "REBUILD_SCRUB";
    case ZPOOL_STATUS_NON_NATIVE_ASHIFT: return "NON_NATIVE_ASHIFT";
    case ZPOOL_STATUS_COMPATIBILITY_ERR: return "COMPATIBILITY_ERR";
    case ZPOOL_STATUS_INCOMPATIBLE_FEAT: return "INCOMPATIBLE_FEAT";
    case ZPOOL_STATUS_OK: return "OK";
    }
    return "UNKNOWN";
}

// go_zpool_health stores the pool's health property (as `zpool list` shows it).
int go_zpool_health(zpool_handle_t *zhp, char *buf, size_t len) {
    return zpool_get_prop(zhp, ZPOOL_PROP_HEALTH, buf, len, NULL, B_FALSE);
}

// go_config_vdev_tree returns the root vdev nvlist of a pool config, or NULL.
nvlist_t *go_config_vdev_tree(nvlist_t *config) {
    nvlist_t *nvroot = NULL;
    if (nvlist_lookup_nvlist(config, ZPOOL_CONFIG_VDEV_TREE, &nvroot) != 0)
        return NULL;
    return nvroot;
}

// go_pool_errcount returns the number of persistent data errors of a pool.
uint64_t go_pool_errcount(nvlist_t *config) {
    uint64_t n = 0;
    (void) nvlist_lookup_uint64(config, ZPOOL_CONFIG_ERRCOUNT, &n);
    return n;
}

#define GO_VDEV_CHILDREN 0
#define GO_VDEV_SPARES   1
#define GO_VDEV_L2CACHE  2

// go_vdev_group stores the child array of the given group (GO_VDEV_*) of a
// vdev and returns its length (0 when absent).
int go_vdev_group(nvlist_t *nv, int group, nvlist_t ***child, uint_t *n) {
    const char *key = group == GO_VDEV_SPARES ? ZPOOL_CONFIG_SPARES :
        group == GO_VDEV_L2CACHE ? ZPOOL_CONFIG_L2CACHE : ZPOOL_CONFIG_CHILDREN;
    if (nvlist_lookup_nvlist_array(nv, key, child, n) != 0) {
        *child = NULL;
        *n = 0;
    }
    return (int)*n;
}

// go_vdev_get_info extracts the type, path, display state and statistics of a
// vdev. nums receives guid, alloc, space, dspace, read, write and checksum
// errors. Spares report AVAIL/INUSE like `zpool status`.
void go_vdev_get_info(nvlist_t *nv, int is_spare, const char **type,
    const char **path, const char **state, uint64_t *nums)
{
    vdev_stat_t *vs;
    uint_t c;

    *type = "";
    *path = "";
    *state = "UNKNOWN";
    memset(nums, 0, 7 * sizeof (uint64_t));
    (void) nvlist_lookup_string(nv, ZPOOL_CONFIG_TYPE, type);
    (void) nvlist_lookup_string(nv, ZPOOL_CONFIG_PATH, path);
    (void) nvlist_lookup_uint64(nv, ZPOOL_CONFIG_GUID, &nums[0]);
    if (nvlist_lookup_uint64_array(nv, ZPOOL_CONFIG_VDEV_STATS,
        (uint64_t **)&vs, &c) != 0)
        return;

    *state = zpool_state_to_name(vs->vs_state, vs->vs_aux);
    if (is_spare) {
        if (vs->vs_aux == VDEV_AUX_SPARED)
            *state = "INUSE";
        else if (vs->vs_state == VDEV_STATE_HEALTHY)
            *state = "AVAIL";
    }
    nums[1] = vs->vs_alloc;
    nums[2] = vs->vs_space;
    nums[3] = vs->vs_dspace;
    nums[4] = vs->vs_read_errors;
    nums[5] = vs->vs_write_errors;
    nums[6] = vs->vs_checksum_errors;
}

// go_pool_scan_stats copies the scan statistics of the root vdev into out.
// Returns -1 when the pool has no scan statistics.
int go_pool_scan_stats(nvlist_t *nvroot, pool_scan_stat_t *out) {
    uint64_t *arr;
    uint_t c;
    size_t n;

    memset(out, 0, sizeof (*out));
    if (nvlist_lookup_uint64_array(nvroot, ZPOOL_CONFIG_SCAN_STATS, &arr, &c) != 0)
        return -1;
    n = c * sizeof (uint64_t);
    if (n > sizeof (*out))
        n = sizeof (*out);
    memcpy(out, arr, n);
    return 0;
}

// Pool export operations
int go_zpool_export(zpool_handle_t* zhp, boolean_t force, const char* message) {
    return zpool_export(zhp, force, message);
}

int go_zpool_export_force(zpool_handle_t* zhp) {
    return zpool_export(zhp, B_TRUE, "Exported via Go ZFS library");
}

// Pool creation helpers
int go_zpool_create(libzfs_handle_t* hdl, const char* poolname, nvlist_t* nvroot,
                    nvlist_t* props, nvlist_t* fsprops) {
    return zpool_create(hdl, poolname, nvroot, props, fsprops);
}

int go_zpool_destroy(zpool_handle_t* zhp, const char* message) {
    return zpool_destroy(zhp, message);
}

// Dataset creation and destruction
int go_zfs_create(libzfs_handle_t* hdl, const char* path, zfs_type_t type, nvlist_t* props) {
    return zfs_create(hdl, path, type, props);
}

int go_zfs_destroy(zfs_handle_t* zhp, boolean_t defer_destroy) {
    return zfs_destroy(zhp, defer_destroy);
}

int go_zfs_destroy_recursive(zfs_handle_t* zhp, const char* snapname) {
    return zfs_destroy(zhp, B_FALSE);
}

// Dataset type helpers
int go_get_zfs_type_filesystem() { return ZFS_TYPE_FILESYSTEM; }
int go_get_zfs_type_volume() { return ZFS_TYPE_VOLUME; }
int go_get_zfs_type_snapshot() { return ZFS_TYPE_SNAPSHOT; }

// Property setting helpers
int go_zfs_prop_set(zfs_handle_t* zhp, const char* propname, const char* propval) {
    return zfs_prop_set(zhp, propname, propval);
}

// Nvlist creation and manipulation for properties
nvlist_t* go_nvlist_alloc() {
    nvlist_t* nvl = NULL;
    if (nvlist_alloc(&nvl, NV_UNIQUE_NAME, 0) != 0) {
        return NULL;
    }
    return nvl;
}

void go_nvlist_free(nvlist_t* nvl) {
    if (nvl != NULL) {
        nvlist_free(nvl);
    }
}

int go_nvlist_add_string(nvlist_t* nvl, const char* name, const char* val) {
    return nvlist_add_string(nvl, name, val);
}

int go_nvlist_add_uint64(nvlist_t* nvl, const char* name, uint64_t val) {
    return nvlist_add_uint64(nvl, name, val);
}

int go_nvlist_add_boolean_value(nvlist_t* nvl, const char* name, boolean_t val) {
    return nvlist_add_boolean_value(nvl, name, val);
}

int go_nvlist_add_nvlist(nvlist_t* nvl, const char* name, nvlist_t* val) {
    return nvlist_add_nvlist(nvl, name, val);
}

int go_nvlist_add_nvlist_array(nvlist_t* nvl, const char* name, nvlist_t** val, uint_t nelem) {
    return nvlist_add_nvlist_array(nvl, name, (const nvlist_t* const*)val, nelem);
}

// Helper to create a simple vdev nvlist for a single device
nvlist_t* go_create_vdev_nvlist(const char* type, const char* path) {
    nvlist_t* vdev = NULL;

    if (nvlist_alloc(&vdev, NV_UNIQUE_NAME, 0) != 0) {
        return NULL;
    }

    if (nvlist_add_string(vdev, "type", type) != 0) {
        nvlist_free(vdev);
        return NULL;
    }

    if (path && nvlist_add_string(vdev, "path", path) != 0) {
        nvlist_free(vdev);
        return NULL;
    }

    return vdev;
}

// Snapshot operations
int go_zfs_snapshot(libzfs_handle_t* hdl, const char* path, boolean_t recursive, nvlist_t* props) {
    return zfs_snapshot(hdl, path, recursive, props);
}

int go_zfs_rollback(zfs_handle_t* zhp, zfs_handle_t* snap, boolean_t force) {
    return zfs_rollback(zhp, snap, force);
}

// Clone operations
int go_zfs_clone(libzfs_handle_t* hdl, const char* snapname, const char* clonename, nvlist_t* props) {
    // Open the snapshot first
    zfs_handle_t* snap_zhp = zfs_open(hdl, snapname, ZFS_TYPE_SNAPSHOT);
    if (snap_zhp == NULL) {
        return -1;
    }

    // Create the clone
    int ret = zfs_clone(snap_zhp, clonename, props);
    zfs_close(snap_zhp);
    return ret;
}

int go_zfs_promote(zfs_handle_t* zhp) {
    return zfs_promote(zhp);
}

// Clone relationship queries
const char* go_zfs_get_origin(zfs_handle_t* zhp) {
    char origin[ZFS_MAX_DATASET_NAME_LEN];
    if (zfs_prop_get(zhp, ZFS_PROP_ORIGIN, origin, sizeof(origin), NULL, NULL, 0, B_FALSE) == 0) {
        // Return a copy that won't be freed
        static char origin_copy[ZFS_MAX_DATASET_NAME_LEN];
        strcpy(origin_copy, origin);
        return origin_copy;
    }
    return NULL;
}

nvlist_t* go_zfs_get_clones_nvlist(zfs_handle_t* zhp) {
    return zfs_get_clones_nvl(zhp);
}

// Helper to check if dataset is a clone
boolean_t go_zfs_is_clone(zfs_handle_t* zhp) {
    char origin[ZFS_MAX_DATASET_NAME_LEN];
    return (zfs_prop_get(zhp, ZFS_PROP_ORIGIN, origin, sizeof(origin), NULL, NULL, 0, B_FALSE) == 0 &&
            strlen(origin) > 0);
}

// Helper to get clone count for a snapshot
int go_zfs_get_clone_count(zfs_handle_t* zhp) {
    nvlist_t* clones = NULL;
    int count = 0;

    clones = zfs_get_clones_nvl(zhp);
    if (clones != NULL) {
        count = nvlist_next_nvpair(clones, NULL) != NULL ? 1 : 0;
        // Count all nvpairs to get actual clone count
        nvpair_t* pair = NULL;
        count = 0;
        while ((pair = nvlist_next_nvpair(clones, pair)) != NULL) {
            count++;
        }
        nvlist_free(clones);
    }

    return count;
}

// Advanced vdev management operations
int go_zpool_add(zpool_handle_t* zhp, nvlist_t* nvroot) {
    return zpool_add(zhp, nvroot, B_FALSE);
}

int go_zpool_attach(zpool_handle_t* zhp, const char* old_disk, const char* new_disk, nvlist_t* props, int replacing) {
    return zpool_vdev_attach(zhp, old_disk, new_disk, props, replacing, B_FALSE);
}

int go_zpool_detach(zpool_handle_t* zhp, const char* path) {
    return zpool_vdev_detach(zhp, path);
}

int go_zpool_replace(zpool_handle_t* zhp, const char* old_disk, const char* new_disk, nvlist_t* props) {
    return zpool_vdev_attach(zhp, old_disk, new_disk, props, B_TRUE, B_FALSE);
}

int go_zpool_remove(zpool_handle_t* zhp, const char* path) {
    return zpool_vdev_remove(zhp, path);
}

int go_zpool_online(zpool_handle_t* zhp, const char* path, int flags, vdev_state_t* newstate) {
    return zpool_vdev_online(zhp, path, flags, newstate);
}

int go_zpool_offline(zpool_handle_t* zhp, const char* path, boolean_t istmp) {
    return zpool_vdev_offline(zhp, path, istmp);
}

int go_zpool_clear(zpool_handle_t* zhp, const char* path, nvlist_t* rewind_policy) {
    return zpool_clear(zhp, path, rewind_policy);
}

// Vdev tree manipulation helpers
nvlist_t* go_create_mirror_vdev(nvlist_t** children, uint_t child_count) {
    nvlist_t* mirror = NULL;

    if (nvlist_alloc(&mirror, NV_UNIQUE_NAME, 0) != 0) {
        return NULL;
    }

    if (nvlist_add_string(mirror, "type", "mirror") != 0) {
        nvlist_free(mirror);
        return NULL;
    }

    if (nvlist_add_nvlist_array(mirror, "children", (const nvlist_t* const*)children, child_count) != 0) {
        nvlist_free(mirror);
        return NULL;
    }

    return mirror;
}

nvlist_t* go_create_raidz_vdev(nvlist_t** children, uint_t child_count, int parity) {
    nvlist_t* raidz = NULL;
    const char* type;

    switch (parity) {
        case 1: type = "raidz"; break;
        case 2: type = "raidz2"; break;
        case 3: type = "raidz3"; break;
        default: return NULL; // Invalid parity level
    }

    if (nvlist_alloc(&raidz, NV_UNIQUE_NAME, 0) != 0) {
        return NULL;
    }

    if (nvlist_add_string(raidz, "type", type) != 0) {
        nvlist_free(raidz);
        return NULL;
    }

    if (nvlist_add_nvlist_array(raidz, "children", (const nvlist_t* const*)children, child_count) != 0) {
        nvlist_free(raidz);
        return NULL;
    }

    return raidz;
}

nvlist_t* go_create_stripe_vdev(nvlist_t** children, uint_t child_count) {
    nvlist_t* stripe = NULL;

    if (nvlist_alloc(&stripe, NV_UNIQUE_NAME, 0) != 0) {
        return NULL;
    }

    if (nvlist_add_string(stripe, "type", "root") != 0) {
        nvlist_free(stripe);
        return NULL;
    }

    if (nvlist_add_nvlist_array(stripe, "children", (const nvlist_t* const*)children, child_count) != 0) {
        nvlist_free(stripe);
        return NULL;
    }

    return stripe;
}

// Pool handle accessors
zpool_handle_t* go_zpool_open(libzfs_handle_t* hdl, const char* name) {
    return zpool_open(hdl, name);
}

void go_zpool_close(zpool_handle_t* zhp) {
    if (zhp != NULL) {
        zpool_close(zhp);
    }
}

// Vdev state constants - using available constants with fallbacks
int go_get_vdev_state_offline() { return VDEV_STATE_OFFLINE; }
int go_get_vdev_state_online() { return VDEV_STATE_HEALTHY; }
int go_get_vdev_state_degraded() { return VDEV_STATE_DEGRADED; }
int go_get_vdev_state_faulted() { return VDEV_STATE_FAULTED; }
int go_get_vdev_state_removed() { return VDEV_STATE_REMOVED; }
int go_get_vdev_state_unavail() { return POOL_STATE_UNAVAIL; }

// Online flags - define defaults if not available
#ifndef ZFS_ONLINE_CHECKREMOVE
#define ZFS_ONLINE_CHECKREMOVE    0x1
#endif
#ifndef ZFS_ONLINE_UNSPARE
#define ZFS_ONLINE_UNSPARE        0x2
#endif
#ifndef ZFS_ONLINE_FORCEFAULT
#define ZFS_ONLINE_FORCEFAULT     0x4
#endif
#ifndef ZFS_ONLINE_EXPAND
#define ZFS_ONLINE_EXPAND         0x8
#endif

int go_get_zfs_online_checkremove() { return ZFS_ONLINE_CHECKREMOVE; }
int go_get_zfs_online_unspare() { return ZFS_ONLINE_UNSPARE; }
int go_get_zfs_online_forcefault() { return ZFS_ONLINE_FORCEFAULT; }
int go_get_zfs_online_expand() { return ZFS_ONLINE_EXPAND; }

// OpenZFS 2.3 added a mount thread-count argument to zpool_enable_datasets.
// The headers carry no version macro; ZPOOL_CONFIG_RAIDZ_EXPANDING first
// appeared in the same release (2.3.0-rc1), so it marks the new signature.
// FreeBSD 14.x ships OpenZFS 2.2, FreeBSD 15.x ships 2.3+.
#ifdef ZPOOL_CONFIG_RAIDZ_EXPANDING
#define GO_ZPOOL_ENABLE_DATASETS(zhp) zpool_enable_datasets((zhp), NULL, 0, 1)
#else
#define GO_ZPOOL_ENABLE_DATASETS(zhp) zpool_enable_datasets((zhp), NULL, 0)
#endif

// Pool import. Return codes must stay in sync with importErr* in the driver.
#define GO_IMPORT_OK              0
#define GO_IMPORT_NOT_FOUND      -1  // no importable pool with that name
#define GO_IMPORT_AMBIGUOUS      -2  // several importable pools share the name
#define GO_IMPORT_SEARCH_FAILED  -3  // device scan failed, see errbuf
#define GO_IMPORT_FAILED         -4  // zpool_import_props failed, see libzfs error
#define GO_IMPORT_OPEN_FAILED    -5  // pool imported but could not be opened
#define GO_IMPORT_MOUNT_FAILED   -6  // pool imported but datasets not mounted
#define GO_IMPORT_NOMEM          -7

// go_zpool_search scans paths (or the default device directories when npaths
// is 0) for pools that are not currently imported, optionally restricted to
// poolname (may be NULL). It returns an nvlist of pool configs keyed by pool
// name, to be freed with nvlist_free, or NULL. On NULL, *err is non-zero and
// errbuf holds a description if the scan itself failed; *err is 0 when simply
// nothing was found. Destroyed pools are included; callers filter by
// ZPOOL_CONFIG_POOL_STATE.
nvlist_t *go_zpool_search(libzfs_handle_t *hdl, const char *poolname,
    char **paths, int npaths, char *errbuf, size_t errlen, int *err)
{
    libpc_handle_t lpch;
    importargs_t args;
    nvlist_t *pools;
    size_t ndefault = 0;

    *err = 0;
    errbuf[0] = '\0';
    memset(&lpch, 0, sizeof (lpch));
    lpch.lpc_ops = &libzfs_config_ops;
    lpch.lpc_lib_handle = hdl;

    memset(&args, 0, sizeof (args));
    if (npaths > 0) {
        args.path = paths;
        args.paths = npaths;
    } else {
        args.path = (char **)zpool_default_search_paths(&ndefault);
        args.paths = (int)ndefault;
    }
    args.poolname = poolname;

    pools = zpool_search_import(&lpch, &args);
    if (pools == NULL && lpch.lpc_error != 0) {
        *err = lpch.lpc_error;
        snprintf(errbuf, errlen, "%s", libpc_error_description(&lpch));
    }
    return pools;
}

// go_import_config_summary extracts the identity, state and health of a pool
// from an import config returned by go_zpool_search. Strings point into the
// config or static storage. Returns 0 on success, -1 if the config is malformed.
int go_import_config_summary(nvlist_t *config, const char **name, uint64_t *guid,
    uint64_t *state, const char **state_name, const char **health)
{
    nvlist_t *nvroot;
    vdev_stat_t *vs;
    uint_t vsc;

    if (nvlist_lookup_string(config, ZPOOL_CONFIG_POOL_NAME, name) != 0 ||
        nvlist_lookup_uint64(config, ZPOOL_CONFIG_POOL_GUID, guid) != 0 ||
        nvlist_lookup_uint64(config, ZPOOL_CONFIG_POOL_STATE, state) != 0 ||
        nvlist_lookup_nvlist(config, ZPOOL_CONFIG_VDEV_TREE, &nvroot) != 0 ||
        nvlist_lookup_uint64_array(nvroot, ZPOOL_CONFIG_VDEV_STATS,
        (uint64_t **)&vs, &vsc) != 0)
        return -1;

    *state_name = zpool_pool_state_to_name((pool_state_t)*state);
    *health = zpool_state_to_name(vs->vs_state, vs->vs_aux);
    return 0;
}

static int collect_leaf_paths(nvlist_t *nv, const char **out, int max, int n)
{
    static const char *const groups[] = {
        ZPOOL_CONFIG_CHILDREN, ZPOOL_CONFIG_L2CACHE, ZPOOL_CONFIG_SPARES
    };
    nvlist_t **child;
    uint_t nchild, c;
    const char *path;
    int found_children = 0;

    for (size_t g = 0; g < sizeof (groups) / sizeof (groups[0]); g++) {
        if (nvlist_lookup_nvlist_array(nv, groups[g], &child, &nchild) != 0)
            continue;
        found_children = 1;
        for (c = 0; c < nchild; c++)
            n = collect_leaf_paths(child[c], out, max, n);
    }
    if (!found_children &&
        nvlist_lookup_string(nv, ZPOOL_CONFIG_PATH, &path) == 0) {
        if (n < max)
            out[n] = path;
        n++;
    }
    return n;
}

// go_import_config_device_paths stores up to max leaf device paths of the
// pool's vdev tree (data, cache and spare devices) in out and returns the total
// number of leaf devices, which may exceed max.
int go_import_config_device_paths(nvlist_t *config, const char **out, int max)
{
    nvlist_t *nvroot;

    if (nvlist_lookup_nvlist(config, ZPOOL_CONFIG_VDEV_TREE, &nvroot) != 0)
        return 0;
    return collect_leaf_paths(nvroot, out, max, 0);
}

// go_zpool_import finds the exported (or, with destroyed, destroyed) pool
// named poolname by scanning paths (or the default device directories when
// npaths is 0), imports it, optionally under newname and/or altroot, and mounts
// its datasets. It mirrors the do_import() path of zpool(8).
int go_zpool_import(libzfs_handle_t *hdl, const char *poolname,
    char **paths, int npaths, const char *newname, const char *altroot,
    int flags, int destroyed, char *errbuf, size_t errlen)
{
    nvlist_t *pools, *config = NULL, *props = NULL;
    nvpair_t *elem = NULL;
    zpool_handle_t *zhp;
    int matches = 0, ret = GO_IMPORT_OK, search_err = 0;

    pools = go_zpool_search(hdl, poolname, paths, npaths, errbuf, errlen, &search_err);
    if (pools == NULL)
        return search_err ? GO_IMPORT_SEARCH_FAILED : GO_IMPORT_NOT_FOUND;

    while ((elem = nvlist_next_nvpair(pools, elem)) != NULL) {
        nvlist_t *c;
        uint64_t state;
        if (nvpair_value_nvlist(elem, &c) != 0 ||
            nvlist_lookup_uint64(c, ZPOOL_CONFIG_POOL_STATE, &state) != 0)
            continue;
        if ((state == POOL_STATE_DESTROYED) != (destroyed != 0))
            continue;
        config = c;
        matches++;
    }
    if (matches == 0) {
        ret = GO_IMPORT_NOT_FOUND;
        goto out;
    }
    if (matches > 1) {
        ret = GO_IMPORT_AMBIGUOUS;
        goto out;
    }

    if (altroot != NULL) {
        if (nvlist_alloc(&props, NV_UNIQUE_NAME, 0) != 0 ||
            nvlist_add_string(props,
            zpool_prop_to_name(ZPOOL_PROP_ALTROOT), altroot) != 0 ||
            nvlist_add_string(props,
            zpool_prop_to_name(ZPOOL_PROP_CACHEFILE), "none") != 0) {
            ret = GO_IMPORT_NOMEM;
            goto out;
        }
    }

    if (zpool_import_props(hdl, config, newname, props, flags) != 0) {
        ret = GO_IMPORT_FAILED;
        goto out;
    }

    zhp = zpool_open_canfail(hdl, newname != NULL ? newname : poolname);
    if (zhp == NULL) {
        ret = GO_IMPORT_OPEN_FAILED;
        goto out;
    }
    if (zpool_get_state(zhp) != POOL_STATE_UNAVAIL &&
        GO_ZPOOL_ENABLE_DATASETS(zhp) != 0)
        ret = GO_IMPORT_MOUNT_FAILED;
    zpool_close(zhp);

out:
    nvlist_free(props);
    nvlist_free(pools);
    return ret;
}
*/
import "C"

// This file exists solely to hold the C code for libzfs integration.
// All Go code that uses these functions should be in the driver package.
