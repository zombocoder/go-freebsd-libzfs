//go:build freebsd

package errors

import (
	"errors"
	"testing"
)

func TestZfsError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ZfsError
		expected string
	}{
		{
			name: "with resource",
			err: &ZfsError{
				Op:       "get_dataset",
				Resource: "tank/data",
				Code:     ErrCodeNotFound,
				Detail:   "dataset not found",
			},
			expected: "zfs get_dataset tank/data: dataset not found (ENOENT)",
		},
		{
			name: "without resource",
			err: &ZfsError{
				Op:     "list_pools",
				Code:   ErrCodePermission,
				Detail: "permission denied",
			},
			expected: "zfs list_pools: permission denied (EPERM)",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.err.Error()
			if result != test.expected {
				t.Errorf("Error() = %q, want %q", result, test.expected)
			}
		})
	}
}

func TestZfsError_Is(t *testing.T) {
	err1 := &ZfsError{
		Op:   "get_dataset",
		Code: ErrCodeNotFound,
	}

	err2 := &ZfsError{
		Op:   "get_dataset",
		Code: ErrCodeNotFound,
	}

	err3 := &ZfsError{
		Op:   "get_pool",
		Code: ErrCodeNotFound,
	}

	if !err1.Is(err2) {
		t.Error("err1 should be equal to err2")
	}

	if err1.Is(err3) {
		t.Error("err1 should not be equal to err3 (different operation)")
	}
}

func TestAsZfsError(t *testing.T) {
	zfsErr := &ZfsError{
		Op:     "test",
		Code:   ErrCodeNotFound,
		Detail: "test error",
	}

	// Test with ZfsError
	result, ok := AsZfsError(zfsErr)
	if !ok || result != zfsErr {
		t.Error("AsZfsError should return the ZfsError")
	}

	// Test with non-ZfsError
	regularErr := errors.New("regular error")
	result, ok = AsZfsError(regularErr)
	if ok || result != nil {
		t.Error("AsZfsError should return false for non-ZfsError")
	}
}

func TestIsZfsError(t *testing.T) {
	zfsErr := &ZfsError{Op: "test", Code: ErrCodeNotFound}
	regularErr := errors.New("regular error")

	if !IsZfsError(zfsErr) {
		t.Error("IsZfsError should return true for ZfsError")
	}

	if IsZfsError(regularErr) {
		t.Error("IsZfsError should return false for regular error")
	}
}

func TestWrapZfsError(t *testing.T) {
	originalErr := &ZfsError{
		Op:     "original_op",
		Code:   ErrCodeNotFound,
		Detail: "original detail",
		Errno:  2,
	}

	// Test wrapping a ZfsError
	wrapped := WrapZfsError("new_op", "resource", originalErr)
	if wrapped.Op != "new_op" {
		t.Errorf("Op = %q, want %q", wrapped.Op, "new_op")
	}
	if wrapped.Resource != "resource" {
		t.Errorf("Resource = %q, want %q", wrapped.Resource, "resource")
	}
	if wrapped.Code != ErrCodeNotFound {
		t.Errorf("Code = %q, want %q", wrapped.Code, ErrCodeNotFound)
	}

	// Test wrapping a regular error
	regularErr := errors.New("test error")
	wrapped = WrapZfsError("op", "res", regularErr)
	if wrapped.Code != "UNKNOWN" {
		t.Errorf("Code = %q, want %q", wrapped.Code, "UNKNOWN")
	}
	if wrapped.Detail != "test error" {
		t.Errorf("Detail = %q, want %q", wrapped.Detail, "test error")
	}
}

func TestErrorPredicates(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		predicates map[string]func(error) bool
		expected   map[string]bool
	}{
		{
			name: "dataset not found",
			err: &ZfsError{
				Op:   "get_dataset",
				Code: ErrCodeNotFound,
			},
			predicates: map[string]func(error) bool{
				"IsDatasetNotFound":  IsDatasetNotFound,
				"IsPoolNotFound":     IsPoolNotFound,
				"IsPermissionDenied": IsPermissionDenied,
			},
			expected: map[string]bool{
				"IsDatasetNotFound":  true,
				"IsPoolNotFound":     false,
				"IsPermissionDenied": false,
			},
		},
		{
			name: "permission denied",
			err: &ZfsError{
				Op:   "create_dataset",
				Code: ErrCodePermission,
			},
			predicates: map[string]func(error) bool{
				"IsPermissionDenied": IsPermissionDenied,
				"IsDatasetNotFound":  IsDatasetNotFound,
			},
			expected: map[string]bool{
				"IsPermissionDenied": true,
				"IsDatasetNotFound":  false,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for name, predicate := range test.predicates {
				result := predicate(test.err)
				expected := test.expected[name]
				if result != expected {
					t.Errorf("%s() = %v, want %v", name, result, expected)
				}
			}
		})
	}
}

func TestMapErrno(t *testing.T) {
	tests := []struct {
		errno    int
		expected string
	}{
		{2, ErrCodeNotFound},      // ENOENT
		{1, ErrCodePermission},    // EPERM
		{17, ErrCodeExists},       // EEXIST
		{16, ErrCodeBusy},         // EBUSY
		{22, ErrCodeInval},        // EINVAL
		{28, ErrCodeNoSpace},      // ENOSPC
		{5, ErrCodeIO},            // EIO
		{45, ErrCodeNotSupported}, // ENOTSUP
		{999, "ERRNO_999"},        // Unknown errno
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			result := MapErrno(test.errno)
			if result != test.expected {
				t.Errorf("MapErrno(%d) = %q, want %q", test.errno, result, test.expected)
			}
		})
	}
}

func TestNewZfsError(t *testing.T) {
	cause := errors.New("underlying error")
	err := NewZfsError("test_op", "test_resource", ErrCodeNotFound, 2, "test detail", cause)

	if err.Op != "test_op" {
		t.Errorf("Op = %q, want %q", err.Op, "test_op")
	}
	if err.Resource != "test_resource" {
		t.Errorf("Resource = %q, want %q", err.Resource, "test_resource")
	}
	if err.Code != ErrCodeNotFound {
		t.Errorf("Code = %q, want %q", err.Code, ErrCodeNotFound)
	}
	if err.Errno != 2 {
		t.Errorf("Errno = %d, want %d", err.Errno, 2)
	}
	if err.Detail != "test detail" {
		t.Errorf("Detail = %q, want %q", err.Detail, "test detail")
	}
	if !errors.Is(err, cause) {
		t.Error("Unwrap should return the cause error")
	}
}
