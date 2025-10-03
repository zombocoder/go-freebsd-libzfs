//go:build freebsd

package driver

import (
	"testing"
)

func TestDatasetType_String(t *testing.T) {
	tests := []struct {
		dtype    DatasetType
		expected string
	}{
		{DatasetFilesystem, "filesystem"},
		{DatasetVolume, "volume"},
		{DatasetSnapshot, "snapshot"},
		{DatasetBookmark, "bookmark"},
		{DatasetType(999), "unknown"},
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			result := test.dtype.String()
			if result != test.expected {
				t.Errorf("DatasetType(%d).String() = %q, want %q", test.dtype, result, test.expected)
			}
		})
	}
}

func TestPropSource_String(t *testing.T) {
	tests := []struct {
		source   PropSource
		expected string
	}{
		{PropSourceLocal, "local"},
		{PropSourceInherited, "inherited"},
		{PropSourceDefault, "default"},
		{PropSourceTemporary, "temporary"},
		{PropSourceReceived, "received"},
		{PropSource(999), "unknown"},
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			result := test.source.String()
			if result != test.expected {
				t.Errorf("PropSource(%d).String() = %q, want %q", test.source, result, test.expected)
			}
		})
	}
}

func TestPropertyConstants(t *testing.T) {
	// Test that property name constants are not empty
	propertyNames := []string{
		PropNameMountpoint,
		PropNameCompression,
		PropNameUsed,
		PropNameAvail,
		PropNameRefer,
		PropNameQuota,
		PropNameReservation,
		PropNameRecordsize,
		PropNameAtime,
		PropNameDevices,
		PropNameExec,
		PropNameReadonly,
		PropNameSetuid,
		PropNameZoned,
		PropNameSnapdir,
		PropNameAclmode,
		PropNameCanmount,
		PropNameXattr,
		PropNameCopies,
		PropNameVersion,
		PropNameUtf8only,
		PropNameNormalize,
		PropNameCase,
		PropNameVscan,
		PropNameNbmand,
		PropNameSharenfs,
		PropNameSharesmb,
		PropNameRefquota,
		PropNameRefreserv,
		PropNameGuid,
		PropNamePrimcache,
		PropNameSeccache,
		PropNameUsedsnap,
		PropNameUsedds,
		PropNameUsedchild,
		PropNameUsedrefreserv,
		PropNameDefer,
		PropNameUserrefs,
		PropNameLogbias,
		PropNameUnique,
		PropNameWritten,
		PropNameClones,
		PropNameLogicalused,
		PropNameLogicalavail,
		PropNameSync,
		PropNameDnodesize,
		PropNameRefcomprat,
		PropNameEncryption,
		PropNameKeylocation,
		PropNameKeyformat,
		PropNamePbkdf2iters,
		PropNameEncroot,
		PropNameKeystatus,
	}

	for _, propName := range propertyNames {
		if propName == "" {
			t.Errorf("Property name constant is empty")
		}
	}
}

func TestPropertyInfo(t *testing.T) {
	prop := PropertyInfo{
		Name:     "test",
		Value:    "value",
		Source:   PropSourceLocal,
		Received: false,
	}

	if prop.Name != "test" {
		t.Errorf("Name = %q, want %q", prop.Name, "test")
	}
	if prop.Value != "value" {
		t.Errorf("Value = %v, want %v", prop.Value, "value")
	}
	if prop.Source != PropSourceLocal {
		t.Errorf("Source = %v, want %v", prop.Source, PropSourceLocal)
	}
	if prop.Received {
		t.Errorf("Received = %v, want %v", prop.Received, false)
	}
}

func TestPoolInfo(t *testing.T) {
	pool := PoolInfo{
		Name:   "testpool",
		GUID:   12345,
		Health: "ONLINE",
		State:  "ACTIVE",
	}

	if pool.Name != "testpool" {
		t.Errorf("Name = %q, want %q", pool.Name, "testpool")
	}
	if pool.GUID != 12345 {
		t.Errorf("GUID = %d, want %d", pool.GUID, 12345)
	}
	if pool.Health != "ONLINE" {
		t.Errorf("Health = %q, want %q", pool.Health, "ONLINE")
	}
	if pool.State != "ACTIVE" {
		t.Errorf("State = %q, want %q", pool.State, "ACTIVE")
	}
}

func TestDatasetInfo(t *testing.T) {
	dataset := DatasetInfo{
		Name: "testpool/data",
		Type: DatasetFilesystem,
		GUID: 67890,
	}

	if dataset.Name != "testpool/data" {
		t.Errorf("Name = %q, want %q", dataset.Name, "testpool/data")
	}
	if dataset.Type != DatasetFilesystem {
		t.Errorf("Type = %v, want %v", dataset.Type, DatasetFilesystem)
	}
	if dataset.GUID != 67890 {
		t.Errorf("GUID = %d, want %d", dataset.GUID, 67890)
	}
}
