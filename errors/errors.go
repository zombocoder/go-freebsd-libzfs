//go:build freebsd

package errors

import (
	"errors"
	"fmt"
)

// Error codes for common ZFS operations
const (
	ErrCodeNotFound           = "ENOENT"
	ErrCodePermission         = "EPERM"
	ErrCodeExists             = "EEXIST"
	ErrCodeBusy               = "EBUSY"
	ErrCodeInval              = "EINVAL"
	ErrCodeNoSpace            = "ENOSPC"
	ErrCodeIO                 = "EIO"
	ErrCodeChecksum           = "ECKSUM"
	ErrCodeFault              = "EFAULT"
	ErrCodeNotSupported       = "ENOTSUP"
	ErrCodeNameTooLong        = "ENAMETOOLONG"
	ErrCodeQuotaExceeded      = "EQUOT"
	ErrCodeNotEmpty           = "ENOTEMPTY"
	ErrCodeCrossDevice        = "EXDEV"
	ErrCodeResourceBusy       = "EBUSY"
	ErrCodeFeatureUnsupported = "ENOTSUP"
)

// ZfsError represents a structured ZFS operation error
type ZfsError struct {
	Op       string // The operation that failed (e.g., "list_pools", "get_property")
	Resource string // The resource involved (e.g., pool name, dataset name)
	Code     string // Error code (ENOENT, EPERM, etc.)
	Errno    int    // Original errno value
	Detail   string // Detailed error message
	Cause    error  // Underlying cause error
}

// Error implements the error interface
func (e *ZfsError) Error() string {
	if e.Resource != "" {
		return fmt.Sprintf("zfs %s %s: %s (%s)", e.Op, e.Resource, e.Detail, e.Code)
	}
	return fmt.Sprintf("zfs %s: %s (%s)", e.Op, e.Detail, e.Code)
}

// Unwrap returns the underlying cause error
func (e *ZfsError) Unwrap() error {
	return e.Cause
}

// Is implements error comparison for errors.Is()
func (e *ZfsError) Is(target error) bool {
	t, ok := target.(*ZfsError)
	if !ok {
		return false
	}
	return e.Code == t.Code && e.Op == t.Op
}

// AsZfsError extracts a ZfsError from an error chain
func AsZfsError(err error) (*ZfsError, bool) {
	var zfsErr *ZfsError
	if errors.As(err, &zfsErr) {
		return zfsErr, true
	}
	return nil, false
}

// IsZfsError checks if an error is a ZfsError
func IsZfsError(err error) bool {
	_, ok := AsZfsError(err)
	return ok
}

// Predefined error variables for common conditions
var (
	// ErrDatasetNotFound indicates a dataset was not found
	ErrDatasetNotFound = &ZfsError{
		Op:     "get_dataset",
		Code:   ErrCodeNotFound,
		Detail: "dataset not found",
	}

	// ErrPoolNotFound indicates a pool was not found
	ErrPoolNotFound = &ZfsError{
		Op:     "get_pool",
		Code:   ErrCodeNotFound,
		Detail: "pool not found",
	}

	// ErrDatasetExists indicates a dataset already exists
	ErrDatasetExists = &ZfsError{
		Op:     "create_dataset",
		Code:   ErrCodeExists,
		Detail: "dataset already exists",
	}

	// ErrPoolExists indicates a pool already exists
	ErrPoolExists = &ZfsError{
		Op:     "create_pool",
		Code:   ErrCodeExists,
		Detail: "pool already exists",
	}

	// ErrPermissionDenied indicates insufficient permissions
	ErrPermissionDenied = &ZfsError{
		Op:     "zfs_operation",
		Code:   ErrCodePermission,
		Detail: "permission denied",
	}

	// ErrPoolBusy indicates a pool is busy
	ErrPoolBusy = &ZfsError{
		Op:     "pool_operation",
		Code:   ErrCodeBusy,
		Detail: "pool is busy",
	}

	// ErrDatasetBusy indicates a dataset is busy
	ErrDatasetBusy = &ZfsError{
		Op:     "dataset_operation",
		Code:   ErrCodeBusy,
		Detail: "dataset is busy",
	}

	// ErrInvalidArgument indicates an invalid argument
	ErrInvalidArgument = &ZfsError{
		Op:     "validate_input",
		Code:   ErrCodeInval,
		Detail: "invalid argument",
	}

	// ErrNoSpace indicates insufficient space
	ErrNoSpace = &ZfsError{
		Op:     "space_check",
		Code:   ErrCodeNoSpace,
		Detail: "no space left on device",
	}

	// ErrIOError indicates an I/O error
	ErrIOError = &ZfsError{
		Op:     "io_operation",
		Code:   ErrCodeIO,
		Detail: "I/O error",
	}

	// ErrChecksumMismatch indicates a checksum error
	ErrChecksumMismatch = &ZfsError{
		Op:     "checksum_verify",
		Code:   ErrCodeChecksum,
		Detail: "checksum mismatch",
	}

	// ErrNotSupported indicates an unsupported operation
	ErrNotSupported = &ZfsError{
		Op:     "feature_check",
		Code:   ErrCodeNotSupported,
		Detail: "operation not supported",
	}

	// ErrFeatureUnsupported indicates an unsupported feature
	ErrFeatureUnsupported = &ZfsError{
		Op:     "feature_check",
		Code:   ErrCodeFeatureUnsupported,
		Detail: "feature not supported",
	}

	// ErrQuotaExceeded indicates a quota was exceeded
	ErrQuotaExceeded = &ZfsError{
		Op:     "quota_check",
		Code:   ErrCodeQuotaExceeded,
		Detail: "quota exceeded",
	}

	// ErrNameTooLong indicates a name is too long
	ErrNameTooLong = &ZfsError{
		Op:     "name_validation",
		Code:   ErrCodeNameTooLong,
		Detail: "name too long",
	}

	// ErrNotEmpty indicates a dataset is not empty
	ErrNotEmpty = &ZfsError{
		Op:     "empty_check",
		Code:   ErrCodeNotEmpty,
		Detail: "directory not empty",
	}

	// ErrCrossDevice indicates a cross-device operation
	ErrCrossDevice = &ZfsError{
		Op:     "device_check",
		Code:   ErrCodeCrossDevice,
		Detail: "cross-device link",
	}
)

// NewZfsError creates a new ZfsError with the given parameters
func NewZfsError(op, resource, code string, errno int, detail string, cause error) *ZfsError {
	return &ZfsError{
		Op:       op,
		Resource: resource,
		Code:     code,
		Errno:    errno,
		Detail:   detail,
		Cause:    cause,
	}
}

// WrapZfsError wraps an existing error as a ZfsError
func WrapZfsError(op, resource string, cause error) *ZfsError {
	if zfsErr, ok := AsZfsError(cause); ok {
		// If it's already a ZfsError, update the operation context
		return &ZfsError{
			Op:       op,
			Resource: resource,
			Code:     zfsErr.Code,
			Errno:    zfsErr.Errno,
			Detail:   zfsErr.Detail,
			Cause:    zfsErr.Cause,
		}
	}

	// For other error types, wrap with a generic code
	return &ZfsError{
		Op:       op,
		Resource: resource,
		Code:     "UNKNOWN",
		Errno:    -1,
		Detail:   cause.Error(),
		Cause:    cause,
	}
}

// IsDatasetNotFound checks if an error indicates a dataset was not found
func IsDatasetNotFound(err error) bool {
	if zfsErr, ok := AsZfsError(err); ok {
		return zfsErr.Code == ErrCodeNotFound &&
			(zfsErr.Op == "get_dataset" || zfsErr.Op == "list_datasets")
	}
	return false
}

// IsPoolNotFound checks if an error indicates a pool was not found
func IsPoolNotFound(err error) bool {
	if zfsErr, ok := AsZfsError(err); ok {
		return zfsErr.Code == ErrCodeNotFound &&
			(zfsErr.Op == "get_pool" || zfsErr.Op == "list_pools")
	}
	return false
}

// IsPermissionDenied checks if an error indicates permission was denied
func IsPermissionDenied(err error) bool {
	if zfsErr, ok := AsZfsError(err); ok {
		return zfsErr.Code == ErrCodePermission
	}
	return false
}

// IsExists checks if an error indicates a resource already exists
func IsExists(err error) bool {
	if zfsErr, ok := AsZfsError(err); ok {
		return zfsErr.Code == ErrCodeExists
	}
	return false
}

// IsBusy checks if an error indicates a resource is busy
func IsBusy(err error) bool {
	if zfsErr, ok := AsZfsError(err); ok {
		return zfsErr.Code == ErrCodeBusy
	}
	return false
}

// IsNoSpace checks if an error indicates insufficient space
func IsNoSpace(err error) bool {
	if zfsErr, ok := AsZfsError(err); ok {
		return zfsErr.Code == ErrCodeNoSpace
	}
	return false
}

// IsNotSupported checks if an error indicates an unsupported operation
func IsNotSupported(err error) bool {
	if zfsErr, ok := AsZfsError(err); ok {
		return zfsErr.Code == ErrCodeNotSupported || zfsErr.Code == ErrCodeFeatureUnsupported
	}
	return false
}

// MapErrno maps a Unix errno to a ZFS error code
func MapErrno(errno int) string {
	switch errno {
	case 2: // ENOENT
		return ErrCodeNotFound
	case 1: // EPERM
		return ErrCodePermission
	case 17: // EEXIST
		return ErrCodeExists
	case 16: // EBUSY
		return ErrCodeBusy
	case 22: // EINVAL
		return ErrCodeInval
	case 28: // ENOSPC
		return ErrCodeNoSpace
	case 5: // EIO
		return ErrCodeIO
	case 45: // ENOTSUP
		return ErrCodeNotSupported
	case 63: // ENAMETOOLONG
		return ErrCodeNameTooLong
	case 69: // EQUOT
		return ErrCodeQuotaExceeded
	case 66: // ENOTEMPTY
		return ErrCodeNotEmpty
	case 18: // EXDEV
		return ErrCodeCrossDevice
	default:
		return fmt.Sprintf("ERRNO_%d", errno)
	}
}
